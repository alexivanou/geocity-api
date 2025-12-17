package stats

import (
	"context"
	"testing"

	"github.com/alexivanou/geocity-api/internal/config"
	"github.com/alexivanou/geocity-api/internal/database"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *sqlx.DB {
	cfg := config.DBConfig{Type: config.DBTypeMemory}
	db, err := database.Connect(context.Background(), cfg)
	require.NoError(t, err)

	driver, err := sqlite3.WithInstance(db.DB, &sqlite3.Config{})
	require.NoError(t, err)

	m, err := migrate.NewWithDatabaseInstance(
		"file://../../migrations/sqlite",
		"sqlite3",
		driver,
	)
	require.NoError(t, err)
	err = m.Up()
	require.NoError(t, err)

	return db
}

func TestCollector_Collect(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.ExecContext(ctx, "INSERT INTO countries (code, name_default) VALUES ('XX', 'Test Country')")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "INSERT INTO cities (id, country_code, name_default, population, lat, lon) VALUES (1, 'XX', 'Test City', 1000, 10.0, 10.0)")
	require.NoError(t, err)

	cfg := config.DBConfig{Type: config.DBTypeMemory}
	collector := NewCollector(db, cfg)

	stats, err := collector.Collect(ctx)
	require.NoError(t, err)

	assert.Equal(t, "memory", stats.Database.Type)
	assert.Greater(t, stats.Database.TotalRecords, int64(0))

	var citiesCount int64
	for _, ts := range stats.Database.TableStats {
		if ts.Name == "cities" {
			citiesCount = ts.RowCount
		}
	}
	assert.Equal(t, int64(1), citiesCount)

	assert.Greater(t, stats.Memory.Alloc, uint64(0))
	assert.GreaterOrEqual(t, stats.Runtime.NumGoroutines, 1)

	stats2, err := collector.Collect(ctx)
	require.NoError(t, err)
	assert.Equal(t, stats.Memory.Alloc, stats2.Memory.Alloc)
}

func TestCollector_EmptyDB(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := config.DBConfig{Type: config.DBTypeMemory}
	collector := NewCollector(db, cfg)

	stats, err := collector.Collect(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(0), stats.Database.TotalRecords)
}
