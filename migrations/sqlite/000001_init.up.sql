CREATE TABLE countries (
                           code VARCHAR(2) PRIMARY KEY,
                           name_default VARCHAR(255) NOT NULL
);

CREATE TABLE cities (
                        id INTEGER PRIMARY KEY,
                        country_code VARCHAR(2) NOT NULL REFERENCES countries(code) ON DELETE CASCADE,
                        name_default VARCHAR(255) NOT NULL,
                        population INTEGER NOT NULL,
                        lat DECIMAL(10, 7) NOT NULL,
                        lon DECIMAL(10, 7) NOT NULL,
                        elevation INTEGER,
                        timezone VARCHAR(100)
);

CREATE TABLE country_translations (
                                      country_code VARCHAR(2) NOT NULL REFERENCES countries(code) ON DELETE CASCADE,
                                      lang VARCHAR(2) NOT NULL,
                                      name VARCHAR(255) NOT NULL,
                                      PRIMARY KEY (country_code, lang)
);

CREATE TABLE city_translations (
                                   city_id INTEGER NOT NULL REFERENCES cities(id) ON DELETE CASCADE,
                                   lang VARCHAR(2) NOT NULL,
                                   name VARCHAR(255) NOT NULL,
                                   PRIMARY KEY (city_id, lang)
);

CREATE INDEX idx_cities_population ON cities(population DESC);
CREATE INDEX idx_cities_country_code ON cities(country_code);
CREATE INDEX idx_cities_name_default ON cities(name_default);

CREATE INDEX idx_city_translations_city_id ON city_translations(city_id);
CREATE INDEX idx_city_translations_lang ON city_translations(lang);
CREATE INDEX idx_country_translations_country_code ON country_translations(country_code);
CREATE INDEX idx_country_translations_lang ON country_translations(lang);
