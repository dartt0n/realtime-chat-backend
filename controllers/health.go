package controllers

import "github.com/gin-gonic/gin"

// HealthController represents a controller for health check endpoints
type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

// Health handles the health check endpoint and returns a 200 OK response
// with a status message indicating the service is healthy
func (ctrl HealthController) Health(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}
