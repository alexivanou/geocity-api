package service

import (
	"context"
	"fmt"

	"github.com/alexivanou/geocity-api/internal/model"
)

const (
	defaultLang    = "en"
	defaultLimit   = 10
	minQueryLength = 2
)

// SuggestCities searches for cities and returns localized results
// Optimized to use a single query with JOINs instead of N+1 queries
func (s *Service) SuggestCities(ctx context.Context, req model.SuggestRequest) (*model.SuggestResponse, error) {
	// Validate query
	if len(req.Query) < minQueryLength {
		return nil, fmt.Errorf("query must be at least %d characters", minQueryLength)
	}

	// Set defaults
	lang := req.Lang
	if lang == "" {
		lang = defaultLang
	}
	limit := req.Limit
	if limit <= 0 {
		limit = defaultLimit
	}

	// Search cities with localized names in a single query (solves N+1 problem)
	results, err := s.cityRepo.SearchCitiesWithLang(ctx, req.Query, lang, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search cities: %w", err)
	}

	return &model.SuggestResponse{Results: results}, nil
}

// GetCityByID retrieves detailed information about a city
func (s *Service) GetCityByID(ctx context.Context, id int, lang string) (*model.CityDetailResponse, error) {
	// Get city from database
	city, err := s.cityRepo.GetCityByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get city: %w", err)
	}
	if city == nil {
		return nil, nil // City not found
	}

	// Set default language if not provided
	if lang == "" {
		lang = defaultLang
	}

	// Get localized city name
	cityName, err := s.cityRepo.GetCityName(ctx, city.ID, lang)
	if err != nil {
		return nil, fmt.Errorf("failed to get city name: %w", err)
	}

	// Get localized country name
	countryName, err := s.countryRepo.GetCountryName(ctx, city.CountryCode, lang)
	if err != nil {
		return nil, fmt.Errorf("failed to get country name: %w", err)
	}

	response := &model.CityDetailResponse{
		ID:      city.ID,
		Name:    cityName,
		Country: countryName,
		Coordinates: model.Coordinate{
			Lat: city.Lat,
			Lon: city.Lon,
		},
		Elevation:  city.Elevation,
		Population: city.Population,
		Timezone:   city.Timezone,
	}

	return response, nil
}

// FindNearestCity finds the closest city to the given coordinates
func (s *Service) FindNearestCity(ctx context.Context, lat, lon float64, lang string) (*model.NearestCityResponse, error) {
	if lang == "" {
		lang = defaultLang
	}

	city, dist, err := s.cityRepo.FindNearestCity(ctx, lat, lon)
	if err != nil {
		return nil, fmt.Errorf("failed to find nearest city: %w", err)
	}
	if city == nil {
		return nil, nil
	}

	// Get localized names
	cityName, err := s.cityRepo.GetCityName(ctx, city.ID, lang)
	if err != nil {
		return nil, fmt.Errorf("failed to get city name: %w", err)
	}

	countryName, err := s.countryRepo.GetCountryName(ctx, city.CountryCode, lang)
	if err != nil {
		return nil, fmt.Errorf("failed to get country name: %w", err)
	}

	return &model.NearestCityResponse{
		City: model.CityDetailResponse{
			ID:          city.ID,
			Name:        cityName,
			Country:     countryName,
			Coordinates: model.Coordinate{Lat: city.Lat, Lon: city.Lon},
			Elevation:   city.Elevation,
			Population:  city.Population,
			Timezone:    city.Timezone,
		},
		RequestCoordinates: model.Coordinate{Lat: lat, Lon: lon},
		DistanceKm:         dist,
	}, nil
}
