package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/scheduler"
	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *Server) registerGameAdminAPI(api *gin.RouterGroup) {
	game := api.Group("/game")
	world := service.NewWorldGameService(s.db)
	game.GET("/dashboard", s.gameDashboardAPI)
	game.GET("/stats", s.gameDashboardAPI)
	game.GET("/worlds", s.gameListWorldsAPI)
	game.POST("/worlds", s.gameCreateWorldAPI)
	game.POST("/worlds/reconcile-counts", s.gameReconcileWorldCountsAPI(world))
	game.POST("/worlds/archive-empty", s.gameArchiveEmptyWorldsAPI(world))
	game.GET("/worlds/:id", s.gameGetWorldAPI)
	game.PATCH("/worlds/:id", s.gamePatchModelAPI(&models.World{}, "world"))
	game.POST("/worlds/:id/simulate", s.gameSimulateWorldAPI(world))
	game.GET("/worlds/:id/routine", s.gameWorldRoutineAPI(world, false))
	game.POST("/worlds/:id/routine/generate", s.gameWorldRoutineAPI(world, true))
	game.POST("/world/simulate", s.gameSimulateWorldAPI(world))
	game.POST("/world/routine/generate", s.gameWorldRoutineAPI(world, true))
	game.POST("/world/message", s.gameCreateModelAPI(&models.DailyAIMessage{}, "ai_message"))
	game.GET("/worlds/:id/continents", s.gameWorldContinentsAPI)
	game.GET("/continents", s.gameListModelAPI(&[]models.Continent{}, "`world_id` ASC, `index` ASC"))
	game.GET("/continents/:id", s.gameGetModelAPI(&models.Continent{}))
	game.PATCH("/continents/:id", s.gamePatchModelAPI(&models.Continent{}, "continent"))
	game.POST("/continents/:id/simulate", s.gameSimulateWorldAPI(world))

	game.GET("/players", s.gamePlayersAPI)
	game.GET("/players/:id", s.gameGetModelAPI(&models.Users{}))
	game.GET("/players/:id/save", s.gamePlayerSaveAPI)
	game.PATCH("/players/:id/save", s.gamePatchPlayerSaveAPI)
	game.POST("/players/:id/resync", s.gameResyncPlayerAPI(world))
	game.GET("/player-metrics", s.gameListModelAPI(&[]models.PlayerWorldMetric{}, "generated_at DESC"))
	game.POST("/players/:id/metrics/recalculate", s.gameRecalculatePlayerMetricsAPI(world))

	game.GET("/events", s.gameListModelAPI(&[]models.GameEvent{}, "starts_at DESC"))
	game.POST("/events", s.gameCreateEventAPI(world))
	game.GET("/events/:id", s.gameGetModelAPI(&models.GameEvent{}))
	game.PATCH("/events/:id", s.gamePatchEventAPI(world))
	game.DELETE("/events/:id", s.gameDeleteModelAPI(&models.GameEvent{}, "event"))
	game.POST("/events/:id/force-start", s.gamePatchStatusAPI(&models.GameEvent{}, "active"))
	game.POST("/events/:id/force-end", s.gamePatchStatusAPI(&models.GameEvent{}, "finished"))
	game.POST("/events/generate-ai", s.gameSimulateWorldAPI(world))

	game.GET("/conflicts", s.gameListModelAPI(&[]models.Conflict{}, "starts_at DESC"))
	game.POST("/conflicts", s.gameCreateConflictAPI(world))
	game.GET("/conflicts/:id", s.gameGetModelAPI(&models.Conflict{}))
	game.PATCH("/conflicts/:id", s.gamePatchModelAPI(&models.Conflict{}, "conflict"))
	game.POST("/conflicts/:id/resolve", s.gameResolveConflictAPI(world))
	game.POST("/conflicts/generate-ai", s.gameSimulateWorldAPI(world))

	game.GET("/weather", s.gameListModelAPI(&[]models.WeatherEvent{}, "starts_at DESC"))
	game.POST("/weather", s.gameCreateModelAPI(&models.WeatherEvent{}, "weather"))
	game.GET("/weather/:id", s.gameGetModelAPI(&models.WeatherEvent{}))
	game.PATCH("/weather/:id", s.gamePatchModelAPI(&models.WeatherEvent{}, "weather"))
	game.POST("/weather/:id/end", s.gameEndWeatherAPI)
	game.POST("/weather/generate-ai", s.gameSimulateWorldAPI(world))

	game.GET("/ai/messages", s.gameListModelAPI(&[]models.DailyAIMessage{}, "created_at DESC"))
	game.POST("/ai/messages", s.gameCreateModelAPI(&models.DailyAIMessage{}, "ai_message"))
	game.POST("/ai/messages/generate", s.gameSimulateWorldAPI(world))
	game.POST("/ai/messages/send", s.gameCreateModelAPI(&models.DailyAIMessage{}, "ai_message"))
	game.GET("/ai/decisions", s.gameListModelAPI(&[]models.AIWorldDecision{}, "created_at DESC"))
	game.GET("/ai/decisions/:id", s.gameGetModelAPI(&models.AIWorldDecision{}))
	game.PATCH("/ai/decisions/:id", s.gamePatchModelAPI(&models.AIWorldDecision{}, "ai_decision"))
	game.POST("/ai/decisions/:id/enable", s.gameDecisionActivationAPI(true))
	game.POST("/ai/decisions/:id/disable", s.gameDecisionActivationAPI(false))
	game.POST("/ai/decisions/:id/replay-dry-run", s.gameDecisionDryRunAPI)
	game.GET("/ai/routines", s.gameListModelAPI(&[]models.WorldRoutineSnapshot{}, "created_at DESC"))
	game.GET("/ai/cron", s.gameWorldCronAPI)
	game.POST("/ai/cron/enable", s.gameWorldCronToggleAPI(true))
	game.POST("/ai/cron/disable", s.gameWorldCronToggleAPI(false))
	game.GET("/ai/providers", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"providers": world.AIProviderStatuses()})
	})
	game.POST("/ai/providers/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"configured": world.AIProviderStatuses()})
	})
	game.GET("/ai/factions", s.gameListModelAPI(&[]models.AIWorldFaction{}, "id DESC"))
	game.POST("/ai/factions", s.gameCreateModelAPI(&models.AIWorldFaction{}, "ai_faction"))
	game.GET("/ai/factions/:id", s.gameGetModelAPI(&models.AIWorldFaction{}))
	game.PATCH("/ai/factions/:id", s.gamePatchModelAPI(&models.AIWorldFaction{}, "ai_faction"))
	game.DELETE("/ai/factions/:id", s.gameDeleteModelAPI(&models.AIWorldFaction{}, "ai_faction"))

	game.GET("/guilds", s.gameListModelAPI(&[]models.Guild{}, "id DESC"))
	game.GET("/guilds/:id", s.gameGetGuildAPI)
	game.PATCH("/guilds/:id", s.gamePatchModelAPI(&models.Guild{}, "guild"))
	game.DELETE("/guilds/:id", s.gameDeleteModelAPI(&models.Guild{}, "guild"))
	game.GET("/guilds/:id/members", s.gameGuildMembersAPI)
	game.PATCH("/guilds/:id/members/:playerId", s.gameGuildRoleAPI)
	game.DELETE("/guilds/:id/members/:playerId", s.gameGuildKickAPI)

	game.GET("/chat/messages", s.gameListModelAPI(&[]models.ChatMessage{}, "created_at DESC"))
	game.DELETE("/chat/messages/:id", s.gameModerateChatAPI(true))
	game.POST("/chat/messages/:id/restore", s.gameModerateChatAPI(false))

	game.GET("/buildings", s.gameListModelAPI(&[]models.BuildingDefinition{}, "sort_order ASC, id ASC"))
	game.POST("/buildings", s.gameCreateModelAPI(&models.BuildingDefinition{}, "building"))
	game.GET("/buildings/:id", s.gameGetBuildingAPI)
	game.PATCH("/buildings/:id", s.gamePatchModelAPI(&models.BuildingDefinition{}, "building"))
	game.DELETE("/buildings/:id", s.gameDeleteModelAPI(&models.BuildingDefinition{}, "building"))
	game.GET("/resources", s.gameListModelAPI(&[]models.ResourceDefinition{}, "sort_order ASC, id ASC"))
	game.POST("/resources", s.gameCreateModelAPI(&models.ResourceDefinition{}, "resource"))
	game.GET("/resources/:id", s.gameGetModelAPI(&models.ResourceDefinition{}))
	game.PATCH("/resources/:id", s.gamePatchModelAPI(&models.ResourceDefinition{}, "resource"))
	game.DELETE("/resources/:id", s.gameDeleteModelAPI(&models.ResourceDefinition{}, "resource"))
	game.GET("/research-trees", s.gameListModelAPI(&[]models.ResearchTreeDefinition{}, "sort_order ASC, id ASC"))
	game.POST("/research-trees", s.gameCreateModelAPI(&models.ResearchTreeDefinition{}, "research_tree"))
	game.GET("/research-trees/:id", s.gameGetResearchTreeAPI)
	game.PATCH("/research-trees/:id", s.gamePatchModelAPI(&models.ResearchTreeDefinition{}, "research_tree"))
	game.DELETE("/research-trees/:id", s.gameDeleteModelAPI(&models.ResearchTreeDefinition{}, "research_tree"))
	game.GET("/research-nodes", s.gameListModelAPI(&[]models.ResearchNodeDefinition{}, "sort_order ASC, id ASC"))
	game.POST("/research-nodes", s.gameCreateModelAPI(&models.ResearchNodeDefinition{}, "research_node"))
	game.GET("/research-nodes/:id", s.gameGetModelAPI(&models.ResearchNodeDefinition{}))
	game.PATCH("/research-nodes/:id", s.gamePatchModelAPI(&models.ResearchNodeDefinition{}, "research_node"))
	game.DELETE("/research-nodes/:id", s.gameDeleteModelAPI(&models.ResearchNodeDefinition{}, "research_node"))
	game.GET("/buildings/:id/assets", s.gameBuildingAssetsAPI)
	game.POST("/buildings/:id/assets", s.gameCreateBuildingAssetAPI(world))
	game.GET("/building-assets", s.gameListModelAPI(&[]models.BuildingAsset{}, "building_definition_id ASC, level ASC"))
	game.PATCH("/building-assets/:assetId", s.gamePatchModelParamAPI(&models.BuildingAsset{}, "assetId", "building_asset"))
	game.DELETE("/building-assets/:assetId", s.gameDeleteModelParamAPI(&models.BuildingAsset{}, "assetId", "building_asset"))
	game.GET("/assets/manifest", s.gameManifestAPI(world))
	game.POST("/assets/manifest/publish", s.gamePublishManifestAPI(world))
	game.GET("/audit", s.gameListModelAPI(&[]models.AdminAuditLog{}, "created_at DESC"))
}

func (s *Server) gameDashboardAPI(c *gin.Context) {
	counts := gin.H{}
	count := func(key string, model any, where string, args ...any) {
		var total int64
		query := s.db.WithContext(c.Request.Context()).Model(model)
		if where != "" {
			query = query.Where(where, args...)
		}
		_ = query.Count(&total).Error
		counts[key] = total
	}
	now := time.Now()
	count("worlds", &models.World{}, "")
	count("continents", &models.Continent{}, "")
	count("players", &models.PlayerSave{}, "")
	count("active24h", &models.PlayerSave{}, "updated_at >= ?", now.Add(-24*time.Hour))
	count("active7d", &models.PlayerSave{}, "updated_at >= ?", now.Add(-7*24*time.Hour))
	count("guilds", &models.Guild{}, "")
	count("activeEvents", &models.GameEvent{}, "status = ?", "active")
	count("activeConflicts", &models.Conflict{}, "status = ?", "active")
	count("activeWeather", &models.WeatherEvent{}, "ends_at >= ?", now)
	count("aiMessagesToday", &models.DailyAIMessage{}, "created_at >= ?", now.Truncate(24*time.Hour))
	count("activeBuildingAssets", &models.BuildingAsset{}, "is_active = ?", true)
	count("chatMessages", &models.ChatMessage{}, "")
	count("aiDecisions", &models.AIWorldDecision{}, "")
	var lastWorld models.World
	_ = s.db.WithContext(c.Request.Context()).Where("last_simulation_at IS NOT NULL").Order("last_simulation_at DESC").First(&lastWorld).Error
	c.JSON(http.StatusOK, gin.H{
		"stats":          counts,
		"lastSimulation": lastWorld.LastSimulationAt,
		"providers":      service.NewWorldGameService(s.db).AIProviderStatuses(),
		"charts":         s.gameDashboardCharts(c, now),
		"generatedAt":    now,
	})
}

type gameChartPoint struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
	Value int64  `json:"value,omitempty"`
}

func (s *Server) gameDashboardCharts(c *gin.Context, now time.Time) gin.H {
	ctx := c.Request.Context()
	chatActivity := make([]gameChartPoint, 0)
	_ = s.db.WithContext(ctx).
		Model(&models.ChatMessage{}).
		Select("channel_type as label, COUNT(*) as count").
		Where("created_at >= ?", now.Add(-7*24*time.Hour)).
		Group("channel_type").
		Order("channel_type ASC").
		Scan(&chatActivity).Error

	playerGrowth := make([]gameChartPoint, 0)
	_ = s.db.WithContext(ctx).
		Model(&models.PlayerSave{}).
		Select("DATE(created_at) as label, COUNT(*) as count").
		Where("created_at >= ?", now.Add(-14*24*time.Hour)).
		Group("DATE(created_at)").
		Order("DATE(created_at) ASC").
		Scan(&playerGrowth).Error

	conflictsByIntensity := make([]gameChartPoint, 0)
	_ = s.db.WithContext(ctx).
		Model(&models.Conflict{}).
		Select("CASE WHEN intensity >= 80 THEN 'critical' WHEN intensity >= 60 THEN 'high' WHEN intensity >= 30 THEN 'medium' ELSE 'low' END as label, COUNT(*) as count").
		Group("label").
		Order("label ASC").
		Scan(&conflictsByIntensity).Error

	weatherBySeverity := make([]gameChartPoint, 0)
	_ = s.db.WithContext(ctx).
		Model(&models.WeatherEvent{}).
		Select("CASE WHEN severity >= 80 THEN 'critical' WHEN severity >= 60 THEN 'high' WHEN severity >= 30 THEN 'medium' ELSE 'low' END as label, COUNT(*) as count").
		Group("label").
		Order("label ASC").
		Scan(&weatherBySeverity).Error

	resources := make([]gameChartPoint, 0)
	var totals struct {
		Food    int64
		Energy  int64
		Credits int64
		Gems    int64
	}
	_ = s.db.WithContext(ctx).
		Model(&models.PlayerSave{}).
		Select("COALESCE(SUM(food),0) as food, COALESCE(SUM(energy),0) as energy, COALESCE(SUM(credits),0) as credits, COALESCE(SUM(gems),0) as gems").
		Scan(&totals).Error
	resources = append(resources,
		gameChartPoint{Label: "food", Value: totals.Food},
		gameChartPoint{Label: "energy", Value: totals.Energy},
		gameChartPoint{Label: "credits", Value: totals.Credits},
		gameChartPoint{Label: "gems", Value: totals.Gems},
	)

	rewardsClaimed := make([]gameChartPoint, 0)
	_ = s.db.WithContext(ctx).
		Model(&models.GameEventClaim{}).
		Select("DATE(created_at) as label, COUNT(*) as count").
		Where("created_at >= ?", now.Add(-14*24*time.Hour)).
		Group("DATE(created_at)").
		Order("DATE(created_at) ASC").
		Scan(&rewardsClaimed).Error

	return gin.H{
		"chatActivity":         chatActivity,
		"playerGrowth":         playerGrowth,
		"conflictsByIntensity": conflictsByIntensity,
		"weatherBySeverity":    weatherBySeverity,
		"resources":            resources,
		"rewardsClaimed":       rewardsClaimed,
	}
}

func (s *Server) gameListWorldsAPI(c *gin.Context) {
	var worlds []models.World
	err := s.db.WithContext(c.Request.Context()).Preload("Continents", func(db *gorm.DB) *gorm.DB {
		return db.Order("`index` ASC")
	}).Order("id DESC").Limit(gameLimit(c)).Find(&worlds).Error
	gameJSON(c, gin.H{"items": worlds}, err)
}

func (s *Server) gameCreateWorldAPI(c *gin.Context) {
	world, err := service.NewWorldGameService(s.db).CreateWorld(c.Request.Context())
	if err == nil {
		s.gameAudit(c, "create", "world", strconv.FormatUint(uint64(world.Id), 10), nil, world)
	}
	gameJSON(c, world, err)
}

func (s *Server) gameReconcileWorldCountsAPI(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := world.ReconcileWorldPopulationCounts(c.Request.Context())
		if err == nil {
			s.gameAudit(c, "reconcile_counts", "world", "all", nil, result)
		}
		gameJSON(c, result, err)
	}
}

func (s *Server) gameArchiveEmptyWorldsAPI(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := world.ArchiveEmptyWorlds(c.Request.Context())
		if err == nil {
			s.gameAudit(c, "archive_empty", "world", "all", nil, result)
		}
		gameJSON(c, result, err)
	}
}

func (s *Server) gameGetWorldAPI(c *gin.Context) {
	id, err := gameParam(c, "id")
	if err != nil {
		gameJSON(c, nil, err)
		return
	}
	var world models.World
	err = s.db.WithContext(c.Request.Context()).Preload("Continents").First(&world, id).Error
	gameJSON(c, world, err)
}

func (s *Server) gameWorldContinentsAPI(c *gin.Context) {
	id, err := gameParam(c, "id")
	if err != nil {
		gameJSON(c, nil, err)
		return
	}
	var items []models.Continent
	err = s.db.WithContext(c.Request.Context()).Where("world_id = ?", id).Order("`index` ASC").Find(&items).Error
	gameJSON(c, gin.H{"items": items}, err)
}

func (s *Server) gameSimulateWorldAPI(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var worldID uint
		if c.Param("id") != "" {
			worldID, _ = gameParam(c, "id")
		}
		cycleType := c.Query("cycleType")
		if cycleType == "" {
			var payload struct {
				CycleType string `json:"cycleType"`
			}
			_ = c.ShouldBindJSON(&payload)
			cycleType = payload.CycleType
		}
		decision, err := world.SimulateWorldCycle(c.Request.Context(), worldID, "admin", cycleType)
		if err == nil && decision != nil {
			s.gameAudit(c, "simulate", "world", strconv.FormatUint(uint64(decision.WorldID), 10), nil, decision)
		}
		gameJSON(c, decision, err)
	}
}

func (s *Server) gameWorldRoutineAPI(world *service.WorldGameService, generate bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var worldID uint
		if c.Param("id") != "" {
			id, err := gameParam(c, "id")
			if err != nil {
				gameJSON(c, nil, err)
				return
			}
			worldID = id
		}
		if generate {
			routine, decision, err := world.GenerateWorldFourPageRoutine(c.Request.Context(), worldID, "admin")
			if err == nil && routine != nil {
				s.gameAudit(c, "generate_routine_4_pages", "world", strconv.FormatUint(uint64(routine.WorldID), 10), nil, gin.H{"routine": routine, "decision": decision})
			}
			gameJSON(c, gin.H{"routine": routine, "decision": decision}, err)
			return
		}
		routine, err := world.LatestWorldFourPageRoutine(c.Request.Context(), worldID)
		gameJSON(c, routine, err)
	}
}

func (s *Server) gameRecalculatePlayerMetricsAPI(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		playerID, err := gameParam(c, "id")
		if err != nil {
			gameJSON(c, nil, err)
			return
		}
		var save models.PlayerSave
		err = s.db.WithContext(c.Request.Context()).Where("player_id = ?", playerID).First(&save).Error
		if err != nil {
			gameJSON(c, nil, err)
			return
		}
		err = world.CalculateAndStorePlayerWorldMetric(c.Request.Context(), save)
		if err != nil {
			gameJSON(c, nil, err)
			return
		}
		var metric models.PlayerWorldMetric
		err = s.db.WithContext(c.Request.Context()).Where("player_id = ? AND world_id = ?", save.PlayerID, save.WorldID).First(&metric).Error
		if err == nil {
			s.gameAudit(c, "recalculate_metrics", "player", strconv.FormatUint(uint64(save.PlayerID), 10), nil, metric)
		}
		gameJSON(c, metric, err)
	}
}

func (s *Server) gamePlayersAPI(c *gin.Context) {
	var items []models.PlayerSave
	err := s.db.WithContext(c.Request.Context()).Preload("World").Preload("Continent").Order("updated_at DESC").Limit(gameLimit(c)).Find(&items).Error
	gameJSON(c, gin.H{"items": items}, err)
}

func (s *Server) gameCreateEventAPI(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var event models.GameEvent
		if err := c.ShouldBindJSON(&event); err != nil {
			gameJSON(c, nil, err)
			return
		}
		err := world.CreateGameEvent(c.Request.Context(), &event)
		if err == nil {
			s.gameAudit(c, "create", "event", strconv.FormatUint(uint64(event.Id), 10), nil, event)
		}
		gameJSON(c, event, err)
	}
}

func (s *Server) gameCreateConflictAPI(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input service.ConflictInput
		if err := c.ShouldBindJSON(&input); err != nil {
			gameJSON(c, nil, err)
			return
		}
		conflict, err := world.CreateConflict(c.Request.Context(), input)
		if err == nil {
			s.gameAudit(c, "create", "conflict", strconv.FormatUint(uint64(conflict.Id), 10), nil, conflict)
		}
		gameJSON(c, conflict, err)
	}
}

func (s *Server) gameResolveConflictAPI(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := gameParam(c, "id")
		if err != nil {
			gameJSON(c, nil, err)
			return
		}
		err = world.ResolveConflict(c.Request.Context(), id, adminUsername())
		if err == nil {
			s.gameAudit(c, "resolve", "conflict", strconv.FormatUint(uint64(id), 10), nil, gin.H{"status": service.ConflictStatusResolved})
		}
		gameJSON(c, gin.H{"status": service.ConflictStatusResolved}, err)
	}
}

func (s *Server) gamePatchEventAPI(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := gameParam(c, "id")
		if err != nil {
			gameJSON(c, nil, err)
			return
		}
		var fields map[string]any
		if err := c.ShouldBindJSON(&fields); err != nil {
			gameJSON(c, nil, err)
			return
		}
		delete(fields, "id")
		delete(fields, "Id")
		err = world.UpdateGameEvent(c.Request.Context(), id, fields)
		if err == nil {
			s.gameAudit(c, "patch", "event", strconv.FormatUint(uint64(id), 10), nil, fields)
		}
		gameJSON(c, gin.H{"updated": true}, err)
	}
}

func (s *Server) gamePlayerSaveAPI(c *gin.Context) {
	id, err := gameParam(c, "id")
	if err != nil {
		gameJSON(c, nil, err)
		return
	}
	var item models.PlayerSave
	err = s.db.WithContext(c.Request.Context()).Where("player_id = ?", id).First(&item).Error
	gameJSON(c, item, err)
}

func (s *Server) gamePatchPlayerSaveAPI(c *gin.Context) {
	id, err := gameParam(c, "id")
	if err != nil {
		gameJSON(c, nil, err)
		return
	}
	var fields map[string]any
	if err := c.ShouldBindJSON(&fields); err != nil {
		gameJSON(c, nil, err)
		return
	}
	delete(fields, "worldId")
	delete(fields, "continentId")
	err = s.db.WithContext(c.Request.Context()).Model(&models.PlayerSave{}).Where("player_id = ?", id).Updates(fields).Error
	if err == nil {
		s.gameAudit(c, "patch_save", "player", strconv.FormatUint(uint64(id), 10), nil, fields)
	}
	gameJSON(c, gin.H{"updated": true}, err)
}

func (s *Server) gameResyncPlayerAPI(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := gameParam(c, "id")
		if err != nil {
			gameJSON(c, nil, err)
			return
		}
		save, err := world.EnsurePlayerSave(c.Request.Context(), id)
		gameJSON(c, save, err)
	}
}

func (s *Server) gameGetGuildAPI(c *gin.Context) {
	id, err := gameParam(c, "id")
	if err != nil {
		gameJSON(c, nil, err)
		return
	}
	var item models.Guild
	err = s.db.WithContext(c.Request.Context()).Preload("Members").First(&item, id).Error
	gameJSON(c, item, err)
}

func (s *Server) gameGuildMembersAPI(c *gin.Context) {
	id, err := gameParam(c, "id")
	if err != nil {
		gameJSON(c, nil, err)
		return
	}
	var items []models.GuildMember
	err = s.db.WithContext(c.Request.Context()).Where("guild_id = ?", id).Order("role ASC, joined_at ASC").Find(&items).Error
	gameJSON(c, gin.H{"items": items}, err)
}

func (s *Server) gameGuildRoleAPI(c *gin.Context) {
	guildID, err1 := gameParam(c, "id")
	playerID, err2 := gameParam(c, "playerId")
	if err1 != nil || err2 != nil {
		gameJSON(c, nil, errors.New("invalid guild member id"))
		return
	}
	var input struct {
		Role string `json:"role"`
	}
	_ = c.ShouldBindJSON(&input)
	if strings.TrimSpace(input.Role) == "" {
		input.Role = service.GuildRoleMember
	}
	err := s.db.WithContext(c.Request.Context()).Model(&models.GuildMember{}).Where("guild_id = ? AND player_id = ?", guildID, playerID).Update("role", input.Role).Error
	gameJSON(c, gin.H{"role": input.Role}, err)
}

func (s *Server) gameGuildKickAPI(c *gin.Context) {
	guildID, err1 := gameParam(c, "id")
	playerID, err2 := gameParam(c, "playerId")
	if err1 != nil || err2 != nil {
		gameJSON(c, nil, errors.New("invalid guild member id"))
		return
	}
	err := s.db.WithContext(c.Request.Context()).Where("guild_id = ? AND player_id = ?", guildID, playerID).Delete(&models.GuildMember{}).Error
	gameJSON(c, gin.H{"deleted": true}, err)
}

func (s *Server) gameModerateChatAPI(hide bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := gameParam(c, "id")
		if err != nil {
			gameJSON(c, nil, err)
			return
		}
		var value any
		if hide {
			now := time.Now()
			value = &now
		}
		err = s.db.WithContext(c.Request.Context()).Model(&models.ChatMessage{}).Where("id = ?", id).Update("moderated_at", value).Error
		if err == nil {
			s.gameAudit(c, "moderate_chat", "chat_message", strconv.FormatUint(uint64(id), 10), nil, gin.H{"hidden": hide})
		}
		gameJSON(c, gin.H{"hidden": hide}, err)
	}
}

func (s *Server) gameGetBuildingAPI(c *gin.Context) {
	id, err := gameParam(c, "id")
	if err != nil {
		gameJSON(c, nil, err)
		return
	}
	var item models.BuildingDefinition
	err = s.db.WithContext(c.Request.Context()).Preload("Assets").First(&item, id).Error
	gameJSON(c, item, err)
}

func (s *Server) gameBuildingAssetsAPI(c *gin.Context) {
	id, err := gameParam(c, "id")
	if err != nil {
		gameJSON(c, nil, err)
		return
	}
	var items []models.BuildingAsset
	err = s.db.WithContext(c.Request.Context()).Where("building_definition_id = ?", id).Order("level ASC, variant ASC").Find(&items).Error
	gameJSON(c, gin.H{"items": items}, err)
}

func (s *Server) gameCreateBuildingAssetAPI(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		buildingID, err := gameParam(c, "id")
		if err != nil {
			gameJSON(c, nil, err)
			return
		}
		if strings.Contains(c.GetHeader("Content-Type"), "multipart/form-data") {
			file, header, err := c.Request.FormFile("image")
			if err != nil {
				gameJSON(c, nil, err)
				return
			}
			defer file.Close()
			level, _ := strconv.Atoi(c.DefaultPostForm("level", "1"))
			version, _ := strconv.Atoi(c.DefaultPostForm("version", "0"))
			asset, err := world.SaveBuildingAssetUpload(c.Request.Context(), buildingID, level, c.DefaultPostForm("variant", "default"), version, header.Filename, file)
			if err == nil {
				s.gameAudit(c, "upload_asset", "building_asset", strconv.FormatUint(uint64(asset.Id), 10), nil, asset)
			}
			gameJSON(c, asset, err)
			return
		}
		var item models.BuildingAsset
		if err := c.ShouldBindJSON(&item); err != nil {
			gameJSON(c, nil, err)
			return
		}
		item.BuildingDefinitionID = buildingID
		if item.Version <= 0 {
			item.Version = 1
		}
		if item.ImageHash == "" {
			item.ImageHash = world.CreateBuildingAssetHash(item.ImageURL, item.Level, item.Version)
		}
		err = s.db.WithContext(c.Request.Context()).Create(&item).Error
		gameJSON(c, item, err)
	}
}

func (s *Server) gameManifestAPI(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		manifest, err := world.BuildingManifest(c.Request.Context(), 0)
		gameJSON(c, manifest, err)
	}
}

func (s *Server) gamePublishManifestAPI(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		version, err := world.PublishCatalog(c.Request.Context(), gin.H{"source": "admin"})
		if err == nil {
			s.gameAudit(c, "publish_manifest", "building_catalog", strconv.Itoa(version.Version), nil, version)
		}
		gameJSON(c, version, err)
	}
}

func (s *Server) gameEndWeatherAPI(c *gin.Context) {
	id, err := gameParam(c, "id")
	if err != nil {
		gameJSON(c, nil, err)
		return
	}
	err = s.db.WithContext(c.Request.Context()).Model(&models.WeatherEvent{}).Where("id = ?", id).Update("ends_at", time.Now()).Error
	gameJSON(c, gin.H{"ended": true}, err)
}

func (s *Server) gameDecisionDryRunAPI(c *gin.Context) {
	id, err := gameParam(c, "id")
	if err != nil {
		gameJSON(c, nil, err)
		return
	}
	var item models.AIWorldDecision
	err = s.db.WithContext(c.Request.Context()).First(&item, id).Error
	if err != nil {
		gameJSON(c, nil, err)
		return
	}
	replay, err := service.NewWorldGameService(s.db).DryRunWorldSimulation(c.Request.Context(), item.WorldID, item.Type)
	gameJSON(c, gin.H{"dryRun": true, "source": item, "decision": replay}, err)
}

func (s *Server) gameDecisionActivationAPI(active bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := gameParam(c, "id")
		if err != nil {
			gameJSON(c, nil, err)
			return
		}
		var before models.AIWorldDecision
		if err := s.db.WithContext(c.Request.Context()).First(&before, id).Error; err != nil {
			gameJSON(c, nil, err)
			return
		}
		err = s.db.WithContext(c.Request.Context()).Model(&models.AIWorldDecision{}).Where("id = ?", id).Update("is_active", active).Error
		if err != nil {
			gameJSON(c, nil, err)
			return
		}
		var after models.AIWorldDecision
		_ = s.db.WithContext(c.Request.Context()).First(&after, id).Error
		action := "enable"
		if !active {
			action = "disable"
		}
		s.gameAudit(c, action, "ai_decision", strconv.FormatUint(uint64(id), 10), before, after)
		gameJSON(c, gin.H{"id": id, "isActive": active, "decision": after}, nil)
	}
}

func (s *Server) gameWorldCronAPI(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"cron": scheduler.WorldSimulationCronSnapshot()})
}

func (s *Server) gameWorldCronToggleAPI(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		snapshot := scheduler.SetWorldSimulationCronEnabled(enabled, "admin")
		action := "enable_world_ai_cron"
		if !enabled {
			action = "disable_world_ai_cron"
		}
		s.gameAudit(c, action, "world_ai_cron", "runtime", nil, snapshot)
		c.JSON(http.StatusOK, gin.H{"cron": snapshot})
	}
}

func (s *Server) gameListModelAPI(dest any, order string) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := s.db.WithContext(c.Request.Context()).Order(order).Limit(gameLimit(c))
		query = gameApplyFilters(c, query)
		err := query.Find(dest).Error
		gameJSON(c, gin.H{"items": dest}, err)
	}
}

func (s *Server) gameGetModelAPI(dest any) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := gameParam(c, "id")
		if err != nil {
			gameJSON(c, nil, err)
			return
		}
		err = s.db.WithContext(c.Request.Context()).First(dest, id).Error
		gameJSON(c, dest, err)
	}
}

func (s *Server) gameGetResearchTreeAPI(c *gin.Context) {
	id, err := gameParam(c, "id")
	if err != nil {
		gameJSON(c, nil, err)
		return
	}
	var tree models.ResearchTreeDefinition
	err = s.db.WithContext(c.Request.Context()).Preload("Nodes", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC, id ASC")
	}).First(&tree, id).Error
	gameJSON(c, tree, err)
}

func (s *Server) gameCreateModelAPI(dest any, target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.ShouldBindJSON(dest); err != nil {
			gameJSON(c, nil, err)
			return
		}
		err := s.db.WithContext(c.Request.Context()).Create(dest).Error
		if err == nil {
			s.gameAudit(c, "create", target, "", nil, dest)
		}
		gameJSON(c, dest, err)
	}
}

func (s *Server) gamePatchModelAPI(model any, target string) gin.HandlerFunc {
	return s.gamePatchModelParamAPI(model, "id", target)
}

func (s *Server) gamePatchModelParamAPI(model any, param string, target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := gameParam(c, param)
		if err != nil {
			gameJSON(c, nil, err)
			return
		}
		var fields map[string]any
		if err := c.ShouldBindJSON(&fields); err != nil {
			gameJSON(c, nil, err)
			return
		}
		delete(fields, "id")
		delete(fields, "Id")
		err = s.db.WithContext(c.Request.Context()).Model(model).Where("id = ?", id).Updates(fields).Error
		if err == nil {
			s.gameAudit(c, "patch", target, strconv.FormatUint(uint64(id), 10), nil, fields)
		}
		gameJSON(c, gin.H{"updated": true}, err)
	}
}

func (s *Server) gameDeleteModelAPI(model any, target string) gin.HandlerFunc {
	return s.gameDeleteModelParamAPI(model, "id", target)
}

func (s *Server) gameDeleteModelParamAPI(model any, param string, target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := gameParam(c, param)
		if err != nil {
			gameJSON(c, nil, err)
			return
		}
		err = s.db.WithContext(c.Request.Context()).Delete(model, id).Error
		if err == nil {
			s.gameAudit(c, "delete", target, strconv.FormatUint(uint64(id), 10), nil, nil)
		}
		gameJSON(c, gin.H{"deleted": true}, err)
	}
}

func (s *Server) gamePatchStatusAPI(model any, status string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := gameParam(c, "id")
		if err != nil {
			gameJSON(c, nil, err)
			return
		}
		err = s.db.WithContext(c.Request.Context()).Model(model).Where("id = ?", id).Update("status", status).Error
		gameJSON(c, gin.H{"status": status}, err)
	}
}

func gameApplyFilters(c *gin.Context, query *gorm.DB) *gorm.DB {
	fields := map[string]string{
		"worldId":                  "world_id",
		"continentId":              "continent_id",
		"guildId":                  "guild_id",
		"playerId":                 "player_id",
		"status":                   "status",
		"isActive":                 "is_active",
		"type":                     "type",
		"domain":                   "domain",
		"buildingKey":              "building_key",
		"researchTreeDefinitionId": "research_tree_definition_id",
		"channelType":              "channel_type",
	}
	for queryKey, column := range fields {
		if value := c.Query(queryKey); value != "" {
			query = query.Where(column+" = ?", value)
		}
	}
	return query
}

func gameParam(c *gin.Context, name string) (uint, error) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || value == 0 {
		return 0, errors.New("invalid " + name)
	}
	return uint(value), nil
}

func gameLimit(c *gin.Context) int {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil || limit <= 0 {
		return 100
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func gameJSON(c *gin.Context, payload any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, payload)
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func (s *Server) gameAudit(c *gin.Context, action string, targetType string, targetID string, before any, after any) {
	beforeJSON := gameMustJSON(before)
	afterJSON := gameMustJSON(after)
	_ = s.db.WithContext(c.Request.Context()).Create(&models.AdminAuditLog{
		AdminID:    adminUsername(),
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		BeforeJSON: beforeJSON,
		AfterJSON:  afterJSON,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
	}).Error
}

func gameMustJSON(value any) datatypes.JSON {
	if value == nil {
		return datatypes.JSON([]byte(`{}`))
	}
	data, err := json.Marshal(value)
	if err != nil || len(data) == 0 {
		return datatypes.JSON([]byte(`{}`))
	}
	return datatypes.JSON(data)
}
