package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexivanou/geocity-api/internal/model"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockService is a mock implementation of ServiceInterface
type MockService struct {
	mock.Mock
}

func (m *MockService) SuggestCities(ctx context.Context, req model.SuggestRequest) (*model.SuggestResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SuggestResponse), args.Error(1)
}

func (m *MockService) GetCityByID(ctx context.Context, id int, lang string) (*model.CityDetailResponse, error) {
	args := m.Called(ctx, id, lang)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.CityDetailResponse), args.Error(1)
}

func (m *MockService) FindNearestCity(ctx context.Context, lat, lon float64, lang string) (*model.NearestCityResponse, error) {
	args := m.Called(ctx, lat, lon, lang)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.NearestCityResponse), args.Error(1)
}

func (m *MockService) GetAvailableLanguages(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func TestHandler_SuggestCities(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		lang           string
		limit          string
		mockSetup      func(*MockService)
		expectedStatus int
		expectedBody   bool
	}{
		{
			name:  "successful request",
			query: "Ber",
			lang:  "de",
			limit: "10",
			mockSetup: func(ms *MockService) {
				ms.On("SuggestCities", mock.Anything, mock.MatchedBy(func(req model.SuggestRequest) bool {
					return req.Query == "Ber" && req.Lang == "de" && req.Limit == 10
				})).Return(&model.SuggestResponse{
					Results: []model.CityResult{
						{ID: 2950159, Name: "Berlin", Country: "Deutschland", CountryCode: "DE", Population: 3644826},
					},
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   true,
		},
		{
			name:           "missing query parameter",
			query:          "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "query too short",
			query:          "B",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			if tt.mockSetup != nil {
				tt.mockSetup(mockService)
			}

			handler := &Handler{service: mockService}

			req, _ := http.NewRequest("GET", "/api/v1/suggest", nil)
			q := req.URL.Query()
			if tt.query != "" {
				q.Add("q", tt.query)
			}
			if tt.lang != "" {
				q.Add("lang", tt.lang)
			}
			if tt.limit != "" {
				q.Add("limit", tt.limit)
			}
			req.URL.RawQuery = q.Encode()

			rr := httptest.NewRecorder()
			handler.SuggestCities(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestHandler_FindNearestCity(t *testing.T) {
	tests := []struct {
		name           string
		lat            string
		lon            string
		mockSetup      func(*MockService)
		expectedStatus int
	}{
		{
			name: "successful request",
			lat:  "52.52",
			lon:  "13.40",
			mockSetup: func(ms *MockService) {
				ms.On("FindNearestCity", mock.Anything, 52.52, 13.40, "en").Return(&model.NearestCityResponse{
					City:       model.CityDetailResponse{Name: "Berlin"},
					DistanceKm: 0.5,
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockService)
			if tt.mockSetup != nil {
				tt.mockSetup(mockService)
			}
			handler := &Handler{service: mockService}
			req, _ := http.NewRequest("GET", "/api/v1/nearest", nil)
			q := req.URL.Query()
			if tt.lat != "" {
				q.Add("lat", tt.lat)
			}
			if tt.lon != "" {
				q.Add("lon", tt.lon)
			}
			req.URL.RawQuery = q.Encode()
			rr := httptest.NewRecorder()
			handler.FindNearestCity(rr, req)
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestHandler_GetCity(t *testing.T) {
	mockService := new(MockService)
	handler := &Handler{service: mockService}

	tests := []struct {
		name           string
		cityID         string
		lang           string
		mockSetup      func(*MockService)
		expectedStatus int
	}{
		{
			name:   "successful request",
			cityID: "2950159",
			lang:   "de",
			mockSetup: func(ms *MockService) {
				elevation := 34
				timezone := "Europe/Berlin"
				ms.On("GetCityByID", mock.Anything, 2950159, "de").Return(&model.CityDetailResponse{
					ID:          2950159,
					Name:        "Berlin",
					Country:     "Deutschland",
					Coordinates: model.Coordinate{Lat: 52.5200, Lon: 13.4050},
					Elevation:   &elevation,
					Population:  3644826,
					Timezone:    &timezone,
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockSetup != nil {
				tt.mockSetup(mockService)
			}
			req, _ := http.NewRequest("GET", "/api/v1/city/"+tt.cityID, nil)
			if tt.lang != "" {
				q := req.URL.Query()
				q.Add("lang", tt.lang)
				req.URL.RawQuery = q.Encode()
			}
			req = mux.SetURLVars(req, map[string]string{"id": tt.cityID})
			rr := httptest.NewRecorder()
			handler.GetCity(rr, req)
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
