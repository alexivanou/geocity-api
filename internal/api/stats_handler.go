package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/alexivanou/geocity-api/internal/stats"
)

// StatsHandler handles statistics requests
type StatsHandler struct {
	collector *stats.Collector
}

// NewStatsHandler creates a new stats handler
func NewStatsHandler(collector *stats.Collector) *StatsHandler {
	return &StatsHandler{collector: collector}
}

// GetStats handles GET /api/v1/stats
func (h *StatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.collector.Collect(r.Context())
	if err != nil {
		log.Printf("Error collecting statistics: %v", err)
		http.Error(w, "failed to collect statistics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Error encoding statistics: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
