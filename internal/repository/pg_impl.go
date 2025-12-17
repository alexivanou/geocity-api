package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/alexivanou/geocity-api/internal/model"
	"github.com/jmoiron/sqlx"
)

// --- PostgreSQL Implementation ---

type pgCityRepository struct {
	db *sqlx.DB
}

func (r *pgCityRepository) SearchCities(ctx context.Context, query string, limit int) ([]model.City, error) {
	q := `
		SELECT DISTINCT c.*
		FROM cities c
		WHERE unaccent(LOWER(c.name_default)) LIKE '%' || unaccent(LOWER($1)) || '%'
		UNION
		SELECT DISTINCT c.*
		FROM cities c
		INNER JOIN city_translations ct ON c.id = ct.city_id
		WHERE unaccent(LOWER(ct.name)) LIKE '%' || unaccent(LOWER($1)) || '%'
		ORDER BY population DESC
		LIMIT $2
	`
	var cities []model.City
	if err := r.db.SelectContext(ctx, &cities, q, query, limit); err != nil {
		return nil, err
	}
	return cities, nil
}

func (r *pgCityRepository) SearchCitiesWithLang(ctx context.Context, query string, lang string, limit int) ([]model.CityResult, error) {
	q := `
		SELECT DISTINCT 
			c.id,
			COALESCE(ct.name, ct_en.name, c.name_default) as name,
			COALESCE(cnt_t.name, cnt_en.name, cnt.name_default) as country,
			c.country_code,
			c.population
		FROM cities c
		JOIN countries cnt ON c.country_code = cnt.code
		LEFT JOIN city_translations ct ON c.id = ct.city_id AND ct.lang = $2
		LEFT JOIN city_translations ct_en ON c.id = ct_en.city_id AND ct_en.lang = 'en'
		LEFT JOIN country_translations cnt_t ON cnt.code = cnt_t.country_code AND cnt_t.lang = $2
		LEFT JOIN country_translations cnt_en ON cnt.code = cnt_en.country_code AND cnt_en.lang = 'en'
		WHERE 
			unaccent(LOWER(c.name_default)) LIKE '%' || unaccent(LOWER($1)) || '%'
			OR 
			EXISTS (
				SELECT 1 FROM city_translations search_ct 
				WHERE search_ct.city_id = c.id 
				AND unaccent(LOWER(search_ct.name)) LIKE '%' || unaccent(LOWER($1)) || '%'
			)
		ORDER BY c.population DESC
		LIMIT $3
	`
	var results []model.CityResult
	if err := r.db.SelectContext(ctx, &results, q, query, lang, limit); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *pgCityRepository) FindNearestCity(ctx context.Context, lat, lon float64) (*model.City, float64, error) {
	// Haversine via SQL
	q := `
		SELECT 
			*,
			(
				6371 * acos(
					least(1.0, greatest(-1.0,
						cos(radians($1)) * cos(radians(lat)) * cos(radians(lon) - radians($2)) +
						sin(radians($1)) * sin(radians(lat))
					))
				)
			) AS distance
		FROM cities
		ORDER BY distance ASC
		LIMIT 1
	`
	// We need a struct that includes Distance to scan into, or scan manually.
	// model.City doesn't have Distance. Let's use a temporary struct.
	type cityWithDist struct {
		model.City
		Distance float64 `db:"distance"`
	}

	var res cityWithDist
	if err := r.db.GetContext(ctx, &res, q, lat, lon); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	return &res.City, res.Distance, nil
}

func (r *pgCityRepository) GetCityByID(ctx context.Context, id int) (*model.City, error) {
	var city model.City
	if err := r.db.GetContext(ctx, &city, "SELECT * FROM cities WHERE id = $1", id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &city, nil
}

func (r *pgCityRepository) GetCityName(ctx context.Context, cityID int, lang string) (string, error) {
	q := `
		SELECT COALESCE(
			(SELECT name FROM city_translations WHERE city_id = $1 AND lang = $2),
			(SELECT name FROM city_translations WHERE city_id = $1 AND lang = 'en'),
			(SELECT name_default FROM cities WHERE id = $1)
		)
	`
	var name string
	if err := r.db.GetContext(ctx, &name, q, cityID, lang); err != nil {
		return "", err
	}
	return name, nil
}

func (r *pgCityRepository) BulkInsertCities(ctx context.Context, cities []model.City) error {
	// Chunking to avoid parameter limit issues even in PG (max 65535 parameters)
	chunkSize := 2000
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

type pgCountryRepository struct {
	db *sqlx.DB
}

func (r *pgCountryRepository) GetCountryName(ctx context.Context, countryCode string, lang string) (string, error) {
	q := `
		SELECT COALESCE(
			(SELECT name FROM country_translations WHERE country_code = $1 AND lang = $2),
			(SELECT name FROM country_translations WHERE country_code = $1 AND lang = 'en'),
			(SELECT name_default FROM countries WHERE code = $1)
		)
	`
	var name string
	if err := r.db.GetContext(ctx, &name, q, countryCode, lang); err != nil {
		return "", err
	}
	return name, nil
}

func (r *pgCountryRepository) BulkInsertCountries(ctx context.Context, countries []model.Country) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO countries (code, name_default)
		VALUES (:code, :name_default)`,
		countries)
	return err
}

type pgTranslationRepository struct {
	db *sqlx.DB
}

func (r *pgTranslationRepository) BulkInsertCityTranslations(ctx context.Context, translations []model.CityTranslation) error {
	// Chunking to avoid parameter limit issues
	chunkSize := 1000
	for i := 0; i < len(translations); i += chunkSize {
		end := i + chunkSize
		if end > len(translations) {
			end = len(translations)
		}
		batch := translations[i:end]

		q := `INSERT INTO city_translations (city_id, lang, name) 
			  VALUES (:city_id, :lang, :name) 
			  ON CONFLICT (city_id, lang) DO UPDATE SET name = EXCLUDED.name`

		if _, err := r.db.NamedExecContext(ctx, q, batch); err != nil {
			return err
		}
	}
	return nil
}

func (r *pgTranslationRepository) BulkInsertCountryTranslations(ctx context.Context, translations []model.CountryTranslation) error {
	chunkSize := 1000
	for i := 0; i < len(translations); i += chunkSize {
		end := i + chunkSize
		if end > len(translations) {
			end = len(translations)
		}
		batch := translations[i:end]

		q := `INSERT INTO country_translations (country_code, lang, name) 
			  VALUES (:country_code, :lang, :name) 
			  ON CONFLICT (country_code, lang) DO UPDATE SET name = EXCLUDED.name`

		if _, err := r.db.NamedExecContext(ctx, q, batch); err != nil {
			return err
		}
	}
	return nil
}

func (r *pgTranslationRepository) GetAvailableLanguages(ctx context.Context) ([]string, error) {
	q := `SELECT DISTINCT lang FROM (
			SELECT lang FROM city_translations
			UNION
			SELECT lang FROM country_translations
		) AS all_langs ORDER BY lang`
	var langs []string
	if err := r.db.SelectContext(ctx, &langs, q); err != nil {
		return nil, err
	}
	return langs, nil
}
