package routes

import (
	"cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/nexus_game/handlers"
	"cgwm/battle/internal/nexus_game/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(router *gin.Engine, database *gorm.DB) {
	redis := cache.NewRedisServiceFromEnv()
	health := handlers.NewHealthHandler(database, redis)
	bootstrap := handlers.NewBootstrapHandler(database)
	avatar := handlers.NewAvatarHandler(database)

	// Auto migrate avatar model (inside nexus_game only)
	database.AutoMigrate(&models.Avatar{})

	// Serve persistent assets (avatars etc.) - mounted via volume in compose.prod.yml
	// Path chosen to match user request: /nexus_game/assets
	router.Static("/nexus_game/assets", "/nexus_game/assets")

	group := router.Group("/api/nexus-game")
	group.GET("/health", health.Health)
	group.GET("/debug/status", health.DebugStatus)

	// Etape 1 - Bootstrap endpoint (chargement assets, quêtes, infos joueur, etc.)
	// Pour l'instant : simulation avec délai de 10s + TODO pour l'implémentation réelle.
	group.GET("/bootstrap", bootstrap.Load)

	// Etape 2 - Avatar management
	// POST multipart: name + image (converted to webp on server)
	group.POST("/assets/avatar", avatar.Upload)
	group.GET("/assets/avatars", avatar.List)
	group.PUT("/assets/avatars/:id", avatar.Update)
	group.DELETE("/assets/avatars/:id", avatar.Delete)
}
