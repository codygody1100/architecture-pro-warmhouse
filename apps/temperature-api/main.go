package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

type temperatureResponse struct {
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	Timestamp   time.Time `json:"timestamp"`
	Location    string    `json:"location"`
	Status      string    `json:"status"`
	SensorID    string    `json:"sensor_id"`
	SensorType  string    `json:"sensor_type"`
	Description string    `json:"description"`
}

func newHandler() http.Handler {
	locations := map[string]string{"1": "Living Room", "2": "Bedroom", "3": "Kitchen"}
	lastValues := make(map[string]int)
	var mu sync.Mutex

	temperature := func(w http.ResponseWriter, r *http.Request) {
		sensorID := r.PathValue("id")
		location := r.URL.Query().Get("location")
		if location == "" {
			location = locations[sensorID]
			if location == "" {
				location = "Unknown"
			}
		}
		if sensorID == "" {
			sensorID = "0"
			for id, name := range locations {
				if name == location {
					sensorID = id
					break
				}
			}
		}

		// Keep consecutive readings for the same sensor different after rounding.
		key := sensorID + ":" + location
		mu.Lock()
		value := 1500 + rand.Intn(1501)
		for value == lastValues[key] {
			value = 1500 + rand.Intn(1501)
		}
		lastValues[key] = value
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(temperatureResponse{
			Value:       float64(value) / 100,
			Unit:        "°C",
			Timestamp:   time.Now().UTC(),
			Location:    location,
			Status:      "active",
			SensorID:    sensorID,
			SensorType:  "temperature",
			Description: "Simulated temperature measurement",
		})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /temperature", temperature)
	mux.HandleFunc("GET /temperature/{id}", temperature)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	return mux
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           newHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("Temperature API listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
