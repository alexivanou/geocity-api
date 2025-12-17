package main

import (
	"flag"
	"log"

	"github.com/alexivanou/geocity-api/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
)

func main() {
	var (
		command = flag.String("command", "up", "Migration command: up, down, or version")
	)
	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Prepare database URL for golang-migrate
	sourceURL := "file://migrations"
	var databaseURL string

	if cfg.DB.IsMemory() {
		// Note: For pure in-memory SQLite, golang-migrate might need a specific handling
		// or shared cache path. Using "sqlite3://" with dsn.
		// Removing "file:" prefix from DSN for the driver if present for compatibility
		dsn := cfg.DB.DSN()
		databaseURL = "sqlite3://" + dsn
	} else {
		databaseURL = cfg.DB.DSN()
	}

	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		logger.Fatal("Failed to create migration instance", zap.Error(err))
	}
	defer m.Close()

	switch *command {
	case "up":
		logger.Info("Running migrations UP")
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			logger.Fatal("Migration up failed", zap.Error(err))
		}
	case "down":
		logger.Info("Running migrations DOWN")
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			logger.Fatal("Migration down failed", zap.Error(err))
		}
	case "version":
		v, dirty, err := m.Version()
		if err != nil {
			logger.Fatal("Failed to get version", zap.Error(err))
		}
		logger.Info("Migration version", zap.Uint("version", v), zap.Bool("dirty", dirty))
	default:
		logger.Fatal("Unknown command", zap.String("command", *command))
	}

	logger.Info("Migration command completed successfully")
}
