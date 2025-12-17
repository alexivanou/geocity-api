package database

import (
	"context"
	"fmt"

	"github.com/alexivanou/geocity-api/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib" // Postgres driver for database/sql
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// Connect creates a database connection based on configuration using sqlx
func Connect(ctx context.Context, cfg config.DBConfig) (*sqlx.DB, error) {
	var driverName string
	var dsn string

	if cfg.IsMemory() {
		driverName = "sqlite3"
		dsn = cfg.DSN()
	} else {
		driverName = "pgx"
		dsn = cfg.DSN()
	}

	db, err := sqlx.ConnectContext(ctx, driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Specific settings for SQLite to enable Foreign Keys
	if cfg.IsMemory() {
		if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
		}
	}

	return db, nil
}
