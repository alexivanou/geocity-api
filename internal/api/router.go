package api

import (
	"github.com/alexivanou/geocity-api/internal/service"
	"github.com/alexivanou/geocity-api/internal/stats"
	"github.com/gorilla/mux"
)

// NewRouter creates a new HTTP router
func NewRouter(service service.ServiceInterface, statsCollector *stats.Collector) *mux.Router {
	handler := NewHandler(service)
	statsHandler := NewStatsHandler(statsCollector)

	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", handler.HealthCheck).Methods("GET")

	// API v1
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/suggest", handler.SuggestCities).Methods("GET")
	v1.HandleFunc("/nearest", handler.FindNearestCity).Methods("GET")
	v1.HandleFunc("/city/{id}", handler.GetCity).Methods("GET")
	v1.HandleFunc("/languages", handler.GetAvailableLanguages).Methods("GET")
	v1.HandleFunc("/stats", statsHandler.GetStats).Methods("GET")

	return router
}
