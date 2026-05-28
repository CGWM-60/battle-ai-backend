package router

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

type playerActionRequest struct {
	ActionType string         `json:"actionType"`
	Payload    datatypes.JSON `json:"payload"`
}

type cityCreateRequest struct {
	CityName       string `json:"cityName"`
	Continent      string `json:"continent"`
	PlayerID       string `json:"playerId"`
	Specialization string `json:"specialization"`
}

type actionSyncRequest struct {
	ClientSaveVersion int                            `json:"clientSaveVersion"`
	Actions           []service.PlayerActionSyncItem `json:"actions"`
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
		if conflict, ok := service.AsSaveSyncConflict(err); ok {
			c.JSON(http.StatusConflict, gin.H{
				"error":         "syncConflict",
				"message":       conflict.Message,
				"serverVersion": conflict.ServerVersion,
				"clientVersion": conflict.ClientVersion,
				"serverSave":    conflict.ServerSave,
			})
			return
		}
		writeWorldResponse(c, save, err)
	})
	private.POST("/player/city/create", func(c *gin.Context) {
		var input cityCreateRequest
		if err := bindPayload(c, &input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid city create payload"})
			return
		}
		state, err := world.CreatePlayerCity(c.Request.Context(), currentUserID(c), service.PlayerCityCreateInput{
			CityName:       input.CityName,
			Continent:      input.Continent,
			Specialization: input.Specialization,
		})
		writeWorldResponse(c, state, err)
	})
	private.POST("/player/actions/sync", func(c *gin.Context) {
		var input actionSyncRequest
		if err := bindPayload(c, &input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid actions sync payload"})
			return
		}
		result, err := world.SyncPlayerActions(c.Request.Context(), currentUserID(c), service.PlayerActionsSyncInput{
			ClientSaveVersion: input.ClientSaveVersion,
			Actions:           input.Actions,
		})
		if conflict, ok := service.AsSaveSyncConflict(err); ok {
			c.JSON(http.StatusConflict, gin.H{
				"error":         "syncConflict",
				"message":       conflict.Message,
				"serverVersion": conflict.ServerVersion,
				"clientVersion": conflict.ClientVersion,
				"serverSave":    conflict.ServerSave,
			})
			return
		}
		writeWorldResponse(c, result, err)
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
	private.GET("/world/routine", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}

		seedSimulation := false
		if events, eErr := world.ListWorldEvents(c.Request.Context(), save.WorldID, save.ContinentID, 1); eErr == nil && len(events) == 0 {
			if conflicts, cErr := world.ListWorldConflicts(c.Request.Context(), save.WorldID, save.ContinentID, 1); cErr == nil && len(conflicts) == 0 {
				if weather, wErr := world.ListActiveWeather(c.Request.Context(), save.WorldID, save.ContinentID); wErr == nil && len(weather) == 0 {
					if messages, mErr := world.ListDailyMessages(c.Request.Context(), currentUserID(c), 1); mErr == nil && len(messages) == 0 {
						seedSimulation = true
					}
				}
			}
		}
		if seedSimulation {
			_, _ = world.SimulateWorldCycle(c.Request.Context(), save.WorldID, "player-routine-autoseed", service.SimulationCycleManual)
		}

		routine, err := world.LatestWorldFourPageRoutine(c.Request.Context(), save.WorldID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			routine, _, err = world.GenerateWorldFourPageRoutine(c.Request.Context(), save.WorldID, "player-api")
		}
		writeWorldResponse(c, routine, err)
	})
	private.GET("/player/metrics", func(c *gin.Context) {
		metrics, err := world.PlayerFourPillarMetrics(c.Request.Context(), currentUserID(c))
		writeWorldResponse(c, metrics, err)
	})
	private.GET("/world/events", playerScopedList(curryPlayerScope(world, func(ctxUser playerScope) (any, error) {
		events, err := world.ListWorldEvents(ctxUser.Context, ctxUser.Save.WorldID, ctxUser.Save.ContinentID, ctxUser.Limit)
		if err != nil {
			return nil, err
		}
		return gin.H{"events": formatEventItems(database, ctxUser.Context, ctxUser.Save.PlayerID, events)}, nil
	})))
	private.GET("/world/conflicts", playerScopedList(curryPlayerScope(world, func(ctxUser playerScope) (any, error) {
		conflicts, err := world.ListWorldConflicts(ctxUser.Context, ctxUser.Save.WorldID, ctxUser.Save.ContinentID, ctxUser.Limit)
		if err != nil {
			return nil, err
		}
		return gin.H{"conflicts": formatConflictItems(database, ctxUser.Context, conflicts)}, nil
	})))
	private.GET("/world/weather", playerScopedList(curryPlayerScope(world, func(ctxUser playerScope) (any, error) {
		weather, err := world.ListActiveWeather(ctxUser.Context, ctxUser.Save.WorldID, ctxUser.Save.ContinentID)
		if err != nil {
			return nil, err
		}
		return gin.H{"weather": formatWeatherItems(weather)}, nil
	})))
	private.GET("/weather/active", playerScopedList(curryPlayerScope(world, func(ctxUser playerScope) (any, error) {
		return world.ListActiveWeather(ctxUser.Context, ctxUser.Save.WorldID, ctxUser.Save.ContinentID)
	})))
	private.GET("/world/messages", func(c *gin.Context) {
		messages, err := world.ListDailyMessages(c.Request.Context(), currentUserID(c), limitFromQuery(c))
		writeWorldResponse(c, gin.H{"messages": messages}, err)
	})
	registerWorldModuleRoutes(private, database, world)
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
	registerResearchRoutes(private, world)
	registerConstructionContractRoutes(private, database, world)
}

func registerWorldModuleRoutes(private *gin.RouterGroup, database *gorm.DB, world *service.WorldGameService) {
	private.GET("/world/conflicts/report", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		conflicts, err := world.ListWorldConflicts(c.Request.Context(), save.WorldID, save.ContinentID, limitFromQuery(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		total := len(conflicts)
		high := 0
		avg := 0
		if total > 0 {
			sum := 0
			for _, conflict := range conflicts {
				intensity := clamp(conflict.Intensity, 0, 100)
				sum += intensity
				if intensity >= 70 {
					high++
				}
			}
			avg = sum / total
		}
		writeWorldResponse(c, gin.H{
			"report": gin.H{
				"activeConflicts":  total,
				"highIntensity":    high,
				"averageIntensity": avg,
			},
		}, nil)
	})

	private.GET("/world/conflicts/:id/detail", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conflict id"})
			return
		}
		conflicts, err := world.ListWorldConflicts(c.Request.Context(), save.WorldID, save.ContinentID, 200)
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		items := formatConflictItems(database, c.Request.Context(), conflicts)
		var conflict gin.H
		for _, item := range items {
			if itemID, _ := item["id"].(string); itemID == strconv.FormatUint(uint64(id), 10) {
				conflict = item
				break
			}
		}
		if conflict == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "conflict not found"})
			return
		}
		writeWorldResponse(c, gin.H{"conflict": conflict}, nil)
	})

	private.POST("/world/conflicts/:id/intervene", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conflict id"})
			return
		}
		input := service.EventActionInput{ActionType: "intervene"}
		err = world.ConflictAction(c.Request.Context(), currentUserID(c), id, input)
		writeWorldResponse(c, gin.H{"accepted": err == nil, "actionType": "intervene"}, err)
	})

	private.GET("/world/diplomacy/relations", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		relations := make([]gin.H, 0)
		var continents []models.Continent
		if err := database.WithContext(c.Request.Context()).Where("world_id = ?", save.WorldID).Order("`index` ASC").Find(&continents).Error; err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		for _, continent := range continents {
			if continent.Id == save.ContinentID {
				continue
			}
			score := clamp(100-continent.TensionLevel, 0, 100)
			relations = append(relations, gin.H{
				"id":             strconv.FormatUint(uint64(continent.Id), 10),
				"faction":        defaultText(continent.Name, "Non disponible"),
				"continentId":    continent.Id,
				"score":          score,
				"stance":         diplomacyStance(score),
				"politicalState": defaultText(continent.PoliticalState, "Non disponible"),
			})
		}
		writeWorldResponse(c, gin.H{"relations": relations}, nil)
	})

	private.GET("/world/diplomacy/treaties", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		events, err := world.ListWorldEvents(c.Request.Context(), save.WorldID, save.ContinentID, limitFromQuery(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		treaties := make([]gin.H, 0)
		for _, event := range events {
			text := strings.ToLower(event.Title + " " + event.Description + " " + event.Type)
			if strings.Contains(text, "trait") || strings.Contains(text, "alliance") || strings.Contains(text, "accord") || strings.Contains(text, "diplom") {
				treaties = append(treaties, gin.H{
					"id":     event.Id,
					"title":  event.Title,
					"status": event.Status,
					"endsAt": event.EndsAt,
				})
			}
		}
		writeWorldResponse(c, gin.H{"treaties": treaties}, nil)
	})

	private.GET("/world/diplomacy/reports", func(c *gin.Context) {
		messages, err := world.ListDailyMessages(c.Request.Context(), currentUserID(c), limitFromQuery(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		reports := make([]gin.H, 0, len(messages))
		for _, message := range messages {
			reports = append(reports, gin.H{
				"id":        message.Id,
				"title":     message.Title,
				"tone":      message.Tone,
				"isRead":    message.IsRead,
				"createdAt": message.CreatedAt,
			})
		}
		writeWorldResponse(c, gin.H{"reports": reports}, nil)
	})

	private.GET("/world/diplomacy/report", func(c *gin.Context) {
		messages, err := world.ListDailyMessages(c.Request.Context(), currentUserID(c), 1)
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		if len(messages) == 0 {
			writeWorldResponse(c, gin.H{"report": nil}, nil)
			return
		}
		writeWorldResponse(c, gin.H{"report": messages[0]}, nil)
	})

	private.POST("/world/diplomacy/negotiations/open", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		payload := bindOptionalMap(c)
		err = world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "diplomacy_negotiation_open", "diplomacy", stableActionID(payload), "accepted", "", payload)
		writeWorldResponse(c, gin.H{
			"opened":    true,
			"status":    "pending",
			"serverNow": time.Now().UTC(),
		}, err)
	})

	private.GET("/world/diplomacy/emissaries/status", func(c *gin.Context) {
		available, total := emissaryAvailability()
		writeWorldResponse(c, gin.H{
			"available":       available,
			"total":           total,
			"cooldownSeconds": 0,
		}, nil)
	})

	private.POST("/world/diplomacy/emissaries/send", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		available, total := emissaryAvailability()
		if available <= 0 {
			_ = world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "diplomacy_emissary_send", "emissary", "none", "rejected", "no emissary available", nil)
			c.JSON(http.StatusConflict, gin.H{
				"error":     "no emissary available",
				"available": available,
				"total":     total,
			})
			return
		}
		payload := bindOptionalMap(c)
		err = world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "diplomacy_emissary_send", "emissary", stableActionID(payload), "accepted", "", payload)
		writeWorldResponse(c, gin.H{
			"sent":      true,
			"status":    "en_route",
			"serverNow": time.Now().UTC(),
			"available": available - 1,
			"total":     total,
		}, err)
	})

	private.GET("/world/commerce/routes", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		conflicts, err := world.ListWorldConflicts(c.Request.Context(), save.WorldID, save.ContinentID, limitFromQuery(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		routes := commerceRoutesFromConflicts(database, c.Request.Context(), conflicts)
		writeWorldResponse(c, gin.H{"routes": routes}, nil)
	})

	private.GET("/world/commerce/details", func(c *gin.Context) {
		routesJSON, err := fetchCommerceRoutes(database, world, c)
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		mode := strings.ToLower(strings.TrimSpace(c.DefaultQuery("type", "all")))
		filtered := make([]gin.H, 0, len(routesJSON))
		for _, item := range routesJSON {
			status, _ := item["status"].(string)
			if mode == "export" && status == "perturbe" {
				continue
			}
			if mode == "import" && status == "active" {
				continue
			}
			filtered = append(filtered, item)
		}
		writeWorldResponse(c, gin.H{"details": filtered}, nil)
	})

	private.POST("/world/commerce/agreements", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		payload := bindOptionalMap(c)
		err = world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "commerce_agreement_create", "commerce_agreement", stableActionID(payload), "accepted", "", payload)
		writeWorldResponse(c, gin.H{
			"created":   true,
			"status":    "draft",
			"serverNow": time.Now().UTC(),
		}, err)
	})

	private.POST("/world/commerce/routes/optimize", func(c *gin.Context) {
		routesJSON, err := fetchCommerceRoutes(database, world, c)
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		err = world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "commerce_routes_optimize", "commerce_routes", "optimize", "accepted", "", gin.H{"routesCount": len(routesJSON)})
		writeWorldResponse(c, gin.H{
			"optimized":      true,
			"routesCount":    len(routesJSON),
			"serverNow":      time.Now().UTC(),
			"recommendation": commerceRecommendation(routesJSON),
		}, err)
	})

	private.GET("/world/commerce/report", func(c *gin.Context) {
		routesJSON, err := fetchCommerceRoutes(database, world, c)
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		var total int64
		active := 0
		for _, item := range routesJSON {
			if volume, ok := item["volume"].(int64); ok {
				total += volume
			}
			if status, _ := item["status"].(string); status == "active" {
				active++
			}
		}
		writeWorldResponse(c, gin.H{
			"report": gin.H{
				"totalVolume":  total,
				"activeRoutes": active,
				"routesCount":  len(routesJSON),
			},
		}, nil)
	})

	private.GET("/world/weather/forecast", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		weather, err := world.ListActiveWeather(c.Request.Context(), save.WorldID, save.ContinentID)
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		forecast := make([]gin.H, 0, len(weather))
		for _, event := range weather {
			forecast = append(forecast, gin.H{
				"id":          event.Id,
				"type":        defaultText(event.Type, "Non disponible"),
				"title":       event.Title,
				"description": event.Description,
				"severity":    clamp(event.Severity, 0, 100),
				"startsAt":    event.StartsAt,
				"endsAt":      event.EndsAt,
			})
		}
		writeWorldResponse(c, gin.H{"forecast": forecast}, nil)
	})

	private.GET("/world/weather/zones", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		weather, err := world.ListActiveWeather(c.Request.Context(), save.WorldID, save.ContinentID)
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		zones := make([]gin.H, 0, len(weather))
		for _, event := range weather {
			zones = append(zones, gin.H{
				"region":   weatherRegionName(database, c.Request.Context(), event.ContinentID),
				"severity": clamp(event.Severity, 0, 100),
				"risk":     riskLabel(event.Severity),
			})
		}
		writeWorldResponse(c, gin.H{"zones": zones}, nil)
	})

	private.POST("/world/weather/actions/:actionKey", func(c *gin.Context) {
		actionKey := strings.TrimSpace(c.Param("actionKey"))
		if actionKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
			return
		}
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		allowed := map[string]bool{"deploy-aid": true, "preposition-resources": true, "activate-defense-protocol": true}
		if !allowed[actionKey] {
			err := world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "weather_action", "weather_action", actionKey, "rejected", "invalid action", nil)
			if err != nil {
				writeWorldResponse(c, nil, err)
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action"})
			return
		}
		payload := bindOptionalMap(c)
		err = world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "weather_action", "weather_action", actionKey, "accepted", "", payload)
		writeWorldResponse(c, gin.H{
			"accepted":  true,
			"actionKey": actionKey,
			"serverNow": time.Now().UTC(),
		}, err)
	})

	private.GET("/world/weather/report", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		weather, err := world.ListActiveWeather(c.Request.Context(), save.WorldID, save.ContinentID)
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		avg := 0
		if len(weather) > 0 {
			total := 0
			for _, event := range weather {
				total += event.Severity
			}
			avg = total / len(weather)
		}
		writeWorldResponse(c, gin.H{
			"report": gin.H{
				"activeEvents":    len(weather),
				"averageSeverity": avg,
				"globalRiskLabel": riskLabel(avg),
				"generatedAt":     time.Now().UTC(),
			},
		}, nil)
	})
}

func fetchCommerceRoutes(database *gorm.DB, world *service.WorldGameService, c *gin.Context) ([]gin.H, error) {
	save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
	if err != nil {
		return nil, err
	}
	conflicts, err := world.ListWorldConflicts(c.Request.Context(), save.WorldID, save.ContinentID, limitFromQuery(c))
	if err != nil {
		return nil, err
	}
	return commerceRoutesFromConflicts(database, c.Request.Context(), conflicts), nil
}

func formatConflictItems(database *gorm.DB, ctx contextLike, conflicts []models.Conflict) []gin.H {
	items := make([]gin.H, 0, len(conflicts))
	for _, conflict := range conflicts {
		attackerName := conflictEntityName(database, ctx, conflict.AttackerType, conflict.AttackerID)
		defenderName := conflictEntityName(database, ctx, conflict.DefenderType, conflict.DefenderID)
		items = append(items, gin.H{
			"id":               strconv.FormatUint(uint64(conflict.Id), 10),
			"title":            defaultText(conflict.Title, "Non disponible"),
			"description":      defaultText(conflict.Description, "Non disponible"),
			"attackerName":     attackerName,
			"attackerType":     defaultText(conflict.AttackerType, "unknown"),
			"defenderName":     defenderName,
			"defenderType":     defaultText(conflict.DefenderType, "unknown"),
			"intensity":        clamp(conflict.Intensity, 0, 100),
			"risk":             clamp(conflict.Intensity, 0, 100),
			"riskLevel":        normalizeRiskLevel(conflict.RiskLevel, conflict.Intensity),
			"status":           defaultText(conflict.Status, "Non disponible"),
			"rewards":          parseJSONObject(conflict.RewardsJSON),
			"penalties":        parseJSONObject(conflict.PenaltiesJSON),
			"availableActions": conflictAvailableActions(conflict.Status),
			"startsAt":         conflict.StartsAt,
			"endsAt":           conflict.EndsAt,
		})
	}
	return items
}

func formatEventItems(database *gorm.DB, ctx contextLike, playerID uint, events []models.GameEvent) []gin.H {
	if len(events) == 0 {
		return []gin.H{}
	}
	eventIDs := make([]uint, 0, len(events))
	for _, event := range events {
		eventIDs = append(eventIDs, event.Id)
	}
	participated := map[uint]bool{}
	claimed := map[uint]bool{}
	if database != nil {
		var participations []models.GameEventParticipation
		_ = database.WithContext(ctx).Where("player_id = ? AND event_id IN ?", playerID, eventIDs).Find(&participations).Error
		for _, item := range participations {
			participated[item.EventID] = true
		}
		var claims []models.GameEventClaim
		_ = database.WithContext(ctx).Where("player_id = ? AND event_id IN ?", playerID, eventIDs).Find(&claims).Error
		for _, item := range claims {
			claimed[item.EventID] = true
		}
	}
	now := time.Now().UTC()
	items := make([]gin.H, 0, len(events))
	for _, event := range events {
		hasParticipated := participated[event.Id]
		isClaimed := claimed[event.Id]
		canClaim := hasParticipated && !isClaimed && !event.EndsAt.After(now)
		items = append(items, gin.H{
			"id":              strconv.FormatUint(uint64(event.Id), 10),
			"title":           defaultText(event.Title, "Non disponible"),
			"description":     defaultText(event.Description, "Non disponible"),
			"type":            defaultText(event.Type, "Non disponible"),
			"difficulty":      defaultText(event.Difficulty, "medium"),
			"status":          defaultText(event.Status, "Non disponible"),
			"startsAt":        event.StartsAt,
			"endsAt":          event.EndsAt,
			"rewards":         parseJSONObject(event.RewardsJSON),
			"requirements":    parseJSONObject(event.RequirementsJSON),
			"consequences":    parseJSONObject(event.ConsequencesJSON),
			"hasParticipated": hasParticipated,
			"canClaim":        canClaim,
			"claimed":         isClaimed,
		})
	}
	return items
}

func formatWeatherItems(events []models.WeatherEvent) []gin.H {
	items := make([]gin.H, 0, len(events))
	for _, event := range events {
		items = append(items, gin.H{
			"id":          strconv.FormatUint(uint64(event.Id), 10),
			"type":        defaultText(event.Type, "Non disponible"),
			"severity":    clamp(event.Severity, 0, 100),
			"title":       defaultText(event.Title, "Non disponible"),
			"description": defaultText(event.Description, "Non disponible"),
			"startsAt":    event.StartsAt,
			"endsAt":      event.EndsAt,
			"effects":     parseJSONObject(event.EffectsJSON),
		})
	}
	return items
}

func parseJSONObject(raw datatypes.JSON) gin.H {
	if len(raw) == 0 {
		return gin.H{}
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed == nil {
		return gin.H{}
	}
	return gin.H(parsed)
}

func conflictAvailableActions(status string) []string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if normalized == "resolved" || normalized == "finished" || normalized == "closed" || normalized == "archived" {
		return []string{}
	}
	return []string{"intervene", "observe"}
}

func conflictEntityName(database *gorm.DB, ctx contextLike, entityType string, entityID uint) string {
	if database == nil || entityID == 0 {
		return "Non disponible"
	}
	switch strings.ToLower(strings.TrimSpace(entityType)) {
	case "continent":
		var continent models.Continent
		if err := database.WithContext(ctx).First(&continent, entityID).Error; err == nil {
			return defaultText(continent.Name, "Non disponible")
		}
	case "ai_faction", "faction":
		var faction models.AIWorldFaction
		if err := database.WithContext(ctx).First(&faction, entityID).Error; err == nil {
			return defaultText(faction.Name, "Non disponible")
		}
	case "guild":
		var guild models.Guild
		if err := database.WithContext(ctx).First(&guild, entityID).Error; err == nil {
			return defaultText(guild.Name, "Non disponible")
		}
	case "player":
		var save models.PlayerSave
		if err := database.WithContext(ctx).First(&save, "player_id = ?", entityID).Error; err == nil {
			return defaultText(save.CityName, "Non disponible")
		}
	}
	return "Non disponible"
}

func commerceRoutesFromConflicts(database *gorm.DB, ctx contextLike, conflicts []models.Conflict) []gin.H {
	routes := make([]gin.H, 0, len(conflicts))
	for _, conflict := range conflicts {
		attackerName := conflictEntityName(database, ctx, conflict.AttackerType, conflict.AttackerID)
		defenderName := conflictEntityName(database, ctx, conflict.DefenderType, conflict.DefenderID)
		volume := int64(1000000 * (100 - clamp(conflict.Intensity, 0, 100)) / 100)
		status := "active"
		if conflict.Intensity >= 70 {
			status = "perturbe"
		}
		route := "Non disponible"
		if attackerName != "Non disponible" && defenderName != "Non disponible" {
			route = attackerName + " -> " + defenderName
		}
		routes = append(routes, gin.H{
			"id":         strconv.FormatUint(uint64(conflict.Id), 10),
			"route":      route,
			"cargo":      "Flux inter-regions",
			"volume":     volume,
			"status":     status,
			"efficiency": clamp(100-conflict.Intensity, 0, 100),
		})
	}
	return routes
}

func normalizeRiskLevel(value string, score int) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "surveille", "modere", "eleve", "critique":
		return strings.ToLower(strings.TrimSpace(value))
	case "low":
		return "surveille"
	case "medium":
		return "modere"
	case "high":
		return "eleve"
	case "critical":
		return "critique"
	default:
		return riskLabel(score)
	}
}

func diplomacyStance(score int) string {
	if score >= 80 {
		return "allie"
	}
	if score >= 60 {
		return "neutre"
	}
	if score >= 35 {
		return "independant"
	}
	return "hostile"
}

func bindOptionalMap(c *gin.Context) gin.H {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil || payload == nil {
		return gin.H{}
	}
	return gin.H(payload)
}

func stableActionID(payload gin.H) string {
	for _, key := range []string{"id", "targetId", "target", "faction", "route"} {
		if value, ok := payload[key]; ok && strings.TrimSpace(toString(value)) != "" {
			return toString(value)
		}
	}
	return "manual"
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int:
		return strconv.Itoa(typed)
	case uint:
		return strconv.FormatUint(uint64(typed), 10)
	default:
		return strings.TrimSpace(defaultText("", ""))
	}
}

func commerceRecommendation(routes []gin.H) string {
	if len(routes) == 0 {
		return "Non disponible"
	}
	for _, route := range routes {
		if status, _ := route["status"].(string); status == "perturbe" {
			return "Stabiliser les routes perturbees avant expansion"
		}
	}
	return "Maintenir les routes actives a fort volume"
}

func weatherRegionName(database *gorm.DB, ctx contextLike, continentID uint) string {
	if database == nil || continentID == 0 {
		return "Non disponible"
	}
	var continent models.Continent
	if err := database.WithContext(ctx).First(&continent, continentID).Error; err != nil {
		return "Non disponible"
	}
	return defaultText(continent.Name, "Non disponible")
}

func riskLabel(severity int) string {
	if severity >= 80 {
		return "critique"
	}
	if severity >= 60 {
		return "eleve"
	}
	if severity >= 35 {
		return "modere"
	}
	return "surveille"
}

func defaultText(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func clamp(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func emissaryAvailability() (available int, total int) {
	// Tant qu'aucun modèle de capacité d'émissaire n'est branché,
	// on renvoie explicitement 0/0 pour éviter des données fictives.
	return 0, 0
}

func registerAdminWorldGameRoutes(group *gin.RouterGroup, database *gorm.DB) {
	registerAdminWorldGameRoutesAt(group.Group("/admin"), database)
}

func registerStrictAdminWorldGameRoutes(group *gin.RouterGroup, database *gorm.DB) {
	registerAdminWorldGameRoutesAt(group, database)
}

func registerAdminWorldGameRoutesAt(admin *gin.RouterGroup, database *gorm.DB) {
	world := service.NewWorldGameService(database)
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
	admin.POST("/worlds/reconcile-counts", func(c *gin.Context) {
		result, err := world.ReconcileWorldPopulationCounts(c.Request.Context())
		writeWorldResponse(c, result, err)
	})
	admin.POST("/worlds/archive-empty", func(c *gin.Context) {
		result, err := world.ArchiveEmptyWorlds(c.Request.Context())
		writeWorldResponse(c, result, err)
	})
	admin.GET("/worlds/:id", adminGet[models.World](database, true))
	admin.PATCH("/worlds/:id", adminPatch[models.World](database, "world"))
	admin.POST("/worlds/:id/simulate", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid world id"})
			return
		}
		decision, err := world.SimulateWorldCycle(c.Request.Context(), id, "admin-api", simulationCycleFromRequest(c))
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
		decision, err := world.SimulateWorldCycle(c.Request.Context(), continent.WorldID, "admin-continent", simulationCycleFromRequest(c))
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
	admin.POST("/events", adminCreateEvent(world))
	admin.GET("/events/:id", adminGet[models.GameEvent](database, false))
	admin.PATCH("/events/:id", adminPatchEvent(world))
	admin.DELETE("/events/:id", adminSoftDelete[models.GameEvent](database, "event"))
	admin.POST("/events/:id/force-start", adminPatchStatus[models.GameEvent](database, "active"))
	admin.POST("/events/:id/force-end", adminPatchStatus[models.GameEvent](database, "finished"))
	admin.POST("/events/generate-ai", adminSimulateWorld(world, 0))

	admin.GET("/conflicts", adminList[models.Conflict](database, "starts_at DESC"))
	admin.POST("/conflicts", adminCreateConflict(world))
	admin.GET("/conflicts/:id", adminGet[models.Conflict](database, false))
	admin.PATCH("/conflicts/:id", adminPatch[models.Conflict](database, "conflict"))
	admin.POST("/conflicts/:id/resolve", adminResolveConflict(world))
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
	admin.PATCH("/ai/decisions/:id", adminPatch[models.AIWorldDecision](database, "ai_decision"))
	admin.POST("/ai/decisions/:id/enable", adminDecisionActivation(database, true))
	admin.POST("/ai/decisions/:id/disable", adminDecisionActivation(database, false))
	admin.POST("/ai/decisions/:id/replay-dry-run", func(c *gin.Context) {
		item, err := adminFindByID[models.AIWorldDecision](c, database)
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		replay, err := world.DryRunWorldSimulation(c.Request.Context(), item.WorldID, item.Type)
		writeWorldResponse(c, gin.H{"dryRun": true, "source": item, "decision": replay}, err)
	})
	admin.GET("/ai/cron", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"cron": scheduler.WorldSimulationCronSnapshot()})
	})
	admin.POST("/ai/cron/enable", adminWorldCronActivation(database, true))
	admin.POST("/ai/cron/disable", adminWorldCronActivation(database, false))
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
	admin.GET("/resources", adminList[models.ResourceDefinition](database, "sort_order ASC, id ASC"))
	admin.POST("/resources", adminCreate[models.ResourceDefinition](database, "resource"))
	admin.GET("/resources/:id", adminGet[models.ResourceDefinition](database, false))
	admin.PATCH("/resources/:id", adminPatch[models.ResourceDefinition](database, "resource"))
	admin.DELETE("/resources/:id", adminSoftDelete[models.ResourceDefinition](database, "resource"))
	admin.GET("/research-trees", adminList[models.ResearchTreeDefinition](database, "sort_order ASC, id ASC"))
	admin.POST("/research-trees", adminCreate[models.ResearchTreeDefinition](database, "research_tree"))
	admin.GET("/research-trees/:id", adminGet[models.ResearchTreeDefinition](database, true))
	admin.PATCH("/research-trees/:id", adminPatch[models.ResearchTreeDefinition](database, "research_tree"))
	admin.DELETE("/research-trees/:id", adminSoftDelete[models.ResearchTreeDefinition](database, "research_tree"))
	admin.GET("/research-nodes", adminList[models.ResearchNodeDefinition](database, "sort_order ASC, id ASC"))
	admin.POST("/research-nodes", adminCreate[models.ResearchNodeDefinition](database, "research_node"))
	admin.GET("/research-nodes/:id", adminGet[models.ResearchNodeDefinition](database, false))
	admin.PATCH("/research-nodes/:id", adminPatch[models.ResearchNodeDefinition](database, "research_node"))
	admin.DELETE("/research-nodes/:id", adminSoftDelete[models.ResearchNodeDefinition](database, "research_node"))
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
		private.GET("/chat/"+ch+"/stream", streamChatChannel(world, ch))
	}
}

func streamChatChannel(world *service.WorldGameService, channel string) gin.HandlerFunc {
	return func(c *gin.Context) {
		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming unsupported"})
			return
		}
		afterRaw, _ := strconv.ParseUint(c.DefaultQuery("after", "0"), 10, 64)
		afterID := uint(afterRaw)
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		deadline := time.NewTimer(2 * time.Minute)
		defer deadline.Stop()
		_ = writeSSEEvent(c, flusher, "ready", gin.H{"channel": channel, "after": afterID})
		for {
			messages, err := world.ListChatAfter(c.Request.Context(), currentUserID(c), channel, afterID, 50)
			if err != nil {
				_ = writeSSEEvent(c, flusher, "error", gin.H{"error": err.Error()})
				return
			}
			for _, message := range messages {
				if message.Id > afterID {
					afterID = message.Id
				}
				if err := writeSSEEvent(c, flusher, "message", message); err != nil {
					return
				}
			}
			select {
			case <-ticker.C:
			case <-deadline.C:
				_ = writeSSEEvent(c, flusher, "done", gin.H{"after": afterID})
				return
			case <-c.Request.Context().Done():
				return
			}
		}
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
		invite, err := world.InviteGuildMember(c.Request.Context(), currentUserID(c), id, input.PlayerID)
		writeWorldResponse(c, invite, err)
	})
	private.POST("/guilds/invites/:id/accept", guildInviteStatus(world, true))
	private.POST("/guilds/invites/:id/decline", guildInviteStatus(world, false))
	private.POST("/guilds/:id/members/:playerId/role", func(c *gin.Context) {
		id, err1 := parseUintParam(c, "id")
		playerID, err2 := parseUintParam(c, "playerId")
		if err1 != nil || err2 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild member id"})
			return
		}
		var input struct {
			Role string `json:"role"`
		}
		_ = bindPayload(c, &input)
		writeWorldResponse(c, gin.H{"role": input.Role}, world.ChangeGuildMemberRole(c.Request.Context(), currentUserID(c), id, playerID, input.Role))
	})
	private.DELETE("/guilds/:id/members/:playerId", func(c *gin.Context) {
		id, err1 := parseUintParam(c, "id")
		playerID, err2 := parseUintParam(c, "playerId")
		if err1 != nil || err2 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild member id"})
			return
		}
		writeWorldResponse(c, gin.H{"deleted": true}, world.RemoveGuildMember(c.Request.Context(), currentUserID(c), id, playerID))
	})
	private.POST("/guilds/:id/contribute", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid guild id"})
			return
		}
		var input service.GuildContributionInput
		_ = bindPayload(c, &input)
		contribution, err := world.ContributeGuild(c.Request.Context(), currentUserID(c), id, input)
		writeWorldResponse(c, contribution, err)
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
	private.GET("/buildings/:key/research-tree", func(c *gin.Context) {
		payload, err := world.ResearchCatalog(c.Request.Context(), currentUserID(c), c.Param("key"))
		writeWorldResponse(c, payload, err)
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

func registerResearchRoutes(private *gin.RouterGroup, world *service.WorldGameService) {
	private.GET("/research/catalog", func(c *gin.Context) {
		payload, err := world.ResearchCatalog(c.Request.Context(), currentUserID(c), c.Query("buildingKey"))
		writeWorldResponse(c, payload, err)
	})
	private.GET("/research/state", func(c *gin.Context) {
		progress, err := world.ResearchProgress(c.Request.Context(), currentUserID(c))
		writeWorldResponse(c, progress, err)
	})
	private.POST("/research/:nodeKey/start", func(c *gin.Context) {
		result, err := world.StartResearch(c.Request.Context(), currentUserID(c), c.Param("nodeKey"))
		writeWorldResponse(c, result, err)
	})
	private.POST("/research/:nodeKey/complete", func(c *gin.Context) {
		result, err := world.CompleteResearch(c.Request.Context(), currentUserID(c), c.Param("nodeKey"))
		writeWorldResponse(c, result, err)
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

func guildInviteStatus(world *service.WorldGameService, accept bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid invite id"})
			return
		}
		err = world.RespondGuildInvite(c.Request.Context(), currentUserID(c), id, accept)
		status := "declined"
		if accept {
			status = "accepted"
		}
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
			"charts":         adminDashboardCharts(c, database),
			"generatedAt":    time.Now(),
		})
	}
}

type adminChartPoint struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
	Value int64  `json:"value,omitempty"`
}

func adminDashboardCharts(c *gin.Context, database *gorm.DB) gin.H {
	now := time.Now()
	ctx := c.Request.Context()
	chatActivity := make([]adminChartPoint, 0)
	_ = database.WithContext(ctx).Model(&models.ChatMessage{}).
		Select("channel_type as label, COUNT(*) as count").
		Where("created_at >= ?", now.Add(-7*24*time.Hour)).
		Group("channel_type").
		Order("channel_type ASC").
		Scan(&chatActivity).Error
	playerGrowth := make([]adminChartPoint, 0)
	_ = database.WithContext(ctx).Model(&models.PlayerSave{}).
		Select("DATE(created_at) as label, COUNT(*) as count").
		Where("created_at >= ?", now.Add(-14*24*time.Hour)).
		Group("DATE(created_at)").
		Order("DATE(created_at) ASC").
		Scan(&playerGrowth).Error
	conflictsByIntensity := make([]adminChartPoint, 0)
	_ = database.WithContext(ctx).Model(&models.Conflict{}).
		Select("CASE WHEN intensity >= 80 THEN 'critical' WHEN intensity >= 60 THEN 'high' WHEN intensity >= 30 THEN 'medium' ELSE 'low' END as label, COUNT(*) as count").
		Group("label").
		Order("label ASC").
		Scan(&conflictsByIntensity).Error
	weatherBySeverity := make([]adminChartPoint, 0)
	_ = database.WithContext(ctx).Model(&models.WeatherEvent{}).
		Select("CASE WHEN severity >= 80 THEN 'critical' WHEN severity >= 60 THEN 'high' WHEN severity >= 30 THEN 'medium' ELSE 'low' END as label, COUNT(*) as count").
		Group("label").
		Order("label ASC").
		Scan(&weatherBySeverity).Error
	var totals struct {
		Food    int64
		Energy  int64
		Credits int64
		Gems    int64
	}
	_ = database.WithContext(ctx).Model(&models.PlayerSave{}).
		Select("COALESCE(SUM(food),0) as food, COALESCE(SUM(energy),0) as energy, COALESCE(SUM(credits),0) as credits, COALESCE(SUM(gems),0) as gems").
		Scan(&totals).Error
	rewardsClaimed := make([]adminChartPoint, 0)
	_ = database.WithContext(ctx).Model(&models.GameEventClaim{}).
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
		"resources": []adminChartPoint{
			{Label: "food", Value: totals.Food},
			{Label: "energy", Value: totals.Energy},
			{Label: "credits", Value: totals.Credits},
			{Label: "gems", Value: totals.Gems},
		},
		"rewardsClaimed": rewardsClaimed,
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

func adminDecisionActivation(database *gorm.DB, active bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var before models.AIWorldDecision
		if err := database.WithContext(c.Request.Context()).First(&before, id).Error; err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		err = database.WithContext(c.Request.Context()).Model(&models.AIWorldDecision{}).Where("id = ?", id).Update("is_active", active).Error
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		var after models.AIWorldDecision
		_ = database.WithContext(c.Request.Context()).First(&after, id).Error
		action := "enable"
		if !active {
			action = "disable"
		}
		writeAudit(c, database, action, "ai_decision", strconv.FormatUint(uint64(id), 10), before, after)
		writeWorldResponse(c, gin.H{"id": id, "isActive": active, "decision": after}, nil)
	}
}

func adminWorldCronActivation(database *gorm.DB, enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		snapshot := scheduler.SetWorldSimulationCronEnabled(enabled, "admin-api")
		action := "enable_world_ai_cron"
		if !enabled {
			action = "disable_world_ai_cron"
		}
		writeAudit(c, database, action, "world_ai_cron", "runtime", nil, snapshot)
		writeWorldResponse(c, gin.H{"cron": snapshot}, nil)
	}
}

func adminSimulateWorld(world *service.WorldGameService, worldID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		decision, err := world.SimulateWorldCycle(c.Request.Context(), worldID, "admin-api", simulationCycleFromRequest(c))
		writeWorldResponse(c, decision, err)
	}
}

func simulationCycleFromRequest(c *gin.Context) string {
	if cycle := strings.TrimSpace(c.Query("cycleType")); cycle != "" {
		return cycle
	}
	var payload struct {
		CycleType string `json:"cycleType"`
	}
	_ = c.ShouldBindJSON(&payload)
	return payload.CycleType
}

func adminPlayers(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var saves []models.PlayerSave
		err := database.WithContext(c.Request.Context()).Preload("World").Preload("Continent").Order("updated_at DESC").Limit(limitFromQuery(c)).Find(&saves).Error
		writeWorldResponse(c, gin.H{"items": saves}, err)
	}
}

func adminCreateEvent(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var event models.GameEvent
		if err := c.ShouldBindJSON(&event); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event payload"})
			return
		}
		writeWorldResponse(c, event, world.CreateGameEvent(c.Request.Context(), &event))
	}
}

func adminCreateConflict(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input service.ConflictInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conflict payload"})
			return
		}
		conflict, err := world.CreateConflict(c.Request.Context(), input)
		writeWorldResponse(c, conflict, err)
	}
}

func adminResolveConflict(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conflict id"})
			return
		}
		writeWorldResponse(c, gin.H{"status": service.ConflictStatusResolved}, world.ResolveConflict(c.Request.Context(), id, "admin-api"))
	}
}

func adminPatchEvent(world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
			return
		}
		var fields map[string]any
		if err := c.ShouldBindJSON(&fields); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event payload"})
			return
		}
		delete(fields, "id")
		delete(fields, "Id")
		writeWorldResponse(c, gin.H{"updated": true}, world.UpdateGameEvent(c.Request.Context(), id, fields))
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
		if strings.Contains(c.GetHeader("Content-Type"), "multipart/form-data") {
			file, header, err := c.Request.FormFile("image")
			if err != nil {
				writeWorldResponse(c, nil, err)
				return
			}
			defer file.Close()
			level, _ := strconv.Atoi(c.DefaultPostForm("level", "1"))
			version, _ := strconv.Atoi(c.DefaultPostForm("version", "0"))
			asset, err := world.SaveBuildingAssetUpload(c.Request.Context(), buildingID, level, c.DefaultPostForm("variant", "default"), version, header.Filename, file)
			writeWorldResponse(c, asset, err)
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
	for _, key := range []string{"world_id", "continent_id", "guild_id", "player_id", "status", "is_active", "type", "channel_type", "domain", "building_key", "research_tree_definition_id"} {
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
