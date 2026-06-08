package routes

import (
	"cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/nexus_game/handlers"
	"cgwm/battle/internal/nexus_game/models"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func RegisterRoutes(router *gin.Engine, database *gorm.DB) {
	redis := cache.NewRedisServiceFromEnv()
	health := handlers.NewHealthHandler(database, redis)
	bootstrap := handlers.NewBootstrapHandler(database)
	avatar := handlers.NewAvatarHandler(database)
	factionH := handlers.NewFactionHandler(database)
	companionH := handlers.NewIACompanionHandler(database)

	// Auto migrate models (inside nexus_game only)
	database.AutoMigrate(&models.Avatar{}, &models.Faction{}, &models.IACompanion{})

	// Serve persistent assets (avatars etc.) - mounted via volume in compose.prod.yml
	// Use env vars so user can configure the persistent path in Dokploy (Persistent Storage)
	assetDir := getEnv("NEXUS_ASSET_DIR", "/nexus_game/assets/avatar")
	assetBase := getEnv("NEXUS_ASSET_BASE_URL", "/nexus_game/assets/avatar")
	router.Static(assetBase, assetDir)

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

	// Factions (same principle as avatars)
	group.GET("/factions", factionH.List)
	group.POST("/factions", factionH.Create)
	group.PUT("/factions/:id", factionH.Update)
	group.DELETE("/factions/:id", factionH.Delete)

	// IA Companions (player's AI agents/companions)
	group.GET("/ia-companions", companionH.List)
	group.POST("/ia-companions", companionH.Create)
	group.PUT("/ia-companions/:id", companionH.Update)
	group.DELETE("/ia-companions/:id", companionH.Delete)
}
