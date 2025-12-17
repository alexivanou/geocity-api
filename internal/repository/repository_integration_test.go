//go:build integration
// +build integration

package repository

import (
	"context"
	"os"
	"testing"

	"github.com/alexivanou/geocity-api/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates a test database connection
// This requires a running PostgreSQL instance
func setupTestDB(t *testing.T) *pgxpool.Pool {
	// Get DSN from environment variable or use default
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		dsn = "postgres://geocity:geocity_password@localhost:5432/geocity_test?sslmode=disable"
	}

	db, err := pgxpool.New(context.Background(), dsn)
	require.NoError(t, err)

	// Test connection
	err = db.Ping(context.Background())
	require.NoError(t, err)

	return db
}

func TestCityRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db, config.DBTypePostgreSQL)
	cityRepo := NewCityRepository(repo)

	ctx := context.Background()

	t.Run("SearchCities", func(t *testing.T) {
		// Note: This test assumes data exists. In a real scenario,
		// you should seed data at the start of the test or use transactions with rollback.
		cities, err := cityRepo.SearchCities(ctx, "Mos", 10)
		require.NoError(t, err)
		assert.NotNil(t, cities)
		// Additional assertions based on test data
	})

	t.Run("GetCityByID", func(t *testing.T) {
		// This test requires test data in the database
		city, err := cityRepo.GetCityByID(ctx, 524901)
		require.NoError(t, err)
		// City might be nil if test data is not loaded
		_ = city
	})

	t.Run("GetCityName", func(t *testing.T) {
		// This test requires test data
		name, err := cityRepo.GetCityName(ctx, 524901, "en")
		require.NoError(t, err)
		assert.NotEmpty(t, name)
	})
}

func TestCountryRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db, config.DBTypePostgreSQL)
	countryRepo := NewCountryRepository(repo)

	ctx := context.Background()

	t.Run("GetCountryName", func(t *testing.T) {
		name, err := countryRepo.GetCountryName(ctx, "RU", "en")
		require.NoError(t, err)
		assert.NotEmpty(t, name)
	})
}
