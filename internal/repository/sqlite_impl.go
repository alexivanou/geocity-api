package repository

import (
	"context"
	"database/sql"
	"errors"
	"math"

	"github.com/alexivanou/geocity-api/internal/model"
	"github.com/jmoiron/sqlx"
)

type sqliteCityRepository struct {
	db *sqlx.DB
}

func (r *sqliteCityRepository) SearchCities(ctx context.Context, query string, limit int) ([]model.City, error) {
	q := `
		SELECT DISTINCT c.*
		FROM cities c
		WHERE LOWER(c.name_default) LIKE '%' || LOWER(?) || '%'
		UNION
		SELECT DISTINCT c.*
		FROM cities c
		INNER JOIN city_translations ct ON c.id = ct.city_id
		WHERE LOWER(ct.name) LIKE '%' || LOWER(?) || '%'
		ORDER BY population DESC
		LIMIT ?
	`
	var cities []model.City
	if err := r.db.SelectContext(ctx, &cities, q, query, query, limit); err != nil {
		return nil, err
	}
	return cities, nil
}

func (r *sqliteCityRepository) SearchCitiesWithLang(ctx context.Context, query string, lang string, limit int) ([]model.CityResult, error) {
	q := `
		SELECT DISTINCT 
			c.id,
			COALESCE(ct.name, ct_en.name, c.name_default) as name,
			COALESCE(cnt_t.name, cnt_en.name, cnt.name_default) as country,
			c.country_code,
			c.population
		FROM cities c
		JOIN countries cnt ON c.country_code = cnt.code
		LEFT JOIN city_translations ct ON c.id = ct.city_id AND ct.lang = ?
		LEFT JOIN city_translations ct_en ON c.id = ct_en.city_id AND ct_en.lang = 'en'
		LEFT JOIN country_translations cnt_t ON cnt.code = cnt_t.country_code AND cnt_t.lang = ?
		LEFT JOIN country_translations cnt_en ON cnt.code = cnt_en.country_code AND cnt_en.lang = 'en'
		WHERE 
			LOWER(c.name_default) LIKE '%' || LOWER(?) || '%'
			OR 
			EXISTS (
				SELECT 1 FROM city_translations search_ct 
				WHERE search_ct.city_id = c.id 
				AND LOWER(search_ct.name) LIKE '%' || LOWER(?) || '%'
			)
		ORDER BY c.population DESC
		LIMIT ?
	`
	var results []model.CityResult
	if err := r.db.SelectContext(ctx, &results, q, lang, lang, query, query, limit); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *sqliteCityRepository) FindNearestCity(ctx context.Context, lat, lon float64) (*model.City, float64, error) {
	delta := 2.0
	q := `
		SELECT * FROM cities
		WHERE lat BETWEEN ? AND ? AND lon BETWEEN ? AND ?
	`
	var candidates []model.City
	err := r.db.SelectContext(ctx, &candidates, q, lat-delta, lat+delta, lon-delta, lon+delta)
	if err != nil {
		return nil, 0, err
	}

	if len(candidates) == 0 {
		if err := r.db.SelectContext(ctx, &candidates, "SELECT * FROM cities"); err != nil {
			return nil, 0, err
		}
	}

	var nearest *model.City
	minDist := math.MaxFloat64

	for i := range candidates {
		city := candidates[i]
		dist := calculateDistance(lat, lon, city.Lat, city.Lon)
		if dist < minDist {
			minDist = dist
			nearest = &city
		}
	}

	if nearest == nil {
		return nil, 0, nil
	}
	return nearest, minDist, nil
}

func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)
	lat1Rad := lat1 * (math.Pi / 180.0)
	lat2Rad := lat2 * (math.Pi / 180.0)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func (r *sqliteCityRepository) GetCityByID(ctx context.Context, id int) (*model.City, error) {
	var city model.City
	if err := r.db.GetContext(ctx, &city, "SELECT * FROM cities WHERE id = ?", id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &city, nil
}

func (r *sqliteCityRepository) GetCityName(ctx context.Context, cityID int, lang string) (string, error) {
	q := `
		SELECT COALESCE(
			(SELECT name FROM city_translations WHERE city_id = ? AND lang = ?),
			(SELECT name FROM city_translations WHERE city_id = ? AND lang = 'en'),
			(SELECT name_default FROM cities WHERE id = ?)
		)
	`
	var name string
	if err := r.db.GetContext(ctx, &name, q, cityID, lang, cityID, cityID); err != nil {
		return "", err
	}
	return name, nil
}

func (r *sqliteCityRepository) BulkInsertCities(ctx context.Context, cities []model.City) error {
	// SQLite variable limit workaround (batch size of 100 * 8 params = 800 variables, well within standard limits)
	chunkSize := 100
	for i := 0; i < len(cities); i += chunkSize {
		end := i + chunkSize
		if end > len(cities) {
			end = len(cities)
		}
		batch := cities[i:end]

		_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO cities (id, country_code, name_default, population, lat, lon, elevation, timezone)
		VALUES (:id, :country_code, :name_default, :population, :lat, :lon, :elevation, :timezone)`,
			batch)
		if err != nil {
			return err
		}
	}
	return nil
}

type sqliteCountryRepository struct {
	db *sqlx.DB
}

func (r *sqliteCountryRepository) GetCountryName(ctx context.Context, countryCode string, lang string) (string, error) {
	q := `
		SELECT COALESCE(
			(SELECT name FROM country_translations WHERE country_code = ? AND lang = ?),
			(SELECT name FROM country_translations WHERE country_code = ? AND lang = 'en'),
			(SELECT name_default FROM countries WHERE code = ?)
		)
	`
	var name string
	if err := r.db.GetContext(ctx, &name, q, countryCode, lang, countryCode, countryCode); err != nil {
		return "", err
	}
	return name, nil
}

func (r *sqliteCountryRepository) BulkInsertCountries(ctx context.Context, countries []model.Country) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO countries (code, name_default)
		VALUES (:code, :name_default)`,
		countries)
	return err
}

type sqliteTranslationRepository struct {
	db *sqlx.DB
}

func (r *sqliteTranslationRepository) BulkInsertCityTranslations(ctx context.Context, translations []model.CityTranslation) error {
	chunkSize := 500
	for i := 0; i < len(translations); i += chunkSize {
		end := i + chunkSize
		if end > len(translations) {
			end = len(translations)
		}
		batch := translations[i:end]

		q := `INSERT OR REPLACE INTO city_translations (city_id, lang, name) 
			  VALUES (:city_id, :lang, :name)`

		if _, err := r.db.NamedExecContext(ctx, q, batch); err != nil {
			return err
		}
	}
	return nil
}

func (r *sqliteTranslationRepository) BulkInsertCountryTranslations(ctx context.Context, translations []model.CountryTranslation) error {
	chunkSize := 500
	for i := 0; i < len(translations); i += chunkSize {
		end := i + chunkSize
		if end > len(translations) {
			end = len(translations)
		}
		batch := translations[i:end]

		q := `INSERT OR REPLACE INTO country_translations (country_code, lang, name) 
			  VALUES (:country_code, :lang, :name)`

		if _, err := r.db.NamedExecContext(ctx, q, batch); err != nil {
			return err
		}
	}
	return nil
}

func (r *sqliteTranslationRepository) GetAvailableLanguages(ctx context.Context) ([]string, error) {
	q := `SELECT DISTINCT lang FROM (
			SELECT lang FROM city_translations
			UNION
			SELECT lang FROM country_translations
		) ORDER BY lang`
	var langs []string
	if err := r.db.SelectContext(ctx, &langs, q); err != nil {
		return nil, err
	}
	return langs, nil
}
