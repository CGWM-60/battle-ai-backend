package cache

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	service := NewRedisServiceFromEnv()

	router.GET("/api/nexus-game/redis/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, service.Status(c.Request.Context()))
	})

	router.GET("/api/nexus-game/redis/health", func(c *gin.Context) {
		status := service.Status(c.Request.Context())
		if status.Enabled && !status.Connected {
			c.JSON(http.StatusServiceUnavailable, status)
			return
		}
		c.JSON(http.StatusOK, status)
	})
}
