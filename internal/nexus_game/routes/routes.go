package routes

import (
	"cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/nexus_game/handlers"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(router *gin.Engine, database *gorm.DB) {
	redis := cache.NewRedisServiceFromEnv()
	health := handlers.NewHealthHandler(database, redis)

	group := router.Group("/api/nexus-game")
	group.GET("/health", health.Health)
	group.GET("/debug/status", health.DebugStatus)
}
