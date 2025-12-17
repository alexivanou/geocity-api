package repository

import (
	"context"

	"github.com/alexivanou/geocity-api/internal/config"
	"github.com/alexivanou/geocity-api/internal/model"
	"github.com/jmoiron/sqlx"
)

// CityRepository defines operations for cities
type CityRepository interface {
	SearchCities(ctx context.Context, query string, limit int) ([]model.City, error)
	SearchCitiesWithLang(ctx context.Context, query string, lang string, limit int) ([]model.CityResult, error)
	FindNearestCity(ctx context.Context, lat, lon float64) (*model.City, float64, error)
	GetCityByID(ctx context.Context, id int) (*model.City, error)
	GetCityName(ctx context.Context, cityID int, lang string) (string, error)
	BulkInsertCities(ctx context.Context, cities []model.City) error
}

// CountryRepository defines operations for countries
type CountryRepository interface {
	GetCountryName(ctx context.Context, countryCode string, lang string) (string, error)
	BulkInsertCountries(ctx context.Context, countries []model.Country) error
}

// TranslationRepository defines operations for translations
type TranslationRepository interface {
	BulkInsertCityTranslations(ctx context.Context, translations []model.CityTranslation) error
	BulkInsertCountryTranslations(ctx context.Context, translations []model.CountryTranslation) error
	GetAvailableLanguages(ctx context.Context) ([]string, error)
}

// Container holds all repositories
type Container struct {
	City        CityRepository
	Country     CountryRepository
	Translation TranslationRepository
}

// NewRepositories creates repository implementations based on DB type
func NewRepositories(db *sqlx.DB, dbType config.DBType) *Container {
	if dbType == config.DBTypePostgreSQL {
		return &Container{
			City:        &pgCityRepository{db: db},
			Country:     &pgCountryRepository{db: db},
			Translation: &pgTranslationRepository{db: db},
		}
	}

	// Default to SQLite
	return &Container{
		City:        &sqliteCityRepository{db: db},
		Country:     &sqliteCountryRepository{db: db},
		Translation: &sqliteTranslationRepository{db: db},
	}
}

// Helper to check if DB is empty (used by main)
func IsDatabaseEmpty(ctx context.Context, db *sqlx.DB) (bool, error) {
	var count int
	// Using a safe query that works on both
	query := "SELECT COUNT(*) FROM cities"
	err := db.GetContext(ctx, &count, query)
	if err != nil {
		// Simplify error handling for non-existent tables
		return true, nil
	}
	return count == 0, nil
}
