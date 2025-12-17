package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/alexivanou/geocity-api/internal/model"
	"github.com/alexivanou/geocity-api/internal/service"
	"github.com/gorilla/mux"
)

// Handler handles HTTP requests
type Handler struct {
	service service.ServiceInterface
}

// NewHandler creates a new handler instance
func NewHandler(service service.ServiceInterface) *Handler {
	return &Handler{service: service}
}

// SuggestCities handles GET /api/v1/suggest
func (h *Handler) SuggestCities(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	if len(query) < 2 {
		http.Error(w, "query must be at least 2 characters", http.StatusBadRequest)
		return
	}

	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "en"
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			http.Error(w, "invalid limit parameter", http.StatusBadRequest)
			return
		}
	}

	req := model.SuggestRequest{
		Query: query,
		Lang:  lang,
		Limit: limit,
	}

	response, err := h.service.SuggestCities(r.Context(), req)
	if err != nil {
		log.Printf("Error suggesting cities: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

// FindNearestCity handles GET /api/v1/nearest
func (h *Handler) FindNearestCity(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lonStr := r.URL.Query().Get("lon")

	if latStr == "" || lonStr == "" {
		http.Error(w, "parameters 'lat' and 'lon' are required", http.StatusBadRequest)
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		http.Error(w, "invalid lat parameter", http.StatusBadRequest)
		return
	}

	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		http.Error(w, "invalid lon parameter", http.StatusBadRequest)
		return
	}

	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		http.Error(w, "invalid coordinates range", http.StatusBadRequest)
		return
	}

	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "en"
	}

	response, err := h.service.FindNearestCity(r.Context(), lat, lon, lang)
	if err != nil {
		log.Printf("Error finding nearest city: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if response == nil {
		http.Error(w, "no cities found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

// GetCity handles GET /api/v1/city/{id}
func (h *Handler) GetCity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid city id", http.StatusBadRequest)
		return
	}

	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "en"
	}

	city, err := h.service.GetCityByID(r.Context(), id, lang)
	if err != nil {
		log.Printf("Error getting city: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if city == nil {
		http.Error(w, "city not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(city); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

// GetAvailableLanguages handles GET /api/v1/languages
func (h *Handler) GetAvailableLanguages(w http.ResponseWriter, r *http.Request) {
	languages, err := h.service.GetAvailableLanguages(r.Context())
	if err != nil {
		log.Printf("Error getting available languages: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"languages": languages,
		"count":     len(languages),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
