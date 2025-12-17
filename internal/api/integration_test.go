package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexivanou/geocity-api/internal/config"
	"github.com/alexivanou/geocity-api/internal/database"
	"github.com/alexivanou/geocity-api/internal/model"
	"github.com/alexivanou/geocity-api/internal/repository"
	"github.com/alexivanou/geocity-api/internal/service"
	"github.com/alexivanou/geocity-api/internal/stats"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupIntegrationStack(t *testing.T) *http.Handler {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	dbName := fmt.Sprintf("testdb_%d", rng.Int())

	cfg := config.DBConfig{
		Type: config.DBTypeMemory,
		Name: dbName,
	}

	db, err := database.Connect(context.Background(), cfg)
	require.NoError(t, err)

	driver, err := sqlite3.WithInstance(db.DB, &sqlite3.Config{})
	require.NoError(t, err)

	// Point to the sqlite migrations folder
	m, err := migrate.NewWithDatabaseInstance(
		"file://../../migrations/sqlite",
		"sqlite3",
		driver,
	)
	require.NoError(t, err)
	err = m.Up()
	require.NoError(t, err)

	ctx := context.Background()
	_, err = db.ExecContext(ctx, "INSERT INTO countries (code, name_default) VALUES ('IE', 'Ireland')")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "INSERT INTO cities (id, country_code, name_default, population, lat, lon) VALUES (1, 'IE', 'Dublin', 544000, 53.3498, -6.2603)")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "INSERT INTO city_translations (city_id, lang, name) VALUES (1, 'ga', 'Baile Átha Cliath')")
	require.NoError(t, err)

	repos := repository.NewRepositories(db, config.DBTypeMemory)
	svc := service.NewService(repos.City, repos.Country, repos.Translation)
	statsCollector := stats.NewCollector(db, cfg)

	router := NewRouter(svc, statsCollector)
	h := http.Handler(router)
	return &h
}

func TestAPI_Integration_Suggest(t *testing.T) {
	handler := *setupIntegrationStack(t)

	req := httptest.NewRequest("GET", "/api/v1/suggest?q=Dub", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.SuggestResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, "Dublin", resp.Results[0].Name)
}

func TestAPI_Integration_Nearest(t *testing.T) {
	handler := *setupIntegrationStack(t)

	req := httptest.NewRequest("GET", "/api/v1/nearest?lat=53.35&lon=-6.26", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var resp model.NearestCityResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, "Dublin", resp.City.Name)
}
