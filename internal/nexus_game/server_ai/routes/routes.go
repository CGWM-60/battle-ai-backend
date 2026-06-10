package routes

import (
	"context"

	"cgwm/battle/internal/nexus_game/server_ai/handlers"
	"cgwm/battle/internal/nexus_game/server_ai/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AutoMigrate(database *gorm.DB) error {
	return services.NewService(database).AutoMigrate()
}

func SeedDefaults(database *gorm.DB) error {
	return services.NewService(database).SeedDefaults(context.Background())
}

func Register(group *gin.RouterGroup, database *gorm.DB) {
	h := handlers.NewHandler(database)

	group.GET("/ai-server/cities", h.PublicCities)
	group.GET("/ai-server/cities/:id", h.City)
	group.GET("/ai-server/threat-level", h.ThreatLevel)
	group.GET("/ai-server/attacks", h.PublicAttacks)
	group.GET("/ai-server/attacks/:id", h.PublicAttack)
	group.GET("/ai-server/daily-broadcast", h.DailyBroadcast)

	group.GET("/seasonal-events/active", h.SeasonalActive)
	group.GET("/seasonal-events/upcoming", h.SeasonalUpcoming)
	group.GET("/seasonal-events/:id", h.SeasonalGet)

	admin := group.Group("/admin")
	admin.GET("/ai-server/dashboard", h.Dashboard)
	admin.POST("/ai-server/worlds/:worldId/ensure-cities", h.EnsureWorldCities)
	admin.GET("/ai-server/cities", h.AdminCities)
	admin.GET("/ai-server/cities/:id", h.City)
	admin.PUT("/ai-server/cities/:id", h.UpdateCity)
	admin.DELETE("/ai-server/cities/:id", h.DeleteCity)

	admin.GET("/ai-server/memory", h.GlobalMemory)
	admin.GET("/ai-server/player-memory", h.PlayerMemory)
	admin.DELETE("/ai-server/player-memory/:id", h.DeletePlayerMemory)

	admin.GET("/ai-server/attacks", h.AdminAttacks)
	admin.POST("/ai-server/attacks/schedule", h.ScheduleAttack)
	admin.POST("/ai-server/attacks/:id/cancel", h.CancelAttack)
	admin.POST("/ai-server/attacks/:id/resolve", h.ResolveAttack)
	admin.DELETE("/ai-server/attacks/:id", h.DeleteAttack)

	admin.GET("/ai-server/sabotages", h.Sabotages)
	admin.POST("/ai-server/sabotages/:id/cancel", h.CancelSabotage)
	admin.GET("/ai-server/espionage", h.Espionage)
	admin.DELETE("/ai-server/espionage/:id", h.DeleteEspionage)

	admin.GET("/ai-server/broadcasts", h.Broadcasts)
	admin.POST("/ai-server/broadcasts/generate", h.GenerateBroadcast)
	admin.POST("/ai-server/daily-broadcast/generate", h.GenerateBroadcast)
	admin.PUT("/ai-server/broadcasts/:id", h.UpdateBroadcast)
	admin.POST("/ai-server/broadcasts/:id/publish", h.PublishBroadcast)
	admin.POST("/ai-server/daily-broadcast/publish", h.PublishBroadcast)
	admin.DELETE("/ai-server/broadcasts/:id", h.DeleteBroadcast)

	admin.GET("/ai-server/prompts", h.Prompts)
	admin.POST("/ai-server/prompts", h.CreatePrompt)
	admin.PUT("/ai-server/prompts/:id", h.UpdatePrompt)
	admin.DELETE("/ai-server/prompts/:id", h.DeletePrompt)
	admin.POST("/ai-server/prompts/:id/test", h.TestPrompt)
	admin.POST("/ai-server/prompts/seed", h.SeedPrompts)

	admin.GET("/ai-server/call-logs", h.CallLogs)
	admin.GET("/ai-server/costs", h.Costs)

	admin.GET("/seasonal-events/proposals", h.SeasonalProposals)
	admin.POST("/seasonal-events/propose-by-ai", h.ProposeSeasonalByAI)
	admin.GET("/seasonal-events", h.SeasonalList)
	admin.GET("/seasonal-events/:id", h.SeasonalGet)
	admin.PUT("/seasonal-events/:id", h.SeasonalUpdate)
	admin.DELETE("/seasonal-events/:id", h.SeasonalDelete)
	admin.POST("/seasonal-events/:id/approve", h.SeasonalTransition("approved"))
	admin.POST("/seasonal-events/:id/reject", h.SeasonalTransition("rejected"))
	admin.POST("/seasonal-events/:id/schedule", h.SeasonalTransition("scheduled"))
	admin.POST("/seasonal-events/:id/start", h.SeasonalTransition("active"))
	admin.POST("/seasonal-events/:id/end", h.SeasonalTransition("ended"))
	admin.POST("/seasonal-events/:id/archive", h.SeasonalTransition("archived"))
}
