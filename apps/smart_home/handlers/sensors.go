package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"smarthome/models"
	"smarthome/services"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type SensorStore interface {
	GetSensors(context.Context) ([]models.Sensor, error)
	GetSensorByID(context.Context, int) (models.Sensor, error)
	CreateSensor(context.Context, models.SensorCreate) (models.Sensor, error)
	UpdateSensor(context.Context, int, models.SensorUpdate) (models.Sensor, error)
	DeleteSensor(context.Context, int) error
	UpdateSensorValue(context.Context, int, float64, string) error
}

// SensorHandler handles sensor-related requests
type SensorHandler struct {
	Store              SensorStore
	TemperatureService *services.TemperatureService
	TelemetryService   *services.TelemetryService
}

// NewSensorHandler creates a new SensorHandler
func NewSensorHandler(store SensorStore, temperatureService *services.TemperatureService, telemetryService *services.TelemetryService) *SensorHandler {
	return &SensorHandler{
		Store:              store,
		TemperatureService: temperatureService,
		TelemetryService:   telemetryService,
	}
}

// RegisterRoutes registers the sensor routes
func (h *SensorHandler) RegisterRoutes(router *gin.RouterGroup) {
	sensors := router.Group("/sensors")
	{
		sensors.GET("", h.GetSensors)
		sensors.GET("/:id", h.GetSensorByID)
		sensors.GET("/:id/telemetry", h.GetTelemetry)
		sensors.POST("", h.CreateSensor)
		sensors.PUT("/:id", h.UpdateSensor)
		sensors.DELETE("/:id", h.DeleteSensor)
		sensors.PATCH("/:id/value", h.UpdateSensorValue)
		sensors.GET("/temperature/:location", h.GetTemperatureByLocation)
	}
}

// GetSensors handles GET /api/v1/sensors
func (h *SensorHandler) GetSensors(c *gin.Context) {
	sensors, err := h.Store.GetSensors(c.Request.Context())
	if err != nil {
		respondStoreError(c, err)
		return
	}

	for i := range sensors {
		h.refreshTemperature(c, &sensors[i])
	}

	c.JSON(http.StatusOK, sensors)
}

// GetSensorByID handles GET /api/v1/sensors/:id
func (h *SensorHandler) GetSensorByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sensor ID"})
		return
	}

	sensor, err := h.Store.GetSensorByID(c.Request.Context(), id)
	if err != nil {
		respondStoreError(c, err)
		return
	}

	h.refreshTemperature(c, &sensor)

	c.JSON(http.StatusOK, sensor)
}

// GetTemperatureByLocation handles GET /api/v1/sensors/temperature/:location
func (h *SensorHandler) GetTemperatureByLocation(c *gin.Context) {
	location := c.Param("location")
	if location == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Location is required"})
		return
	}

	// Fetch temperature data from the external API
	tempData, err := h.TemperatureService.GetTemperature(location)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to fetch temperature data: %v", err),
		})
		return
	}

	// Return the temperature data
	c.JSON(http.StatusOK, gin.H{
		"location":    tempData.Location,
		"value":       tempData.Value,
		"unit":        tempData.Unit,
		"status":      tempData.Status,
		"timestamp":   tempData.Timestamp,
		"description": tempData.Description,
	})
}

// CreateSensor handles POST /api/v1/sensors
func (h *SensorHandler) CreateSensor(c *gin.Context) {
	var sensorCreate models.SensorCreate
	if err := c.ShouldBindJSON(&sensorCreate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sensor, err := h.Store.CreateSensor(c.Request.Context(), sensorCreate)
	if err != nil {
		respondStoreError(c, err)
		return
	}

	h.refreshTemperature(c, &sensor)

	c.JSON(http.StatusCreated, sensor)
}

// UpdateSensor handles PUT /api/v1/sensors/:id
func (h *SensorHandler) UpdateSensor(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sensor ID"})
		return
	}

	var sensorUpdate models.SensorUpdate
	if err := c.ShouldBindJSON(&sensorUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sensor, err := h.Store.UpdateSensor(c.Request.Context(), id, sensorUpdate)
	if err != nil {
		respondStoreError(c, err)
		return
	}

	c.JSON(http.StatusOK, sensor)
}

// DeleteSensor handles DELETE /api/v1/sensors/:id
func (h *SensorHandler) DeleteSensor(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sensor ID"})
		return
	}

	err = h.Store.DeleteSensor(c.Request.Context(), id)
	if err != nil {
		respondStoreError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sensor deleted successfully"})
}

// UpdateSensorValue handles PATCH /api/v1/sensors/:id/value
func (h *SensorHandler) UpdateSensorValue(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sensor ID"})
		return
	}

	var request struct {
		Value  *float64 `json:"value" binding:"required"`
		Status string   `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.Store.UpdateSensorValue(c.Request.Context(), id, *request.Value, request.Status)
	if err != nil {
		respondStoreError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sensor value updated successfully"})
}

// Readings are collected on demand; a telemetry outage must not block sensor reads.
func (h *SensorHandler) refreshTemperature(c *gin.Context, sensor *models.Sensor) {
	if sensor.Type != models.Temperature {
		return
	}
	reading, err := h.TemperatureService.GetTemperatureByID(strconv.Itoa(sensor.ID))
	if err != nil {
		log.Printf("Failed to fetch temperature for sensor %d: %v", sensor.ID, err)
		return
	}
	sensor.Value = reading.Value
	sensor.Status = reading.Status
	sensor.LastUpdated = reading.Timestamp
	if h.TelemetryService != nil {
		err = h.TelemetryService.Save(c.Request.Context(), services.Reading{
			DeviceID: sensor.ID, Metric: string(sensor.Type), Value: reading.Value,
			Unit: reading.Unit, RecordedAt: reading.Timestamp,
		})
		if err != nil {
			log.Printf("Failed to save telemetry for sensor %d: %v", sensor.ID, err)
		}
	}
}

func (h *SensorHandler) GetTelemetry(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sensor ID"})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil || limit < 1 || limit > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Limit must be from 1 to 1000"})
		return
	}
	if h.TelemetryService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Telemetry service is not configured"})
		return
	}
	if _, err := h.Store.GetSensorByID(c.Request.Context(), id); err != nil {
		respondStoreError(c, err)
		return
	}
	readings, err := h.TelemetryService.History(c.Request.Context(), id, limit)
	if err != nil {
		respondStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, readings)
}

func respondStoreError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	var apiError *services.APIError
	if errors.As(err, &apiError) {
		status = apiError.Status
	} else if errors.Is(err, pgx.ErrNoRows) {
		status = http.StatusNotFound
	}
	c.JSON(status, gin.H{"error": err.Error()})
}
