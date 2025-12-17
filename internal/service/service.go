package service

import (
	"context"

	"github.com/alexivanou/geocity-api/internal/repository"
)

// Service provides business logic for the API
type Service struct {
	cityRepo        repository.CityRepository
	countryRepo     repository.CountryRepository
	translationRepo repository.TranslationRepository
}

// NewService creates a new service instance
func NewService(
	cityRepo repository.CityRepository,
	countryRepo repository.CountryRepository,
	translationRepo repository.TranslationRepository,
) *Service {
	return &Service{
		cityRepo:        cityRepo,
		countryRepo:     countryRepo,
		translationRepo: translationRepo,
	}
}

// GetAvailableLanguages returns a list of all available languages
func (s *Service) GetAvailableLanguages(ctx context.Context) ([]string, error) {
	return s.translationRepo.GetAvailableLanguages(ctx)
}
