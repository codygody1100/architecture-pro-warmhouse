package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"smarthome/models"
)

// APIError keeps downstream failures distinct from a missing device.
type APIError struct {
	Status int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("service request failed with HTTP %d", e.Status)
}

func requestJSON(ctx context.Context, client *http.Client, method, url string, body, result any) error {
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, url, &payload)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return &APIError{Status: http.StatusServiceUnavailable}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return &APIError{Status: response.StatusCode}
	}
	if result != nil {
		if err := json.NewDecoder(response.Body).Decode(result); err != nil {
			return &APIError{Status: http.StatusBadGateway}
		}
	}
	return nil
}

// DeviceService implements the same operations as the legacy database through HTTP.
type DeviceService struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewDeviceService(baseURL string) *DeviceService {
	return &DeviceService{
		BaseURL: strings.TrimRight(baseURL, "/"), HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *DeviceService) GetSensors(ctx context.Context) ([]models.Sensor, error) {
	sensors := []models.Sensor{}
	err := requestJSON(ctx, s.HTTPClient, http.MethodGet, s.BaseURL+"/devices", nil, &sensors)
	return sensors, err
}

func (s *DeviceService) GetSensorByID(ctx context.Context, id int) (models.Sensor, error) {
	var sensor models.Sensor
	err := requestJSON(ctx, s.HTTPClient, http.MethodGet, fmt.Sprintf("%s/devices/%d", s.BaseURL, id), nil, &sensor)
	return sensor, err
}

func (s *DeviceService) CreateSensor(ctx context.Context, input models.SensorCreate) (models.Sensor, error) {
	var sensor models.Sensor
	err := requestJSON(ctx, s.HTTPClient, http.MethodPost, s.BaseURL+"/devices", input, &sensor)
	return sensor, err
}

func (s *DeviceService) UpdateSensor(ctx context.Context, id int, input models.SensorUpdate) (models.Sensor, error) {
	var sensor models.Sensor
	err := requestJSON(ctx, s.HTTPClient, http.MethodPut, fmt.Sprintf("%s/devices/%d", s.BaseURL, id), input, &sensor)
	return sensor, err
}

func (s *DeviceService) DeleteSensor(ctx context.Context, id int) error {
	return requestJSON(ctx, s.HTTPClient, http.MethodDelete, fmt.Sprintf("%s/devices/%d", s.BaseURL, id), nil, nil)
}

func (s *DeviceService) UpdateSensorValue(ctx context.Context, id int, value float64, status string) error {
	input := map[string]any{"value": value, "status": status}
	return requestJSON(ctx, s.HTTPClient, http.MethodPatch, fmt.Sprintf("%s/devices/%d/value", s.BaseURL, id), input, nil)
}
