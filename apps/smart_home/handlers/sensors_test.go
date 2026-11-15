package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"smarthome/models"
	"smarthome/services"

	"github.com/gin-gonic/gin"
)

func TestSensorsDelegateAndSaveTelemetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sensor := models.Sensor{ID: 7, Name: "Kitchen", Type: models.Temperature, Unit: "°C"}
	var saved []services.Reading
	deviceCalls := 0
	telemetryDown := false
	deviceDown := false
	zeroUpdated := false

	deviceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deviceCalls++
		if deviceDown {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /devices/404":
			w.WriteHeader(http.StatusNotFound)
		case "GET /devices":
			json.NewEncoder(w).Encode([]models.Sensor{sensor})
		case "GET /devices/7":
			json.NewEncoder(w).Encode(sensor)
		case "PATCH /devices/7/value":
			var input struct {
				Value float64 `json:"value"`
			}
			json.NewDecoder(r.Body).Decode(&input)
			zeroUpdated = input.Value == 0
			json.NewEncoder(w).Encode(map[string]string{"message": "ok"})
		default:
			t.Errorf("unexpected device request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer deviceServer.Close()
	temperatureServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(services.TemperatureResponse{
			Value: 23.5, Unit: "°C", Timestamp: time.Now().UTC(), Status: "active",
		})
	}))
	defer temperatureServer.Close()
	telemetryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if telemetryDown {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if r.Method == http.MethodPost {
			var reading services.Reading
			if err := json.NewDecoder(r.Body).Decode(&reading); err != nil {
				t.Error(err)
			}
			saved = append(saved, reading)
			w.WriteHeader(http.StatusCreated)
		} else {
			if r.URL.Query().Get("device_id") != "7" {
				t.Error("incorrect history device")
			}
			json.NewEncoder(w).Encode(saved)
		}
	}))
	defer telemetryServer.Close()
	router := gin.New()
	NewSensorHandler(services.NewDeviceService(deviceServer.URL),
		services.NewTemperatureService(temperatureServer.URL),
		services.NewTelemetryService(telemetryServer.URL)).RegisterRoutes(router.Group("/api/v1"))
	call := func(method, path, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, "/api/v1/sensors"+path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != status {
			t.Fatalf("%s %s: got %d: %s", method, path, response.Code, response.Body)
		}
		return response
	}
	response := call("GET", "", "", 200)
	var sensors []models.Sensor
	if err := json.Unmarshal(response.Body.Bytes(), &sensors); err != nil {
		t.Fatal(err)
	}
	if len(sensors) != 1 || sensors[0].Value != 23.5 || sensors[0].Status != "active" {
		t.Fatalf("unexpected sensors: %+v", sensors)
	}
	if len(saved) != 1 || saved[0].DeviceID != 7 || saved[0].Value != 23.5 {
		t.Fatalf("unexpected telemetry: %+v", saved)
	}
	call("GET", "/7/telemetry", "", 200)
	call("GET", "/404", "", 404)
	call("PATCH", "/7/value", `{"value":0,"status":"active"}`, 200)
	if !zeroUpdated {
		t.Fatal("zero must be accepted as a valid reading")
	}
	call("PATCH", "/7/value", `{"status":"active"}`, 400)
	telemetryDown = true
	call("GET", "/7", "", 200)
	call("GET", "/7/telemetry", "", 503)
	deviceDown = true
	call("GET", "/7", "", 503)
	if deviceCalls == 0 {
		t.Fatal("device API was not called")
	}
}
