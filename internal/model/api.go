package model

// SuggestRequest represents the request parameters for city search
type SuggestRequest struct {
	Query string
	Lang  string
	Limit int
}

// SuggestResponse represents the response for city search
type SuggestResponse struct {
	Results []CityResult `json:"results"`
}

// CityResult represents a city in the search results
type CityResult struct {
	ID          int    `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Country     string `json:"country" db:"country"`
	CountryCode string `json:"country_code" db:"country_code"`
	Population  int    `json:"population" db:"population"`
}

// CityDetailResponse represents detailed information about a city
type CityDetailResponse struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Country     string     `json:"country"`
	Coordinates Coordinate `json:"coordinates"`
	Elevation   *int       `json:"elevation"`
	Population  int        `json:"population"`
	Timezone    *string    `json:"timezone"`
}

// Coordinate represents geographic coordinates
type Coordinate struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// NearestCityResponse represents the response for nearest city search
type NearestCityResponse struct {
	City               CityDetailResponse `json:"city"`
	RequestCoordinates Coordinate         `json:"request_coordinates"`
	DistanceKm         float64            `json:"distance_km"`
}
