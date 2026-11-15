package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTemperatureRoutes(t *testing.T) {
	handler := newHandler()
	tests := []struct {
		path, location, sensorID string
	}{
		{"/temperature?location=Living%20Room", "Living Room", "1"},
		{"/temperature/2", "Bedroom", "2"},
		{"/temperature?location=Kitchen", "Kitchen", "3"},
		{"/temperature?location=Office", "Office", "0"},
		{"/temperature/99", "Unknown", "99"},
		{"/temperature", "Unknown", "0"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.Code)
			}
			var reading temperatureResponse
			if err := json.Unmarshal(response.Body.Bytes(), &reading); err != nil {
				t.Fatal(err)
			}
			if reading.Location != tt.location || reading.SensorID != tt.sensorID {
				t.Fatalf("location/id = %q/%q, want %q/%q", reading.Location, reading.SensorID, tt.location, tt.sensorID)
			}
			if reading.Value < 15 || reading.Value > 30 || reading.Unit != "°C" || reading.Status != "active" || reading.SensorType != "temperature" {
				t.Fatalf("unexpected reading: %+v", reading)
			}
			if reading.Timestamp.IsZero() || time.Since(reading.Timestamp) > time.Minute {
				t.Fatalf("invalid measurement time: %v", reading.Timestamp)
			}
			if response.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("temperature responses must not be cached")
			}
		})
	}
}

func TestConsecutiveTemperaturesDiffer(t *testing.T) {
	handler := newHandler()
	previous := -1.0
	for i := 0; i < 100; i++ {
		path := "/temperature/1"
		if i%2 == 0 {
			path = "/temperature?location=Living%20Room"
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		var reading temperatureResponse
		if err := json.Unmarshal(response.Body.Bytes(), &reading); err != nil {
			t.Fatal(err)
		}
		if reading.Value == previous {
			t.Fatalf("consecutive reading repeated: %v", reading.Value)
		}
		previous = reading.Value
	}
}

func TestHealthAndUnknownRoute(t *testing.T) {
	handler := newHandler()
	for _, tt := range []struct {
		method, path string
		status       int
	}{
		{http.MethodGet, "/health", http.StatusOK},
		{http.MethodPost, "/temperature", http.StatusMethodNotAllowed},
		{http.MethodGet, "/missing", http.StatusNotFound},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
		if response.Code != tt.status {
			t.Errorf("%s %s: status = %d, want %d", tt.method, tt.path, response.Code, tt.status)
		}
	}
}
