package repository

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateDistance(t *testing.T) {
	tests := []struct {
		name     string
		lat1     float64
		lon1     float64
		lat2     float64
		lon2     float64
		expected float64 // km
		epsilon  float64
	}{
		{
			name:     "Same point",
			lat1:     52.5200,
			lon1:     13.4050,
			lat2:     52.5200,
			lon2:     13.4050,
			expected: 0.0,
			epsilon:  0.001,
		},
		{
			name: "Berlin to Potsdam",
			// Berlin
			lat1: 52.5200,
			lon1: 13.4050,
			// Potsdam
			lat2: 52.3989,
			lon2: 13.0657,
			// Approx 26 km
			expected: 26.0,
			epsilon:  1.0,
		},
		{
			name:     "North Pole to South Pole",
			lat1:     90.0,
			lon1:     0.0,
			lat2:     -90.0,
			lon2:     0.0,
			expected: 20003.9, // Half of the meridian
			epsilon:  50.0,    // Larger tolerance due to sphere model
		},
		{
			name:     "Equator 1 degree diff",
			lat1:     0.0,
			lon1:     0.0,
			lat2:     0.0,
			lon2:     1.0,
			expected: 111.19, // ~111 km
			epsilon:  0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)

			// Check with tolerance
			diff := math.Abs(got - tt.expected)
			assert.True(t, diff <= tt.epsilon,
				"Expected distance ~%.2f km, got %.2f km (diff %.4f > epsilon %.4f)",
				tt.expected, got, diff, tt.epsilon)
		})
	}
}
