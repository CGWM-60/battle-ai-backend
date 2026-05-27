package router

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type playerActionRequest struct {
	ActionType string         `json:"actionType"`
	Payload    datatypes.JSON `json:"payload"`
}

func registerWorldGameRoutes(private *gin.RouterGroup, database *gorm.DB) {
	world := service.NewWorldGameService(database)

	private.GET("/player/state", func(c *gin.Context) {
		state, err := world.PlayerState(c.Request.Context(), currentUserID(c))
		writeWorldResponse(c, state, err)
	})
	private.GET("/player/save", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		writeWorldResponse(c, save, err)
	})
	private.POST("/player/save/sync", func(c *gin.Context) {
		var input service.PlayerSaveSyncInput
		if err := bindPayload(c, &input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid save sync payload"})
			return
		}
		save, err := world.SyncPlayerSave(c.Request.Context(), currentUserID(c), input)
		writeWorldResponse(c, save, err)
	})
	private.POST("/player/action", func(c *gin.Context) {
		var input playerActionRequest
		if err := bindPayload(c, &input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player action payload"})
			return
		}
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		payload := input.Payload
		if len(payload) == 0 {
			payload = datatypes.JSON([]byte(`{}`))
		}
		c.JSON(http.StatusAccepted, gin.H{"accepted": true, "worldId": save.WorldID, "continentId": save.ContinentID, "actionType": input.ActionType, "payload": payload})
	})

	private.GET("/world/current", func(c *gin.Context) {
		current, err := world.CurrentWorld(c.Request.Context(), currentUserID(c))
		writeWorldResponse(c, current, err)
	})
	private.GET("/world/status", func(c *gin.Context) {
		state, err := world.PlayerState(c.Request.Context(), currentUserID(c))
		writeWorldResponse(c, state, err)
	})
	private.GET("/world/events", playerScopedList(curryPlayerScope(world, func(ctxUser playerScope) (any, error) {
		return world.ListWorldEvents(ctxUser.Context, ctxUser.Save.WorldID, ctxUser.Save.ContinentID, ctxUser.Limit)
	})))
	private.GET("/world/conflicts", playerScopedList(curryPlayerScope(world, func(ctxUser playerScope) (any, error) {
		return world.ListWorldConflicts(ctxUser.Context, ctxUser.Save.WorldID, ctxUser.Save.ContinentID, ctxUser.Limit)
	})))
	private.GET("/world/weather", playerScopedList(curryPlayerScope(world, func(ctxUser playerScope) (any, error) {
		return world.ListActiveWeather(ctxUser.Context, ctxUser.Save.WorldID, ctxUser.Save.ContinentID)
	})))
	private.GET("/weather/active", playerScopedList(curryPlayerScope(world, func(ctxUser playerScope) (any, error) {
		return world.ListActiveWeather(ctxUser.Context, ctxUser.Save.WorldID, ctxUser.Save.ContinentID)
	})))
	private.GET("/world/messages", func(c *gin.Context) {
		messages, err := world.ListDailyMessages(c.Request.Context(), currentUserID(c), limitFromQuery(c))
		writeWorldResponse(c, gin.H{"messages": messages}, err)
	})
	private.POST("/world/messages/:id/read", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
			return
		}
		writeWorldResponse(c, gin.H{"updated": true}, world.MarkDailyMessageRead(c.Request.Context(), currentUserID(c), id))
	})

	private.GET("/events/:id", getOwnedWorldEntity[models.GameEvent](database, world, "game event", "world_id = ? AND (continent_id IS NULL OR continent_id = ?)"))
	private.POST("/events/:id/participate", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		writeWorldResponse(c, gin.H{"participating": true}, world.ParticipateEvent(c.Request.Context(), currentUserID(c), id))
	})
	private.POST("/events/:id/claim", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		writeWorldResponse(c, gin.H{"claimed": true}, world.ClaimEvent(c.Request.Context(), currentUserID(c), id))
	})
	private.GET("/conflicts/:id", getOwnedWorldEntity[models.Conflict](database, world, "conflict", "world_id = ? AND (continent_id IS NULL OR continent_id = ?)"))
	private.POST("/conflicts/:id/action", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conflict id"})
			return
		}
		var input service.EventActionInput
		_ = bindPayload(c, &input)
		writeWorldResponse(c, gin.H{"accepted": true}, world.ConflictAction(c.Request.Context(), currentUserID(c), id, input))
	})

	registerChatRoutes(private, world)
	registerGuildRoutes(private, database, world)
	registerBuildingRoutes(private, world)
}

func registerAdminWorldGameRoutes(group *gin.RouterGroup, database *gorm.DB) {
	world := service.NewWorldGameService(database)
	admin := group.Group("/admin")
	game := admin.Group("/game")
	game.GET("/dashboard", adminGameDashboard(database, world))
	game.GET("/stats", adminGameDashboard(database, world))
	game.POST("/world/simulate", adminSimulateWorld(world, 0))
	game.POST("/world/message", adminSendAIMessage(database))

	admin.GET("/worlds", adminList[models.World](database, "id DESC"))
	admin.POST("/worlds", func(c *gin.Context) {
		item, err := world.CreateWorld(c.Request.Context())
		writeWorldResponse(c, item, err)
	})
	admin.GET("/worlds/:id", adminGet[models.World](database, true))
	admin.PATCH("/worlds/:id", adminPatch[models.World](database, "world"))
	admin.POST("/worlds/:id/simulate", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid world id"})
			return
		}
		decision, err := world.SimulateWorld(c.Request.Context(), id, "admin-api")
		writeWorldResponse(c, decision, err)
	})
	admin.GET("/worlds/:id/continents", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid world id"})
			return
		}
		var items []models.Continent
		err = database.WithContext(c.Request.Context()).Where("world_id = ?", id).Order("`index` ASC").Find(&items).Error
		writeWorldResponse(c, gin.H{"continents": items}, err)
	})
	admin.GET("/continents", adminList[models.Continent](database, "`world_id` ASC, `index` ASC"))
	admin.GET("/continents/:id", adminGet[models.Continent](database, false))
	admin.PATCH("/continents/:id", adminPatch[models.Continent](database, "continent"))
	admin.POST("/continents/:id/simulate", func(c *gin.Context) {
		var continent models.Continent
		id, err := parseUintParam(c, "id")
		if err == nil {
			err = database.WithContext(c.Request.Context()).First(&continent, id).Error
		}
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		decision, err := world.SimulateWorld(c.Request.Context(), continent.WorldID, "admin-continent")
		writeWorldResponse(c, decision, err)
	})

	admin.GET("/players", adminPlayers(database))
	admin.GET("/players/:id", adminPlayer(database))
	admin.GET("/players/:id/save", adminGetPlayerSave(database))
	admin.PATCH("/players/:id/save", adminPatchPlayerSave(database))
	admin.POST("/players/:id/resync", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player id"})
			return
		}
		save, err := world.EnsurePlayerSave(c.Request.Context(), id)
		writeWorldResponse(c, save, err)
	})

	admin.GET("/events", adminList[models.GameEvent](database, "starts_at DESC"))
	admin.POST("/events", adminCreate[models.GameEvent](database, "event"))
	admin.GET("/events/:id", adminGet[models.GameEvent](database, false))
	admin.PATCH("/events/:id", adminPatch[models.GameEvent](database, "event"))
	admin.DELETE("/events/:id", adminSoftDelete[models.GameEvent](database, "event"))
	admin.POST("/events/:id/force-start", adminPatchStatus[models.GameEvent](database, "active"))
	admin.POST("/events/:id/force-end", adminPatchStatus[models.GameEvent](database, "finished"))
	admin.POST("/events/generate-ai", adminSimulateWorld(world, 0))

	admin.GET("/conflicts", adminList[models.Conflict](database, "starts_at DESC"))
	admin.POST("/conflicts", adminCreate[models.Conflict](database, "conflict"))
	admin.GET("/conflicts/:id", adminGet[models.Conflict](database, false))
	admin.PATCH("/conflicts/:id", adminPatch[models.Conflict](database, "conflict"))
	admin.POST("/conflicts/:id/resolve", adminPatchStatus[models.Conflict](database, "resolved"))
	admin.POST("/conflicts/generate-ai", adminSimulateWorld(world, 0))

	admin.GET("/weather", adminList[models.WeatherEvent](database, "starts_at DESC"))
	admin.POST("/weather", adminCreate[models.WeatherEvent](database, "weather"))
	admin.GET("/weather/:id", adminGet[models.WeatherEvent](database, false))
	admin.PATCH("/weather/:id", adminPatch[models.WeatherEvent](database, "weather"))
	admin.POST("/weather/:id/end", func(c *gin.Context) {
		id, _ := parseUintParam(c, "id")
		err := database.WithContext(c.Request.Context()).Model(&models.WeatherEvent{}).Where("id = ?", id).Update("ends_at", time.Now()).Error
		writeWorldResponse(c, gin.H{"updated": true}, err)
	})
	admin.POST("/weather/generate-ai", adminSimulateWorld(world, 0))

	admin.GET("/ai/messages", adminList[models.DailyAIMessage](database, "created_at DESC"))
	admin.POST("/ai/messages", adminCreate[models.DailyAIMessage](database, "ai_message"))
	admin.POST("/ai/messages/generate", adminSimulateWorld(world, 0))
	admin.POST("/ai/messages/send", adminSendAIMessage(database))
	admin.GET("/ai/decisions", adminList[models.AIWorldDecision](database, "created_at DESC"))
	admin.GET("/ai/decisions/:id", adminGet[models.AIWorldDecision](database, false))
	admin.POST("/ai/decisions/:id/replay-dry-run", func(c *gin.Context) {
		item, err := adminFindByID[models.AIWorldDecision](c, database)
		writeWorldResponse(c, gin.H{"dryRun": true, "decision": item}, err)
	})
	admin.GET("/ai/providers", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"providers": world.AIProviderStatuses()})
	})
	admin.POST("/ai/providers/test", testAIProvider())
	admin.GET("/ai/factions", adminList[models.AIWorldFaction](database, "id DESC"))
	admin.POST("/ai/factions", adminCreate[models.AIWorldFaction](database, "ai_faction"))
	admin.GET("/ai/factions/:id", adminGet[models.AIWorldFaction](database, false))
	admin.PATCH("/ai/factions/:id", adminPatch[models.AIWorldFaction](database, "ai_faction"))
	admin.DELETE("/ai/factions/:id", adminSoftDelete[models.AIWorldFaction](database, "ai_faction"))

	admin.GET("/guilds", adminList[models.Guild](database, "id DESC"))
	admin.GET("/guilds/:id", adminGet[models.Guild](database, true))
	admin.PATCH("/guilds/:id", adminPatch[models.Guild](database, "guild"))
	admin.DELETE("/guilds/:id", adminSoftDelete[models.Guild](database, "guild"))
	admin.GET("/guilds/:id/members", adminGuildMembers(database))
	admin.PATCH("/guilds/:id/members/:playerId", adminPatchGuildMember(database))
	admin.DELETE("/guilds/:id/members/:playerId", adminDeleteGuildMember(database))

	admin.GET("/chat/messages", adminList[models.ChatMessage](database, "created_at DESC"))
	admin.DELETE("/chat/messages/:id", adminModerateChat(database, true))
	admin.POST("/chat/messages/:id/restore", adminModerateChat(database, false))

	admin.GET("/buildings", adminList[models.BuildingDefinition](database, "sort_order ASC, id ASC"))
	admin.POST("/buildings", adminCreate[models.BuildingDefinition](database, "building"))
	admin.GET("/buildings/:id", adminGet[models.BuildingDefinition](database, true))
	admin.PATCH("/buildings/:id", adminPatch[models.BuildingDefinition](database, "building"))
	admin.DELETE("/buildings/:id", adminSoftDelete[models.BuildingDefinition](database, "building"))
	admin.GET("/buildings/:id/assets", adminBuildingAssets(database))
	admin.POST("/buildings/:id/assets", adminCreateBuildingAsset(database, world))
	admin.GET("/building-assets", adminList[models.BuildingAsset](database, "building_definition_id ASC, level ASC"))
	admin.PATCH("/building-assets/:assetId", adminPatchParam[models.BuildingAsset](database, "assetId", "building_asset"))
	admin.DELETE("/building-assets/:assetId", adminSoftDeleteParam[models.BuildingAsset](database, "assetId", "building_asset"))
	admin.GET("/assets/manifest", func(c *gin.Context) {
		manifest, err := world.BuildingManifest(c.Request.Context(), 0)
		writeWorldResponse(c, manifest, err)
	})
	admin.POST("/assets/manifest/publish", func(c *gin.Context) {
		version, err := world.PublishCatalog(c.Request.Context(), gin.H{"source": "admin-api"})
		writeWorldResponse(c, version, err)
	})
	admin.GET("/audit", adminList[models.AdminAuditLog](database, "created_at DESC"))
}

func registerChatRoutes(private *gin.RouterGroup, world *service.WorldGameService) {
	channels := []string{service.ChatWorldChannel, service.ChatContinentChannel, service.ChatGuildChannel}
	for _, channel := range channels {
		ch := channel
		private.GET("/chat/"+ch, func(c *gin.Context) {
			messages, err := world.ListChat(c.Request.Context(), currentUserID(c), ch, limitFromQuery(c))
			writeWorldResponse(c, gin.H{"messages": messages}, err)
		})
		private.POST("/chat/"+ch, func(c *gin.Context) {
			var input service.ChatInput
			if err := bindPayload(c, &input); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chat payload"})
				return
			}
			message, err := world.SendChat(c.Request.Context(), currentUserID(c), ch, input)
			writeWorldResponse(c, message, err)
		})
	}
}

func registerGuildRoutes(private *gin.RouterGroup, database *gorm.DB, world *service.WorldGameService) {
	private.GET("/guilds", func(c *gin.Context) {
		guilds, err := world.ListGuilds(c.Request.Context(), currentUserID(c), limitFromQuery(c))
		writeWorldResponse(c, gin.H{"guilds": guilds}, err)
	})
	private.POST("/guilds", func(c *gin.Context) {
		var input service.GuildInput
		if err := bindPayload(c, &input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild payload"})
			return
		}
		guild, err := world.CreateGuild(c.Request.Context(), currentUserID(c), input)
		writeWorldResponse(c, guild, err)
	})
	private.GET("/guilds/:id", getOwnedGuild(database, world))
	private.POST("/guilds/:id/join", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild id"})
			return
		}
		writeWorldResponse(c, gin.H{"joined": true}, world.JoinGuild(c.Request.Context(), currentUserID(c), id))
	})
	private.POST("/guilds/:id/leave", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild id"})
			return
		}
		writeWorldResponse(c, gin.H{"left": true}, world.LeaveGuild(c.Request.Context(), currentUserID(c), id))
	})
	private.GET("/guilds/:id/members", adminGuildMembers(database))
	private.GET("/guilds/:id/chat", func(c *gin.Context) {
		messages, err := world.ListChat(c.Request.Context(), currentUserID(c), service.ChatGuildChannel, limitFromQuery(c))
		writeWorldResponse(c, gin.H{"messages": messages}, err)
	})
	private.POST("/guilds/:id/invite", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild id"})
			return
		}
		var input struct {
			PlayerID uint `json:"playerId"`
		}
		_ = bindPayload(c, &input)
		invite := models.GuildInvite{GuildID: id, InviterPlayerID: currentUserID(c), InvitedPlayerID: input.PlayerID, Status: "pending", ExpiresAt: time.Now().Add(7 * 24 * time.Hour)}
		writeWorldResponse(c, invite, database.WithContext(c.Request.Context()).Create(&invite).Error)
	})
	private.POST("/guilds/invites/:id/accept", guildInviteStatus(database, "accepted", true))
	private.POST("/guilds/invites/:id/decline", guildInviteStatus(database, "declined", false))
	private.POST("/guilds/:id/members/:playerId/role", func(c *gin.Context) {
		adminPatchGuildMember(database)(c)
	})
}

func registerBuildingRoutes(private *gin.RouterGroup, world *service.WorldGameService) {
	private.GET("/buildings/catalog", func(c *gin.Context) {
		catalog, err := world.BuildingCatalog(c.Request.Context(), true)
		writeWorldResponse(c, gin.H{"buildings": catalog}, err)
	})
	private.GET("/buildings/catalog/version", func(c *gin.Context) {
		version, err := world.CurrentCatalogVersion(c.Request.Context())
		writeWorldResponse(c, gin.H{"version": version}, err)
	})
	private.GET("/buildings/:key", func(c *gin.Context) {
		catalog, err := world.BuildingCatalog(c.Request.Context(), true)
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		for _, building := range catalog {
			if building.Key == c.Param("key") {
				c.JSON(http.StatusOK, building)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "building not found"})
	})
	private.GET("/assets/buildings/manifest", func(c *gin.Context) {
		since, _ := strconv.Atoi(c.DefaultQuery("sinceVersion", "0"))
		manifest, err := world.BuildingManifest(c.Request.Context(), since)
		writeWorldResponse(c, manifest, err)
	})
	private.GET("/assets/buildings/updates", func(c *gin.Context) {
		since, _ := strconv.Atoi(c.DefaultQuery("sinceVersion", "0"))
		manifest, err := world.BuildingManifest(c.Request.Context(), since)
		writeWorldResponse(c, manifest, err)
	})
}

type playerScope struct {
	Context contextLike
	Save    *models.PlayerSave
	Limit   int
}

type contextLike interface {
	Done() <-chan struct{}
	Err() error
	Value(key any) any
	Deadline() (deadline time.Time, ok bool)
}

func curryPlayerScope(world *service.WorldGameService, fn func(playerScope) (any, error)) func(*gin.Context) (any, error) {
	return func(c *gin.Context) (any, error) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			return nil, err
		}
		return fn(playerScope{Context: c.Request.Context(), Save: save, Limit: limitFromQuery(c)})
	}
}

func playerScopedList(fn func(*gin.Context) (any, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := fn(c)
		writeWorldResponse(c, data, err)
	}
}

func getOwnedWorldEntity[T any](database *gorm.DB, world *service.WorldGameService, label string, scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + label + " id"})
			return
		}
		var item T
		err = database.WithContext(c.Request.Context()).
			Where("id = ?", id).
			Where(scope, save.WorldID, save.ContinentID).
			First(&item).Error
		writeWorldResponse(c, item, err)
	}
}

func getOwnedGuild(database *gorm.DB, world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild id"})
			return
		}
		var guild models.Guild
		err = database.WithContext(c.Request.Context()).Preload("Members").Where("id = ? AND world_id = ?", id, save.WorldID).First(&guild).Error
		writeWorldResponse(c, guild, err)
	}
}

func guildInviteStatus(database *gorm.DB, status string, join bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invite id"})
			return
		}
		var invite models.GuildInvite
		err = database.WithContext(c.Request.Context()).Where("id = ? AND invited_player_id = ?", id, currentUserID(c)).First(&invite).Error
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		err = database.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&invite).Update("status", status).Error; err != nil {
				return err
			}
			if join {
				return service.NewWorldGameService(tx).JoinGuild(c.Request.Context(), currentUserID(c), invite.GuildID)
			}
			return nil
		})
		writeWorldResponse(c, gin.H{"status": status}, err)
	}
}

func adminGameDashboard(database *gorm.DB, world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		counts := map[string]int64{}
		countModel := func(key string, model any, query ...string) {
			var total int64
			q := database.WithContext(c.Request.Context()).Model(model)
			if len(query) > 0 && query[0] != "" {
				q = q.Where(query[0], queryArgs(query[1:])...)
			}
			_ = q.Count(&total).Error
			counts[key] = total
		}
		countModel("worlds", &models.World{})
		countModel("continents", &models.Continent{})
		countModel("players", &models.PlayerSave{})
		countModel("guilds", &models.Guild{})
		countModel("activeEvents", &models.GameEvent{}, "status = ?", "active")
		countModel("activeConflicts", &models.Conflict{}, "status = ?", "active")
		countModel("activeWeather", &models.WeatherEvent{}, "ends_at >= ?", time.Now().Format(time.RFC3339))
		countModel("aiMessagesToday", &models.DailyAIMessage{}, "created_at >= ?", time.Now().Truncate(24*time.Hour).Format(time.RFC3339))
		countModel("activeBuildingAssets", &models.BuildingAsset{}, "is_active = ?", "true")
		var last models.World
		_ = database.WithContext(c.Request.Context()).Where("last_simulation_at IS NOT NULL").Order("last_simulation_at DESC").First(&last).Error
		c.JSON(http.StatusOK, gin.H{
			"stats":          counts,
			"lastSimulation": last.LastSimulationAt,
			"providers":      world.AIProviderStatuses(),
			"generatedAt":    time.Now(),
		})
	}
}

func queryArgs(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		if value == "true" {
			out = append(out, true)
			continue
		}
		out = append(out, value)
	}
	return out
}

func adminList[T any](database *gorm.DB, order string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var items []T
		query := database.WithContext(c.Request.Context()).Order(order).Limit(limitFromQuery(c))
		applyAdminFilters(c, query)
		err := query.Find(&items).Error
		writeWorldResponse(c, gin.H{"items": items}, err)
	}
}

func adminGet[T any](database *gorm.DB, preload bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := adminFindByID[T](c, database)
		if err == nil && preload {
			// Keep generic handler lightweight; current GORM object is already loaded.
		}
		writeWorldResponse(c, item, err)
	}
}

func adminFindByID[T any](c *gin.Context, database *gorm.DB) (T, error) {
	var item T
	id, err := parseUintParam(c, "id")
	if err != nil {
		return item, err
	}
	err = database.WithContext(c.Request.Context()).First(&item, id).Error
	return item, err
}

func adminCreate[T any](database *gorm.DB, target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var item T
		if err := c.ShouldBindJSON(&item); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		err := database.WithContext(c.Request.Context()).Create(&item).Error
		if err == nil {
			writeAudit(c, database, "create", target, "", nil, item)
		}
		writeWorldResponse(c, item, err)
	}
}

func adminPatch[T any](database *gorm.DB, target string) gin.HandlerFunc {
	return adminPatchParam[T](database, "id", target)
}

func adminPatchParam[T any](database *gorm.DB, param string, target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, param)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var fields map[string]any
		if err := c.ShouldBindJSON(&fields); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		delete(fields, "id")
		delete(fields, "Id")
		err = database.WithContext(c.Request.Context()).Model(new(T)).Where("id = ?", id).Updates(fields).Error
		if err == nil {
			writeAudit(c, database, "patch", target, strconv.FormatUint(uint64(id), 10), nil, fields)
		}
		writeWorldResponse(c, gin.H{"updated": true}, err)
	}
}

func adminSoftDelete[T any](database *gorm.DB, target string) gin.HandlerFunc {
	return adminSoftDeleteParam[T](database, "id", target)
}

func adminSoftDeleteParam[T any](database *gorm.DB, param string, target string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, param)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		err = database.WithContext(c.Request.Context()).Delete(new(T), id).Error
		if err == nil {
			writeAudit(c, database, "delete", target, strconv.FormatUint(uint64(id), 10), nil, nil)
		}
		writeWorldResponse(c, gin.H{"deleted": true}, err)
	}
}

func adminPatchStatus[T any](database *gorm.DB, status string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		err = database.WithContext(c.Request.Context()).Model(new(T)).Where("id = ?", id).Update("status", status).Error
		writeWorldResponse(c, gin.H{"status": status}, err)
	}
}

func adminSimulateWorld(world *service.WorldGameService, worldID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		decision, err := world.SimulateWorld(c.Request.Context(), worldID, "admin-api")
		writeWorldResponse(c, decision, err)
	}
}

func adminPlayers(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var saves []models.PlayerSave
		err := database.WithContext(c.Request.Context()).Preload("World").Preload("Continent").Order("updated_at DESC").Limit(limitFromQuery(c)).Find(&saves).Error
		writeWorldResponse(c, gin.H{"items": saves}, err)
	}
}

func adminPlayer(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player id"})
			return
		}
		var user models.Users
		err = database.WithContext(c.Request.Context()).First(&user, id).Error
		writeWorldResponse(c, user, err)
	}
}

func adminGetPlayerSave(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player id"})
			return
		}
		var save models.PlayerSave
		err = database.WithContext(c.Request.Context()).Where("player_id = ?", id).First(&save).Error
		writeWorldResponse(c, save, err)
	}
}

func adminPatchPlayerSave(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid player id"})
			return
		}
		var fields map[string]any
		if err := c.ShouldBindJSON(&fields); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		delete(fields, "worldId")
		delete(fields, "continentId")
		err = database.WithContext(c.Request.Context()).Model(&models.PlayerSave{}).Where("player_id = ?", id).Updates(fields).Error
		if err == nil {
			writeAudit(c, database, "patch_save", "player", strconv.FormatUint(uint64(id), 10), nil, fields)
		}
		writeWorldResponse(c, gin.H{"updated": true}, err)
	}
}

func adminGuildMembers(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild id"})
			return
		}
		var members []models.GuildMember
		err = database.WithContext(c.Request.Context()).Where("guild_id = ?", id).Order("role ASC, joined_at ASC").Find(&members).Error
		writeWorldResponse(c, gin.H{"members": members}, err)
	}
}

func adminPatchGuildMember(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err1 := parseUintParam(c, "id")
		playerID, err2 := parseUintParam(c, "playerId")
		if err1 != nil || err2 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild member id"})
			return
		}
		var input struct {
			Role string `json:"role"`
		}
		_ = c.ShouldBindJSON(&input)
		role := strings.TrimSpace(input.Role)
		if role == "" {
			role = service.GuildRoleMember
		}
		err := database.WithContext(c.Request.Context()).Model(&models.GuildMember{}).Where("guild_id = ? AND player_id = ?", id, playerID).Update("role", role).Error
		writeWorldResponse(c, gin.H{"role": role}, err)
	}
}

func adminDeleteGuildMember(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err1 := parseUintParam(c, "id")
		playerID, err2 := parseUintParam(c, "playerId")
		if err1 != nil || err2 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild member id"})
			return
		}
		err := database.WithContext(c.Request.Context()).Where("guild_id = ? AND player_id = ?", id, playerID).Delete(&models.GuildMember{}).Error
		writeWorldResponse(c, gin.H{"deleted": true}, err)
	}
}

func adminModerateChat(database *gorm.DB, hide bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
			return
		}
		var value any
		if hide {
			now := time.Now()
			value = &now
		}
		err = database.WithContext(c.Request.Context()).Model(&models.ChatMessage{}).Where("id = ?", id).Update("moderated_at", value).Error
		if err == nil {
			action := "restore_chat"
			if hide {
				action = "moderate_chat"
			}
			writeAudit(c, database, action, "chat_message", strconv.FormatUint(uint64(id), 10), nil, gin.H{"moderated": hide})
		}
		writeWorldResponse(c, gin.H{"moderated": hide}, err)
	}
}

func adminBuildingAssets(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building id"})
			return
		}
		var assets []models.BuildingAsset
		err = database.WithContext(c.Request.Context()).Where("building_definition_id = ?", id).Order("level ASC, variant ASC").Find(&assets).Error
		writeWorldResponse(c, gin.H{"items": assets}, err)
	}
}

func adminCreateBuildingAsset(database *gorm.DB, world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		buildingID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building id"})
			return
		}
		var asset models.BuildingAsset
		if err := c.ShouldBindJSON(&asset); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid asset payload"})
			return
		}
		asset.BuildingDefinitionID = buildingID
		if asset.Version <= 0 {
			asset.Version = 1
		}
		if strings.TrimSpace(asset.ImageHash) == "" {
			asset.ImageHash = world.CreateBuildingAssetHash(asset.ImageURL, asset.Level, asset.Version)
		}
		err = database.WithContext(c.Request.Context()).Create(&asset).Error
		writeWorldResponse(c, asset, err)
	}
}

func adminSendAIMessage(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input models.DailyAIMessage
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message payload"})
			return
		}
		if strings.TrimSpace(input.Title) == "" {
			input.Title = "Transmission NEXUS"
		}
		if strings.TrimSpace(input.Tone) == "" {
			input.Tone = "bilan froid"
		}
		err := database.WithContext(c.Request.Context()).Create(&input).Error
		writeWorldResponse(c, input, err)
	}
}

func applyAdminFilters(c *gin.Context, query *gorm.DB) {
	for _, key := range []string{"world_id", "continent_id", "guild_id", "player_id", "status", "type", "channel_type"} {
		if value := c.Query(key); value != "" {
			query.Where(key+" = ?", value)
		}
	}
}

func writeAudit(c *gin.Context, database *gorm.DB, action string, targetType string, targetID string, before any, after any) {
	beforeJSON, _ := jsonMarshalOrEmpty(before)
	afterJSON, _ := jsonMarshalOrEmpty(after)
	_ = database.WithContext(c.Request.Context()).Create(&models.AdminAuditLog{
		AdminID:    "admin",
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		BeforeJSON: beforeJSON,
		AfterJSON:  afterJSON,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
	}).Error
}

func jsonMarshalOrEmpty(value any) (datatypes.JSON, error) {
	if value == nil {
		return datatypes.JSON([]byte(`{}`)), nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return datatypes.JSON([]byte(`{}`)), err
	}
	return datatypes.JSON(data), nil
}

func writeWorldResponse(c *gin.Context, payload any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, payload)
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	message := err.Error()
	status := http.StatusBadRequest
	if strings.Contains(message, "database") || strings.Contains(message, "SQL") {
		status = http.StatusInternalServerError
	}
	c.JSON(status, gin.H{"error": message})
}
