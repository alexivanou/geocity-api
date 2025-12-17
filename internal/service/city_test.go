package service

import (
	"context"
	"testing"

	"github.com/alexivanou/geocity-api/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCityRepository implements repository.CityRepository interface
type MockCityRepository struct {
	mock.Mock
}

func (m *MockCityRepository) SearchCities(ctx context.Context, query string, limit int) ([]model.City, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.City), args.Error(1)
}

func (m *MockCityRepository) SearchCitiesWithLang(ctx context.Context, query string, lang string, limit int) ([]model.CityResult, error) {
	args := m.Called(ctx, query, lang, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.CityResult), args.Error(1)
}

func (m *MockCityRepository) FindNearestCity(ctx context.Context, lat, lon float64) (*model.City, float64, error) {
	args := m.Called(ctx, lat, lon)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).(*model.City), args.Get(1).(float64), args.Error(2)
}

func (m *MockCityRepository) GetCityByID(ctx context.Context, id int) (*model.City, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.City), args.Error(1)
}

func (m *MockCityRepository) GetCityName(ctx context.Context, cityID int, lang string) (string, error) {
	args := m.Called(ctx, cityID, lang)
	return args.String(0), args.Error(1)
}

func (m *MockCityRepository) BulkInsertCities(ctx context.Context, cities []model.City) error {
	args := m.Called(ctx, cities)
	return args.Error(0)
}

// MockCountryRepository implements repository.CountryRepository interface
type MockCountryRepository struct {
	mock.Mock
}

func (m *MockCountryRepository) GetCountryName(ctx context.Context, countryCode string, lang string) (string, error) {
	args := m.Called(ctx, countryCode, lang)
	return args.String(0), args.Error(1)
}

func (m *MockCountryRepository) BulkInsertCountries(ctx context.Context, countries []model.Country) error {
	args := m.Called(ctx, countries)
	return args.Error(0)
}

type MockTranslationRepository struct {
	mock.Mock
}

func (m *MockTranslationRepository) BulkInsertCityTranslations(ctx context.Context, translations []model.CityTranslation) error {
	args := m.Called(ctx, translations)
	return args.Error(0)
}
func (m *MockTranslationRepository) BulkInsertCountryTranslations(ctx context.Context, translations []model.CountryTranslation) error {
	args := m.Called(ctx, translations)
	return args.Error(0)
}
func (m *MockTranslationRepository) GetAvailableLanguages(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func TestService_SuggestCities(t *testing.T) {
	tests := []struct {
		name          string
		req           model.SuggestRequest
		setupMocks    func(*MockCityRepository, *MockCountryRepository)
		expectedError string
		expectedCount int
	}{
		{
			name: "successful search",
			req: model.SuggestRequest{
				Query: "Dub",
				Lang:  "en",
				Limit: 10,
			},
			setupMocks: func(cityRepo *MockCityRepository, countryRepo *MockCountryRepository) {
				cityRepo.On("SearchCitiesWithLang", mock.Anything, "Dub", "en", 10).Return([]model.CityResult{
					{ID: 1, Name: "Dublin", Country: "Ireland", CountryCode: "IE", Population: 500000},
				}, nil)
			},
			expectedCount: 1,
		},
		{
			name: "query too short",
			req: model.SuggestRequest{
				Query: "D",
				Lang:  "en",
			},
			expectedError: "query must be at least 2 characters",
		},
		{
			name: "empty query",
			req: model.SuggestRequest{
				Query: "",
				Lang:  "en",
			},
			expectedError: "query must be at least 2 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCityRepo := new(MockCityRepository)
			mockCountryRepo := new(MockCountryRepository)
			mockTranslationRepo := new(MockTranslationRepository)

			if tt.setupMocks != nil {
				tt.setupMocks(mockCityRepo, mockCountryRepo)
			}

			svc := NewService(mockCityRepo, mockCountryRepo, mockTranslationRepo)

			resp, err := svc.SuggestCities(context.Background(), tt.req)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Len(t, resp.Results, tt.expectedCount)
			}
		})
	}
}
