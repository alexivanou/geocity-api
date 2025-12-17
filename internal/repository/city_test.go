package repository

import (
	"context"
	"testing"

	"github.com/alexivanou/geocity-api/internal/config"
	"github.com/alexivanou/geocity-api/internal/database"
	"github.com/alexivanou/geocity-api/internal/model"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRepo(t *testing.T) (*Container, func()) {
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

	repos := NewRepositories(db, config.DBTypeMemory)
	ctx := context.Background()

	_, err = db.Exec("INSERT INTO countries (code, name_default) VALUES (?, ?)", "DE", "Germany")
	require.NoError(t, err)

	cities := []model.City{
		{ID: 1, CountryCode: "DE", NameDefault: "Berlin", Population: 3600000, Lat: 52.5200, Lon: 13.4050},
		{ID: 2, CountryCode: "DE", NameDefault: "Potsdam", Population: 180000, Lat: 52.3967, Lon: 13.0583},
	}
	err = repos.City.BulkInsertCities(ctx, cities)
	require.NoError(t, err)

	translations := []model.CityTranslation{
		{CityID: 1, Lang: "de", Name: "Berlin"},
		{CityID: 1, Lang: "en", Name: "Berlin"},
	}
	err = repos.Translation.BulkInsertCityTranslations(ctx, translations)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
	}

	return repos, cleanup
}

func TestCityRepository_SearchCitiesWithLang(t *testing.T) {
	repos, cleanup := setupRepo(t)
	defer cleanup()
	ctx := context.Background()

	tests := []struct {
		name          string
		query         string
		lang          string
		expectedName  string
		expectedCount int
	}{
		{
			name:          "Search in English (default)",
			query:         "Berl",
			lang:          "en",
			expectedName:  "Berlin",
			expectedCount: 1,
		},
		{
			name:          "Search case insensitive",
			query:         "berl",
			lang:          "en",
			expectedName:  "Berlin",
			expectedCount: 1,
		},
		{
			name:          "Search mismatch",
			query:         "Paris",
			lang:          "en",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := repos.City.SearchCitiesWithLang(ctx, tt.query, tt.lang, 10)
			require.NoError(t, err)
			assert.Len(t, results, tt.expectedCount)
			if tt.expectedCount > 0 {
				assert.Equal(t, tt.expectedName, results[0].Name)
			}
		})
	}
}

func TestCityRepository_FindNearestCity(t *testing.T) {
	repos, cleanup := setupRepo(t)
	defer cleanup()
	ctx := context.Background()

	lat, lon := 52.5200, 13.4000
	city, dist, err := repos.City.FindNearestCity(ctx, lat, lon)
	require.NoError(t, err)
	require.NotNil(t, city)
	assert.Equal(t, "Berlin", city.NameDefault)
	assert.Less(t, dist, 10.0)
}
