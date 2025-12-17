-- Drop indexes
DROP INDEX IF EXISTS idx_country_translations_lang;
DROP INDEX IF EXISTS idx_country_translations_country_code;
DROP INDEX IF EXISTS idx_city_translations_lang;
DROP INDEX IF EXISTS idx_city_translations_city_id;
DROP INDEX IF EXISTS idx_city_translations_name_pattern;
DROP INDEX IF EXISTS idx_city_translations_name_trgm;
DROP INDEX IF EXISTS idx_cities_name_default_pattern;
DROP INDEX IF EXISTS idx_cities_name_default_trgm;
DROP INDEX IF EXISTS idx_cities_name_default;
DROP INDEX IF EXISTS idx_cities_country_code;
DROP INDEX IF EXISTS idx_cities_population;

-- Drop tables
DROP TABLE IF EXISTS city_translations;
DROP TABLE IF EXISTS country_translations;
DROP TABLE IF EXISTS cities;
DROP TABLE IF EXISTS countries;


