package model

// City represents a city in the database
type City struct {
	ID          int     `db:"id"`
	CountryCode string  `db:"country_code"`
	NameDefault string  `db:"name_default"`
	Population  int     `db:"population"`
	Lat         float64 `db:"lat"`
	Lon         float64 `db:"lon"`
	Elevation   *int    `db:"elevation"`
	Timezone    *string `db:"timezone"`
}

// CityTranslation represents a translation of a city name
type CityTranslation struct {
	CityID int    `db:"city_id"`
	Lang   string `db:"lang"`
	Name   string `db:"name"`
}

// Country represents a country in the database
type Country struct {
	Code        string `db:"code"`
	NameDefault string `db:"name_default"`
	// GeonameID is used during seeding to link alternate names to the country
	// It is not stored in the countries table (which uses Code as PK)
	GeonameID int `db:"-"`
}

// CountryTranslation represents a translation of a country name
type CountryTranslation struct {
	CountryCode string `db:"country_code"`
	Lang        string `db:"lang"`
	Name        string `db:"name"`
}
