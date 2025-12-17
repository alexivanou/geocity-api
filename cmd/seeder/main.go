package main

import (
	"context"
	"fmt"
	"log"

	"github.com/alexivanou/geocity-api/internal/config"
	"github.com/alexivanou/geocity-api/internal/database"
	"github.com/alexivanou/geocity-api/internal/model"
	"github.com/alexivanou/geocity-api/internal/repository"
	"github.com/alexivanou/geocity-api/internal/seeder"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	db, err := database.Connect(context.Background(), cfg.DB)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}

	logger.Info("Connected to database", zap.String("type", string(cfg.DB.Type)))
	logger.Info("Starting data import...")

	parser := seeder.NewParser("data", cfg.Seeder)

	logger.Info("Parsing countries...")
	countries, err := parser.ParseCountries()
	if err != nil {
		logger.Fatal("Failed to parse countries", zap.Error(err))
	}

	logger.Info("Parsing cities...")
	cities, err := parser.ParseCities()
	if err != nil {
		logger.Fatal("Failed to parse cities", zap.Error(err))
	}

	ctx := context.Background()
	// Auto-migrate if using memory DB to ensure schema exists
	if cfg.DB.IsMemory() {
		m, err := migrate.New("file://migrations", "sqlite3://"+cfg.DB.DSN())
		if err != nil {
			logger.Fatal("Failed to init migration", zap.Error(err))
		}
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			logger.Fatal("Failed to run migration", zap.Error(err))
		}
	}

	// Begin transaction is handled by Repository usually or here if we want atomic seed
	// For simplicity using repos directly.
	repos := repository.NewRepositories(db, cfg.DB.Type)

	// Clear existing data (optional, simplified)
	if cfg.DB.IsMemory() {
		// Fast truncate for testing
		_, _ = db.Exec("DELETE FROM city_translations; DELETE FROM country_translations; DELETE FROM cities; DELETE FROM countries;")
	}

	logger.Info("Inserting countries...")
	if err := repos.Country.BulkInsertCountries(ctx, countries); err != nil {
		logger.Fatal("Failed to insert countries", zap.Error(err))
	}

	logger.Info("Inserting cities...")
	if err := repos.City.BulkInsertCities(ctx, cities); err != nil {
		logger.Fatal("Failed to insert cities", zap.Error(err))
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
				return fmt.Errorf("failed to insert city translations batch: %w", err)
			}
			totalCityTranslations += len(batch)
			return nil
		},
		func(batch []model.CountryTranslation) error {
			if err := repos.Translation.BulkInsertCountryTranslations(ctx, batch); err != nil {
				return fmt.Errorf("failed to insert country translations batch: %w", err)
			}
			totalCountryTranslations += len(batch)
			return nil
		},
	)
	if err != nil {
		logger.Fatal("Failed to process alternate names", zap.Error(err))
	}

	logger.Info("Data import completed successfully!",
		zap.Int("cities", len(cities)),
		zap.Int("city_translations", totalCityTranslations),
	)
}
