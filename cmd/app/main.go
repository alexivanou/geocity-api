package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexivanou/geocity-api/internal/api"
	"github.com/alexivanou/geocity-api/internal/config"
	"github.com/alexivanou/geocity-api/internal/database"
	"github.com/alexivanou/geocity-api/internal/model"
	"github.com/alexivanou/geocity-api/internal/repository"
	"github.com/alexivanou/geocity-api/internal/seeder"
	"github.com/alexivanou/geocity-api/internal/service"
	"github.com/alexivanou/geocity-api/internal/stats"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	db, err := database.Connect(context.Background(), cfg.DB)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}
	logger.Info("Connected to database", zap.String("type", string(cfg.DB.Type)))

	repos := repository.NewRepositories(db, cfg.DB.Type)

	ctx := context.Background()
	// Run migrations
	if err := runMigrations(db, cfg); err != nil {
		logger.Fatal("Failed to run migrations", zap.Error(err))
	}

	isEmpty, err := repository.IsDatabaseEmpty(ctx, db)
	if err != nil {
		logger.Warn("Failed to check if database is empty", zap.Error(err))
	} else if isEmpty {
		logger.Info("Database is empty, auto-seeding data...")
		if err := autoSeedDatabase(ctx, db, repos, cfg, logger); err != nil {
			logger.Fatal("Failed to auto-seed database", zap.Error(err))
		}
		logger.Info("Database seeded successfully")
	}

	svc := service.NewService(repos.City, repos.Country, repos.Translation)
	statsCollector := stats.NewCollector(db, cfg.DB)
	router := api.NewRouter(svc, statsCollector)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("Starting server", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func runMigrations(db *sqlx.DB, cfg *config.Config) error {
	var m *migrate.Migrate
	var err error

	// Choose migration source based on DB type
	sourcePath := "file://migrations/postgres"

	if cfg.DB.IsMemory() {
		sourcePath = "file://migrations/sqlite"
		// Use driver instance directly to avoid DSN parsing issues with in-memory SQLite
		driver, err := sqlite3.WithInstance(db.DB, &sqlite3.Config{})
		if err != nil {
			return fmt.Errorf("could not create sqlite driver: %w", err)
		}
		m, err = migrate.NewWithDatabaseInstance(
			sourcePath,
			"sqlite3",
			driver,
		)
		if err != nil {
			return fmt.Errorf("could not create migrate instance: %w", err)
		}
	} else {
		// For Postgres, standard connection string works fine
		m, err = migrate.New(sourcePath, cfg.DB.DSN())
		if err != nil {
			return err
		}
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}

func autoSeedDatabase(ctx context.Context, db *sqlx.DB, repos *repository.Container, cfg *config.Config, logger *zap.Logger) error {
	parser := seeder.NewParser("data", cfg.Seeder)

	logger.Info("Parsing countries...")
	countries, err := parser.ParseCountries()
	if err != nil {
		return fmt.Errorf("failed to parse countries: %w", err)
	}

	logger.Info("Parsing cities...")
	cities, err := parser.ParseCities()
	if err != nil {
		return fmt.Errorf("failed to parse cities: %w", err)
	}

	logger.Info("Inserting countries...")
	if err := repos.Country.BulkInsertCountries(ctx, countries); err != nil {
		return fmt.Errorf("failed to insert countries: %w", err)
	}

	logger.Info("Inserting cities...")
	if err := repos.City.BulkInsertCities(ctx, cities); err != nil {
		return fmt.Errorf("failed to insert cities: %w", err)
	}

	cityIDMap := seeder.CreateCityIDMap(cities)
	countryCodeMap := seeder.CreateCountryCodeMap(countries)
	geonameIDToCountryCode := seeder.CreateCountryGeonameIDMap(countries)

	logger.Info("Parsing alternate names (streaming mode)...")
	var totalCityTranslations int
	var totalCountryTranslations int

	err = parser.ProcessAlternateNamesWithCountries(
		cityIDMap,
		countryCodeMap,
		geonameIDToCountryCode,
		func(batch []model.CityTranslation) error {
			if err := repos.Translation.BulkInsertCityTranslations(ctx, batch); err != nil {
				return err
			}
			totalCityTranslations += len(batch)
			return nil
		},
		func(batch []model.CountryTranslation) error {
			if err := repos.Translation.BulkInsertCountryTranslations(ctx, batch); err != nil {
				return err
			}
			totalCountryTranslations += len(batch)
			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("failed to process alternate names: %w", err)
	}

	logger.Info("Processed translations",
		zap.Int("city_translations", totalCityTranslations),
		zap.Int("country_translations", totalCountryTranslations),
	)

	return nil
}
