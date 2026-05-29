package router

import (
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerWorldPhase2Routes(private *gin.RouterGroup, database *gorm.DB, world *service.WorldGameService) {
	registerDiplomacyRoutes(private, database, world)
	registerTradeRoutes(private, database, world)
	registerWeatherRoutes(private, database, world)
	registerWorldRegionAndReportsRoutes(private, database, world)
}

func registerDiplomacyRoutes(private *gin.RouterGroup, database *gorm.DB, world *service.WorldGameService) {
	private.GET("/diplomacy/overview", func(c *gin.Context) {
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
				"id":                 strconv.FormatUint(uint64(continent.Id), 10),
				"targetId":           strconv.FormatUint(uint64(continent.Id), 10),
				"targetName":         defaultText(continent.Name, "Non disponible"),
				"targetType":         "continent",
				"relationScore":      score,
				"status":             diplomacyStance(score),
				"trust":              score,
				"hostility":          100 - score,
				"activeTreaties":     0,
				"activeNegotiations": 0,
				"risks":              []string{},
				"history":            []string{},
				// Aliases for Flutter parsers expecting faction/stance
				"faction": defaultText(continent.Name, "Non disponible"),
				"stance":  diplomacyStance(score),
			})
		}
		repLabel := "Neutre"
		repScore := 50
		writeWorldResponse(c, gin.H{
			"relations": relations,
			"reputation": gin.H{
				"score":         repScore,
				"label":         repLabel,
				"bonus":         []string{},
				"malus":         []string{},
				"explanation":   "Réputation diplomatique calculée sur les actions récentes.",
				"recentHistory": []string{},
			},
		}, nil)
	})

	private.GET("/diplomacy/relations", func(c *gin.Context) {
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
				"id":            strconv.FormatUint(uint64(continent.Id), 10),
				"targetId":      strconv.FormatUint(uint64(continent.Id), 10),
				"targetName":    defaultText(continent.Name, "Non disponible"),
				"targetType":    "continent",
				"relationScore": score,
				"status":        diplomacyStance(score),
				"trust":         score,
				"hostility":     100 - score,
				// Aliases for Flutter parsers expecting faction/stance
				"faction": defaultText(continent.Name, "Non disponible"),
				"stance":  diplomacyStance(score),
			})
		}
		writeWorldResponse(c, gin.H{"relations": relations}, nil)
	})

	private.GET("/diplomacy/treaties", func(c *gin.Context) {
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
					"id":          strconv.FormatUint(uint64(event.Id), 10),
					"title":       event.Title,
					"description": event.Description,
					"type":        "temporary_truce",
					"signatories": []string{"player", "continent"},
					"startsAt":    event.StartsAt,
					"endsAt":      event.EndsAt,
					"status":      event.Status,
					"bonuses":     gin.H{},
					"obligations": []string{},
					"penalties":   gin.H{},
					"cost":        gin.H{},
					"conditions":  []string{},
				})
			}
		}
		writeWorldResponse(c, gin.H{"treaties": treaties}, nil)
	})

	private.POST("/diplomacy/treaties/:id/accept", diplomacyTreatyAction(database, world, "accept"))
	private.POST("/diplomacy/treaties/:id/reject", diplomacyTreatyAction(database, world, "reject"))
	private.POST("/diplomacy/treaties/:id/break", diplomacyTreatyAction(database, world, "break"))

	private.GET("/diplomacy/negotiations", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		var logs []models.PlayerActionLog
		err = database.WithContext(c.Request.Context()).
			Where("player_id = ? AND world_id = ? AND action IN ?", save.PlayerID, save.WorldID, []string{"diplomacy_negotiation_open", "diplomacy_negotiation_action"}).
			Order("created_at DESC").
			Limit(limitFromQuery(c)).
			Find(&logs).Error
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		items := make([]gin.H, 0, len(logs))
		for _, log := range logs {
			items = append(items, gin.H{
				"id":              strconv.FormatUint(uint64(log.Id), 10),
				"targetId":        log.TargetID,
				"targetName":      "Négociation",
				"targetType":      "continent",
				"subject":         "Diplomatie",
				"objective":       "Améliorer la relation",
				"status":          "in_progress",
				"progress":        50,
				"durationSeconds": 3600,
				"startsAt":        log.CreatedAt,
				"finishesAt":      log.CreatedAt.Add(time.Hour),
				"cost":            gin.H{},
				"successChance":   0.6,
				"risks":           []string{},
				"choices":         []string{"send_emissary", "offer_resources", "propose_treaty"},
				"finalResult":     nil,
			})
		}
		writeWorldResponse(c, gin.H{"negotiations": items}, nil)
	})

	private.POST("/diplomacy/negotiations/start", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		payload := bindOptionalMap(c)
		targetID := strings.TrimSpace(toString(payload["targetId"]))
		if targetID == "" {
			writeWorldResponse(c, nil, badRequestError("TARGET_REQUIRED", "La cible de négociation est obligatoire.", nil))
			return
		}
		if cooldownActive(database, c, save.PlayerID, "diplomacy_negotiation_open", targetID, 5*time.Minute) {
			writeWorldResponse(c, nil, conflictError("COOLDOWN_ACTIVE", "Négociation en cooldown, réessayez plus tard.", map[string]any{"cooldownSeconds": 300}))
			return
		}
		actionID := stableActionID(payload)
		err = world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "diplomacy_negotiation_open", "diplomacy", actionID, "accepted", "", payload)
		writeWorldResponse(c, gin.H{"started": true, "id": actionID, "status": "in_progress"}, err)
	})

	private.POST("/diplomacy/negotiations/:id/action", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		payload := bindOptionalMap(c)
		actionType := strings.TrimSpace(strings.ToLower(toString(payload["action"])))
		if !isAllowedNegotiationAction(actionType) {
			writeWorldResponse(c, nil, badRequestError("INVALID_NEGOTIATION_ACTION", "Action de négociation invalide.", map[string]any{"allowed": allowedNegotiationActions()}))
			return
		}
		err = world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "diplomacy_negotiation_action", "diplomacy", c.Param("id"), "accepted", "", payload)
		writeWorldResponse(c, gin.H{"applied": true, "id": c.Param("id")}, err)
	})

	private.GET("/diplomacy/emissaries", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		available, total := emissaryAvailability(database, c.Request.Context(), save)
		items := make([]gin.H, 0, total)
		for i := 0; i < total; i++ {
			status := "on_mission"
			if i < available {
				status = "available"
			}
			items = append(items, gin.H{
				"id":               i + 1,
				"name":             "Émissaire " + strconv.Itoa(i+1),
				"level":            1,
				"experience":       0,
				"specialty":        "paix",
				"status":           status,
				"currentMission":   nil,
				"target":           nil,
				"remainingSeconds": 0,
				"bonuses":          []string{},
				"risk":             "low",
			})
		}
		writeWorldResponse(c, gin.H{"emissaries": items}, nil)
	})

	private.POST("/diplomacy/emissaries/send", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		available, _ := emissaryAvailability(database, c.Request.Context(), save)
		if available <= 0 {
			writeWorldResponse(c, nil, conflictError("EMISSARY_NOT_AVAILABLE", "Aucun émissaire disponible.", nil))
			return
		}
		payload := bindOptionalMap(c)
		target := strings.TrimSpace(toString(payload["targetContinentId"]))
		if target == "" {
			writeWorldResponse(c, nil, badRequestError("TARGET_REQUIRED", "La cible de mission est obligatoire.", nil))
			return
		}
		missionType := strings.TrimSpace(strings.ToLower(toString(payload["missionType"])))
		if !isAllowedEmissaryMission(missionType) {
			writeWorldResponse(c, nil, badRequestError("INVALID_MISSION_TYPE", "Type de mission invalide.", map[string]any{"allowed": allowedEmissaryMissions()}))
			return
		}
		if cooldownActive(database, c, save.PlayerID, "diplomacy_emissary_send", target, 10*time.Minute) {
			writeWorldResponse(c, nil, conflictError("COOLDOWN_ACTIVE", "Émissaire en cooldown, réessayez plus tard.", map[string]any{"cooldownSeconds": 600}))
			return
		}
		actionID := stableActionID(payload)
		err = world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "diplomacy_emissary_send", "emissary", actionID, "accepted", "", payload)
		writeWorldResponse(c, gin.H{"sent": true, "actionId": actionID, "status": "on_mission"}, err)
	})

	private.GET("/diplomacy/reports", func(c *gin.Context) {
		messages, err := world.ListDailyMessages(c.Request.Context(), currentUserID(c), limitFromQuery(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		reports := make([]gin.H, 0, len(messages))
		for _, message := range messages {
			reports = append(reports, gin.H{
				"id":                 message.Id,
				"type":               "diplomatic_report",
				"title":              message.Title,
				"summary":            message.Message,
				"details":            message.Message,
				"createdAt":          message.CreatedAt,
				"importance":         "info",
				"recommendedActions": []string{},
				"metrics":            gin.H{},
			})
		}
		writeWorldResponse(c, gin.H{"reports": reports}, nil)
	})

	// Aggregated targets for "Nouvel accord", émissaires, négociations (regions + guildes + IA)
	private.GET("/diplomacy/targets", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		ctx := c.Request.Context()

		// Regions / Continents (exclude own)
		var continents []models.Continent
		_ = database.WithContext(ctx).Where("world_id = ?", save.WorldID).Order("`index` ASC").Find(&continents).Error
		targets := make([]gin.H, 0, 32)
		for _, ct := range continents {
			if ct.Id == save.ContinentID {
				continue
			}
			score := clamp(100-ct.TensionLevel, 0, 100)
			targets = append(targets, gin.H{
				"id":       "r_" + strconv.FormatUint(uint64(ct.Id), 10),
				"rawId":    ct.Id,
				"name":     defaultText(ct.Name, "Région"),
				"type":     "region",
				"score":    score,
				"status":   diplomacyStance(score),
			})
		}

		// Guilds (ListGuilds seeds samples if needed)
		guilds, _ := world.ListGuilds(ctx, currentUserID(c), 50)
		for _, g := range guilds {
			targets = append(targets, gin.H{
				"id":    "g_" + strconv.FormatUint(uint64(g.Id), 10),
				"rawId": g.Id,
				"name":  defaultText(g.Name, g.Tag),
				"type":  "guild",
				"tag":   g.Tag,
			})
		}

		// AI World Factions as targets
		var aiFactions []models.AIWorldFaction
		_ = database.WithContext(ctx).Where("world_id = ?", save.WorldID).Limit(20).Find(&aiFactions).Error
		for _, a := range aiFactions {
			targets = append(targets, gin.H{
				"id":    "ai_" + strconv.FormatUint(uint64(a.Id), 10),
				"rawId": a.Id,
				"name":  defaultText(a.Name, "IA"),
				"type":  "ai",
				"aggro": a.Aggressiveness,
			})
		}

		// Fallback synthetic targets if still nothing (never empty)
		if len(targets) == 0 {
			targets = append(targets,
				gin.H{"id": "r_fallback1", "name": "Région voisine", "type": "region"},
				gin.H{"id": "g_fallback1", "name": "Guilde Marchande", "type": "guild"},
				gin.H{"id": "ai_fallback1", "name": "IA Locale", "type": "ai"},
			)
		}

		writeWorldResponse(c, gin.H{"targets": targets, "total": len(targets)}, nil)
	})
}

func diplomacyTreatyAction(database *gorm.DB, world *service.WorldGameService, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		id, parseErr := parseUintParam(c, "id")
		if parseErr != nil {
			writeWorldResponse(c, nil, badRequestError("INVALID_TREATY_ID", "Identifiant de traité invalide.", nil))
			return
		}
		var event models.GameEvent
		if err := database.WithContext(c.Request.Context()).Where("id = ? AND world_id = ?", id, save.WorldID).First(&event).Error; err != nil {
			writeWorldResponse(c, nil, notFoundError("TREATY_NOT_FOUND", "Traité introuvable.", nil))
			return
		}
		if event.EndsAt.Before(time.Now().UTC()) && action != "break" {
			writeWorldResponse(c, nil, conflictError("TREATY_EXPIRED", "Ce traité a expiré.", nil))
			return
		}
		err = world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "diplomacy_treaty_"+action, "treaty", c.Param("id"), "accepted", "", nil)
		writeWorldResponse(c, gin.H{"status": action, "treatyId": c.Param("id")}, err)
	}
}

func registerTradeRoutes(private *gin.RouterGroup, database *gorm.DB, world *service.WorldGameService) {
	private.GET("/trade/overview", func(c *gin.Context) {
		routes, err := fetchCommerceRoutes(database, world, c)
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		exports := int64(0)
		imports := int64(0)
		for _, route := range routes {
			v, _ := route["volume"].(int64)
			if route["status"] == "active" {
				exports += v
			} else {
				imports += v
			}
		}
		balance := exports - imports
		status := "balanced"
		if balance > 0 {
			status = "positive"
		} else if balance < 0 {
			status = "negative"
		}
		writeWorldResponse(c, gin.H{
			"exportsValue24h": exports,
			"importsValue24h": imports,
			"balance":         balance,
			"balanceStatus":   status,
			"explanation":     "Solde commercial sur 24h.",
			"impacts":         []string{},
		}, nil)
	})

	private.GET("/trade/routes", func(c *gin.Context) {
		routes, err := fetchCommerceRoutes(database, world, c)
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		items := make([]gin.H, 0, len(routes))
		for _, route := range routes {
			items = append(items, gin.H{
				"id":         route["id"],
				"from":       "continent_a",
				"to":         "continent_b",
				"cargo":      route["cargo"],
				"volume24h":  route["volume"],
				"efficiency": route["efficiency"],
				"profit24h":  route["volume"],
				"status":     route["status"],
				"risk":       riskLabel(100 - route["efficiency"].(int)),
				"duration":   86400,
				"protection": "standard",
				"ownerType":  "player",
				"targetType": "continent",
			})
		}
		writeWorldResponse(c, gin.H{"routes": items}, nil)
	})

	private.POST("/trade/routes/create", func(c *gin.Context) {
		writeWorldResponse(c, gin.H{"created": true, "status": "negotiating"}, nil)
	})
	private.POST("/trade/routes/:id/optimize", func(c *gin.Context) {
		if err := validateRouteOwnership(database, world, c, c.Param("id")); err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		writeWorldResponse(c, gin.H{"optimized": true, "id": c.Param("id")}, nil)
	})
	private.POST("/trade/routes/:id/pause", func(c *gin.Context) {
		if err := validateRouteOwnership(database, world, c, c.Param("id")); err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		writeWorldResponse(c, gin.H{"paused": true, "id": c.Param("id")}, nil)
	})
	private.POST("/trade/routes/:id/protect", func(c *gin.Context) {
		if err := validateRouteOwnership(database, world, c, c.Param("id")); err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		writeWorldResponse(c, gin.H{"protected": true, "id": c.Param("id")}, nil)
	})

	// Cancel a trade route / commercial action (supports optimistic local ops and future real routes)
	private.POST("/trade/routes/:id/cancel", func(c *gin.Context) {
		routeID := strings.TrimSpace(c.Param("id"))
		if routeID == "" {
			writeWorldResponse(c, nil, badRequestError("INVALID_ROUTE_ID", "Identifiant de route invalide.", nil))
			return
		}

		// validateRouteOwnership is best-effort (may be no-op for local optimistic IDs)
		if err := validateRouteOwnership(database, world, c, routeID); err != nil {
			// For local/commerce optimistic IDs we still allow cancel client-side
			// so we don't return error here for non-existing routes
		}

		// Log the action for history (even if route is not a real DB record yet)
		save, _ := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if save != nil {
			_ = world.LogPlayerWorldAction(
				c.Request.Context(),
				currentUserID(c),
				save.WorldID,
				save.ContinentID,
				"commerce_route_cancel",
				"trade",
				routeID,
				"cancelled",
				"",
				gin.H{"routeId": routeID},
			)
		}

		writeWorldResponse(c, gin.H{"cancelled": true, "id": routeID, "status": "cancelled"}, nil)
	})

	private.GET("/trade/agreements", func(c *gin.Context) {
		// Synthesize visible agreements from recent player commerce actions (newAgreement etc.)
		agreements := synthesizePlayerAgreementsFromLogs(database, c)
		writeWorldResponse(c, gin.H{"agreements": agreements}, nil)
	})
	private.POST("/trade/agreements/:id/accept", func(c *gin.Context) {
		if strings.TrimSpace(c.Param("id")) == "" {
			writeWorldResponse(c, nil, badRequestError("INVALID_AGREEMENT_ID", "Identifiant d'accord invalide.", nil))
			return
		}
		writeWorldResponse(c, gin.H{"status": "accepted", "id": c.Param("id")}, nil)
	})
	private.POST("/trade/agreements/:id/reject", func(c *gin.Context) {
		if strings.TrimSpace(c.Param("id")) == "" {
			writeWorldResponse(c, nil, badRequestError("INVALID_AGREEMENT_ID", "Identifiant d'accord invalide.", nil))
			return
		}
		writeWorldResponse(c, gin.H{"status": "rejected", "id": c.Param("id")}, nil)
	})
	private.POST("/trade/agreements/:id/negotiate", func(c *gin.Context) {
		if strings.TrimSpace(c.Param("id")) == "" {
			writeWorldResponse(c, nil, badRequestError("INVALID_AGREEMENT_ID", "Identifiant d'accord invalide.", nil))
			return
		}
		writeWorldResponse(c, gin.H{"status": "negotiating", "id": c.Param("id")}, nil)
	})

	private.GET("/trade/reports", func(c *gin.Context) {
		routes, err := fetchCommerceRoutes(database, world, c)
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		writeWorldResponse(c, gin.H{
			"reports": []gin.H{{
				"id":                 "trade-report",
				"type":               "economic_report",
				"title":              "Rapport commerce",
				"summary":            "État des routes et flux 24h.",
				"details":            routes,
				"createdAt":          time.Now().UTC(),
				"importance":         "info",
				"recommendedActions": []string{},
				"metrics":            gin.H{"routesCount": len(routes)},
			}},
		}, nil)
	})
}

func registerWeatherRoutes(private *gin.RouterGroup, database *gorm.DB, world *service.WorldGameService) {
	private.GET("/weather/overview", func(c *gin.Context) {
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
			sum := 0
			for _, item := range weather {
				sum += clamp(item.Severity, 0, 100)
			}
			avg = sum / len(weather)
		}
		writeWorldResponse(c, gin.H{"activeEvents": len(weather), "averageSeverity": avg, "globalRisk": riskLabel(avg)}, nil)
	})

	private.GET("/weather/events", func(c *gin.Context) {
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
		writeWorldResponse(c, gin.H{"events": formatWeatherItems(weather)}, nil)
	})

	private.GET("/weather/risks", func(c *gin.Context) {
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
		risks := make([]gin.H, 0, len(weather))
		for _, item := range weather {
			risks = append(risks, gin.H{"id": item.Id, "type": item.Type, "risk": riskLabel(item.Severity), "severity": item.Severity})
		}
		writeWorldResponse(c, gin.H{"risks": risks}, nil)
	})

	private.GET("/weather/plans", func(c *gin.Context) {
		plans := []gin.H{
			weatherActionPlan("deploy-aid"),
			weatherActionPlan("preposition-resources"),
			weatherActionPlan("activate-defense-protocol"),
		}
		writeWorldResponse(c, gin.H{"plans": plans}, nil)
	})

	private.POST("/weather/plans/:id/start", func(c *gin.Context) {
		raw := strings.TrimSpace(c.Param("id"))
		// Normalize common aliases / old keys coming from UI fallbacks
		actionKey := raw
		switch raw {
		case "preposition", "preposition-resources":
			actionKey = "preposition-resources"
		case "defense-protocol", "activate-defense-protocol":
			actionKey = "activate-defense-protocol"
		case "deploy-aid", "deployAid":
			actionKey = "deploy-aid"
		}
		if actionKey == "" {
			writeWorldResponse(c, nil, badRequestError("INVALID_PLAN_ID", "Identifiant de plan invalide.", nil))
			return
		}
		plan := weatherActionPlan(actionKey)
		if len(plan) == 0 {
			writeWorldResponse(c, nil, badRequestError("UNKNOWN_PLAN", "Plan météo inconnu.", nil))
			return
		}

		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		if cooldownActive(database, c, save.PlayerID, "weather_plan_start", actionKey, 15*time.Minute) {
			writeWorldResponse(c, nil, conflictError("COOLDOWN_ACTIVE", "Plan météo en cooldown.", map[string]any{"cooldownSeconds": 900}))
			return
		}
		costMap, _ := plan["cost"].(gin.H)
		creditsCost := toInt64(costMap["credits"])
		foodCost := toInt64(costMap["food"])
		energyCost := toInt64(costMap["energy"])
		if save.Credits < creditsCost || save.Food < foodCost || save.Energy < energyCost {
			writeWorldResponse(c, nil, conflictError("NOT_ENOUGH_RESOURCES", "Ressources insuffisantes pour démarrer ce plan météo.", map[string]any{
				"missingFood":    maxInt64Phase2(0, foodCost-save.Food),
				"missingEnergy":  maxInt64Phase2(0, energyCost-save.Energy),
				"missingCredits": maxInt64Phase2(0, creditsCost-save.Credits),
			}))
			return
		}
		err = database.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.PlayerSave{}).Where("id = ?", save.Id).Updates(map[string]any{
				"credits": save.Credits - creditsCost,
				"food":    save.Food - foodCost,
				"energy":  save.Energy - energyCost,
			}).Error; err != nil {
				return err
			}
			return world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "weather_plan_start", "weather_plan", actionKey, "accepted", "", gin.H{"cost": costMap})
		})
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		now := time.Now().UTC()
		durationMinutes, _ := plan["durationMinutes"].(int)
		writeWorldResponse(c, gin.H{"id": actionKey, "status": "running", "startsAt": now, "finishesAt": now.Add(time.Duration(durationMinutes) * time.Minute), "plan": plan}, nil)
	})

	private.GET("/weather/reports", func(c *gin.Context) {
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
		writeWorldResponse(c, gin.H{
			"reports": []gin.H{{
				"id":                 "weather-report",
				"type":               "weather_report",
				"title":              "Rapport météo",
				"summary":            "Événements météo actifs et risques.",
				"details":            formatWeatherItems(weather),
				"createdAt":          time.Now().UTC(),
				"importance":         "warning",
				"recommendedActions": []string{},
				"metrics":            gin.H{"activeEvents": len(weather)},
			}},
		}, nil)
	})

	_ = database
}

func isAllowedNegotiationAction(actionType string) bool {
	for _, action := range allowedNegotiationActions() {
		if action == actionType {
			return true
		}
	}
	return false
}

func allowedNegotiationActions() []string {
	return []string{"send_emissary", "offer_resources", "threaten", "propose_treaty", "request_alliance", "request_ceasefire", "negotiate_trade_route", "request_military_support"}
}

func isAllowedEmissaryMission(missionType string) bool {
	for _, mission := range allowedEmissaryMissions() {
		if mission == missionType {
			return true
		}
	}
	return false
}

func allowedEmissaryMissions() []string {
	return []string{"commerce", "paix", "intimidation", "alliance", "espionnage", "crise", "guilde"}
}

func cooldownActive(database *gorm.DB, c *gin.Context, playerID uint, action string, targetID string, d time.Duration) bool {
	if database == nil {
		return false
	}
	var count int64
	_ = database.WithContext(c.Request.Context()).Model(&models.PlayerActionLog{}).
		Where("player_id = ? AND action = ? AND target_id = ? AND status = ? AND created_at >= ?", playerID, action, targetID, "accepted", time.Now().UTC().Add(-d)).
		Count(&count).Error
	return count > 0
}

func validateRouteOwnership(database *gorm.DB, world *service.WorldGameService, c *gin.Context, routeID string) error {
	routes, err := fetchCommerceRoutes(database, world, c)
	if err != nil {
		return err
	}
	for _, route := range routes {
		if toString(route["id"]) == strings.TrimSpace(routeID) {
			return nil
		}
	}
	return notFoundError("ROUTE_NOT_FOUND", "Route commerciale introuvable.", map[string]any{"routeId": routeID})
}

func toInt64(value any) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func maxInt64Phase2(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func registerWorldRegionAndReportsRoutes(private *gin.RouterGroup, database *gorm.DB, world *service.WorldGameService) {
	private.GET("/world/regions", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		var continents []models.Continent
		if err := database.WithContext(c.Request.Context()).Where("world_id = ?", save.WorldID).Order("`index` ASC").Find(&continents).Error; err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		items := make([]gin.H, 0, len(continents))
		for _, continent := range continents {
			status := "stable"
			if continent.TensionLevel >= 70 {
				status = "at_war"
			} else if continent.TensionLevel >= 40 {
				status = "tense"
			}
			items = append(items, gin.H{
				"id":                  continent.Id,
				"name":                defaultText(continent.Name, "Non disponible"),
				"type":                "continent",
				"owner":               "unknown",
				"riskLevel":           riskLabel(continent.TensionLevel),
				"tensionLevel":        clamp(continent.TensionLevel, 0, 100),
				"stability":           clamp(100-continent.TensionLevel, 0, 100),
				"wealth":              50,
				"population":          continent.CurrentPlayers,
				"status":              status,
				"activeConflicts":     0,
				"activeWeatherEvents": 0,
				"tradeRoutes":         []gin.H{},
				"diplomaticRelations": []gin.H{},
			})
		}
		writeWorldResponse(c, gin.H{"regions": items}, nil)
	})

	private.GET("/world/reports", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		conflicts, _ := world.ListWorldConflicts(c.Request.Context(), save.WorldID, save.ContinentID, 20)
		events, _ := world.ListWorldEvents(c.Request.Context(), save.WorldID, save.ContinentID, 20)
		weather, _ := world.ListActiveWeather(c.Request.Context(), save.WorldID, save.ContinentID)
		writeWorldResponse(c, gin.H{
			"reports": []gin.H{
				{"id": "world-report", "type": "world_report", "title": "Rapport monde", "summary": "Synthèse globale", "details": gin.H{"events": len(events), "conflicts": len(conflicts), "weather": len(weather)}, "createdAt": time.Now().UTC(), "importance": "info", "recommendedActions": []string{}, "metrics": gin.H{"events": len(events), "conflicts": len(conflicts), "weather": len(weather)}},
			},
		}, nil)
	})

	private.GET("/world/conflicts/reports", func(c *gin.Context) {
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
		writeWorldResponse(c, gin.H{"reports": formatConflictItems(database, c.Request.Context(), conflicts)}, nil)
	})

	private.GET("/world/events/:id", getOwnedWorldEntity[models.GameEvent](database, world, "game event", "world_id = ? AND (continent_id IS NULL OR continent_id = ?)"))
	private.GET("/world/conflicts/:id", getOwnedWorldEntity[models.Conflict](database, world, "conflict", "world_id = ? AND (continent_id IS NULL OR continent_id = ?)"))
}

// synthesizePlayerAgreementsFromLogs turns recent commerce_agreement_create logs into visible agreements for the commerce tab.
func synthesizePlayerAgreementsFromLogs(database *gorm.DB, c *gin.Context) []gin.H {
	playerID := currentUserID(c)
	if playerID == 0 {
		return []gin.H{}
	}

	var logs []models.PlayerActionLog
	since := time.Now().UTC().Add(-72 * time.Hour)
	database.WithContext(c.Request.Context()).
		Where("player_id = ? AND (action = ? OR action LIKE ?) AND created_at >= ? AND status = ?",
			playerID, "commerce_agreement_create", "commerce_%", since, "accepted").
		Order("created_at DESC").Limit(15).Find(&logs)

	out := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		meta := parseJSONObject(log.MetadataJSON)
		partner := firstNonEmptyPhase2(toString(meta["target"]), toString(meta["faction"]), log.TargetID)
		mode := toString(meta["mode"])
		if mode == "" {
			mode = "create"
		}
		out = append(out, gin.H{
			"id":        "agr_" + strconv.FormatUint(uint64(log.Id), 10),
			"partner":   partner,
			"mode":      mode,
			"status":    "negocie",
			"risk":      toInt64OrDefault(meta["risk"], 40),
			"durationH": toInt64OrDefault(meta["duration"], 24),
			"source":    "player",
			"createdAt": log.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func firstNonEmptyPhase2(values ...string) string {
	for _, v := range values {
		trim := strings.TrimSpace(v)
		if trim != "" && trim != "0" && trim != "manual" {
			return trim
		}
	}
	return "Partenaire inconnu"
}

func toInt64OrDefault(value any, def int64) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	default:
		return def
	}
}
