package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Reading struct {
	ID         int       `json:"id,omitempty"`
	DeviceID   int       `json:"device_id"`
	Metric     string    `json:"metric"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	RecordedAt time.Time `json:"recorded_at"`
}

type TelemetryService struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewTelemetryService(baseURL string) *TelemetryService {
	return &TelemetryService{
		BaseURL: strings.TrimRight(baseURL, "/"), HTTPClient: &http.Client{Timeout: 3 * time.Second},
	}
}

func (s *TelemetryService) Save(ctx context.Context, reading Reading) error {
	return requestJSON(ctx, s.HTTPClient, http.MethodPost, s.BaseURL+"/readings", reading, nil)
}

func (s *TelemetryService) History(ctx context.Context, deviceID, limit int) ([]Reading, error) {
	readings := []Reading{}
	url := fmt.Sprintf("%s/readings?device_id=%d&limit=%d", s.BaseURL, deviceID, limit)
	err := requestJSON(ctx, s.HTTPClient, http.MethodGet, url, nil, &readings)
	return readings, err
}
