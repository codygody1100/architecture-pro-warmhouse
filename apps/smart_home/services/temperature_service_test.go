package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetTemperatureEscapesLocation(t *testing.T) {
	for _, location := range []string{"Living Room", "Кухня & столовая"} {
		t.Run(location, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("location"); got != location {
					t.Errorf("location = %q, want %q", got, location)
				}
				json.NewEncoder(w).Encode(TemperatureResponse{
					Value: 21.5, Unit: "°C", Timestamp: time.Now().UTC(), Location: location, Status: "active",
				})
			}))
			defer server.Close()
			service := NewTemperatureService(server.URL)
			reading, err := service.GetTemperature(location)
			if err != nil {
				t.Fatal(err)
			}
			if reading.Location != location || reading.Value != 21.5 {
				t.Fatalf("unexpected response: %+v", reading)
			}
		})
	}
}
