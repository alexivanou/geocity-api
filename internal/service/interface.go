package service

import (
	"context"

	"github.com/alexivanou/geocity-api/internal/model"
)

// ServiceInterface defines the service interface for testing
type ServiceInterface interface {
	SuggestCities(ctx context.Context, req model.SuggestRequest) (*model.SuggestResponse, error)
	GetCityByID(ctx context.Context, id int, lang string) (*model.CityDetailResponse, error)
	FindNearestCity(ctx context.Context, lat, lon float64, lang string) (*model.NearestCityResponse, error)
	GetAvailableLanguages(ctx context.Context) ([]string, error)
}
