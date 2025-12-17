package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Save and restore environment variables after the test
	envVars := []string{
		"DB_TYPE", "DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"APP_PORT", "SEEDER_BATCH_SIZE", "SEEDER_MIN_POPULATION", "SEEDER_ALLOWED_LANGUAGES",
	}
	originalEnv := make(map[string]string)
	for _, key := range envVars {
		originalEnv[key] = os.Getenv(key)
		os.Unsetenv(key) // Clear before test
	}
	defer func() {
		for key, val := range originalEnv {
			if val != "" {
				os.Setenv(key, val)
			}
		}
	}()

	t.Run("Default values", func(t *testing.T) {
		cfg, err := Load()
		require.NoError(t, err)

		assert.Equal(t, DBTypeMemory, cfg.DB.Type)
		assert.Equal(t, "8080", cfg.Server.Port)
		assert.Equal(t, 10000, cfg.Seeder.BatchSize)
		assert.Empty(t, cfg.Seeder.AllowedLanguages)
	})

	t.Run("Custom environment variables", func(t *testing.T) {
		t.Setenv("DB_TYPE", "postgres")
		t.Setenv("DB_HOST", "test-db")
		t.Setenv("APP_PORT", "9090")
		t.Setenv("SEEDER_BATCH_SIZE", "500")
		t.Setenv("SEEDER_ALLOWED_LANGUAGES", "en,ru, de") // Space after comma

		cfg, err := Load()
		require.NoError(t, err)

		assert.Equal(t, DBTypePostgreSQL, cfg.DB.Type)
		assert.Equal(t, "test-db", cfg.DB.Host)
		assert.Equal(t, "9090", cfg.Server.Port)
		assert.Equal(t, 500, cfg.Seeder.BatchSize)
		assert.Equal(t, []string{"en", "ru", "de"}, cfg.Seeder.AllowedLanguages)
	})

	t.Run("Invalid integer fallback", func(t *testing.T) {
		t.Setenv("SEEDER_BATCH_SIZE", "not-a-number")
		cfg, err := Load()
		require.NoError(t, err)

		// Should return default value
		assert.Equal(t, 10000, cfg.Seeder.BatchSize)
	})
}

func TestDBConfig_DSN(t *testing.T) {
	t.Run("Memory DSN default", func(t *testing.T) {
		c := DBConfig{Type: DBTypeMemory}
		assert.Equal(t, "file::memory:?cache=shared", c.DSN())
	})

	t.Run("Memory DSN file", func(t *testing.T) {
		c := DBConfig{Type: DBTypeMemory, Name: "test.db"}
		assert.Equal(t, "file:test.db?mode=memory&cache=shared", c.DSN())
	})

	t.Run("Postgres DSN", func(t *testing.T) {
		c := DBConfig{
			Type:     DBTypePostgreSQL,
			Host:     "localhost",
			Port:     "5432",
			User:     "user",
			Password: "pass",
			Name:     "db",
			SSLMode:  "disable",
		}
		expected := "postgres://user:pass@localhost:5432/db?sslmode=disable"
		assert.Equal(t, expected, c.DSN())
	})
}
