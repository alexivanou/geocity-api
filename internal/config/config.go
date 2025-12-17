package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds application configuration
type Config struct {
	DB     DBConfig
	Server ServerConfig
	Seeder SeederConfig
}

// DBType represents database type
type DBType string

const (
	DBTypePostgreSQL DBType = "postgres"
	DBTypeMemory     DBType = "memory"
)

// DBConfig holds database configuration
type DBConfig struct {
	Type     DBType
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// SeederConfig holds settings for data import
type SeederConfig struct {
	BatchSize        int
	MinPopulation    int
	AllowedLanguages []string
}

// DSN returns the database connection string
func (c DBConfig) DSN() string {
	if c.Type == DBTypeMemory {
		// SQLite in-memory database
		if c.Name != "" && c.Name != "geocity" {
			return fmt.Sprintf("file:%s?mode=memory&cache=shared", c.Name)
		}
		return "file::memory:?cache=shared"
	}
	// PostgreSQL connection string
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}

// IsMemory returns true if using in-memory database
func (c DBConfig) IsMemory() bool {
	return c.Type == DBTypeMemory
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	_ = godotenv.Load()

	dbType := DBType(getEnv("DB_TYPE", "memory"))
	if dbType != DBTypePostgreSQL && dbType != DBTypeMemory {
		dbType = DBTypeMemory
	}

	config := &Config{
		DB: DBConfig{
			Type:     dbType,
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "geocity"),
			Password: getEnv("DB_PASSWORD", "geocity_password"),
			Name:     getEnv("DB_NAME", "geocity"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Server: ServerConfig{
			Port: getEnv("APP_PORT", "8080"),
		},
		Seeder: SeederConfig{
			BatchSize:        getEnvAsInt("SEEDER_BATCH_SIZE", 10000),
			MinPopulation:    getEnvAsInt("SEEDER_MIN_POPULATION", 10000),
			AllowedLanguages: getEnvAsSlice("SEEDER_ALLOWED_LANGUAGES"),
		},
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	var result []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
