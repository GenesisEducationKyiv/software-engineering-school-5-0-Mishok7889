package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type WeatherResponse struct {
	Current Current `json:"current"`
}

type Current struct {
	TempC     float64   `json:"temp_c"`
	Humidity  float64   `json:"humidity"`
	Condition Condition `json:"condition"`
}

type Condition struct {
	Text string `json:"text"`
}

var weatherData = map[string]WeatherResponse{
	"london": {
		Current: Current{
			TempC:    15.0,
			Humidity: 76.0,
			Condition: Condition{Text: "Partly cloudy"},
		},
	},
	"paris": {
		Current: Current{
			TempC:    18.0,
			Humidity: 68.0,
			Condition: Condition{Text: "Clear"},
		},
	},
	"berlin": {
		Current: Current{
			TempC:    12.0,
			Humidity: 82.0,
			Condition: Condition{Text: "Overcast"},
		},
	},
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/current.json", func(c *gin.Context) {
		city := strings.ToLower(c.Query("q"))
		key := c.Query("key")

		if key == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "API key required"})
			return
		}

		if city == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "City parameter required"})
			return
		}

		if city == "servererror" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		if city == "timeout" {
			c.Header("Connection", "close")
			c.AbortWithStatus(http.StatusRequestTimeout)
			return
		}

		weather, exists := weatherData[city]
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "City not found"})
			return
		}

		c.JSON(http.StatusOK, weather)
	})

	slog.Info("Mock Weather API server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
