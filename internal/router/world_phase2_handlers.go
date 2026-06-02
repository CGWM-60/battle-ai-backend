package router

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
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
		writeWorldResponse(c, gin.H{
			"relations": relations,
			"summary":   "Lecture des relations IA par continent.",
			"impact":    "Les relations influencent les traités, les émissaires et les risques de conflit.",
			"rewards":   worldOperationRewards("diplomacy"),
			"recommendations": []string{
				"Envoyer un émissaire vers une relation tendue.",
				"Ouvrir une négociation si le score descend sous 50.",
			},
		}, nil)
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
		writeWorldResponse(c, gin.H{
			"treaties": treaties,
			"summary":  "Traités actifs et opportunités de stabilisation diplomatique.",
			"impact":   "Un traité accepté baisse la tension et améliore le score diplomatique.",
			"rewards":  worldOperationRewards("diplomacy"),
		}, nil)
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
			meta := parseJSONObject(log.MetadataJSON)
			duration := 4 * time.Hour
			if log.Action == "diplomacy_negotiation_action" {
				duration = 45 * time.Minute
			}
			operation := worldOperationFromLog(log, "Négociation diplomatique", "diplomacy", duration)
			objective := firstNonEmptyPhase2(toString(meta["subject"]), toString(meta["objective"]), "Améliorer la relation")
			items = append(items, gin.H{
				"id":               operation["id"],
				"actionId":         operation["actionId"],
				"targetId":         log.TargetID,
				"targetName":       "Négociation",
				"targetType":       "continent",
				"subject":          "Diplomatie",
				"objective":        objective,
				"status":           operation["status"],
				"progress":         operation["progress"],
				"durationSeconds":  operation["durationSeconds"],
				"remainingSeconds": operation["remainingSeconds"],
				"startsAt":         operation["startsAt"],
				"finishesAt":       operation["finishesAt"],
				"canCancel":        operation["canCancel"],
				"canClaim":         operation["canClaim"],
				"cancelEndpoint":   operation["cancelEndpoint"],
				"claimEndpoint":    operation["claimEndpoint"],
				"cost":             gin.H{},
				"successChance":    0.6,
				"risks":            []string{},
				"choices":          []string{"send_emissary", "offer_resources", "propose_treaty"},
				"finalResult":      nil,
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
		targetID := strings.TrimSpace(firstNonEmptyPhase2(toString(payload["targetId"]), toString(payload["target"])))
		if targetID == "" {
			writeWorldResponse(c, nil, badRequestError("TARGET_REQUIRED", "La cible de négociation est obligatoire.", nil))
			return
		}
		if cooldownActive(database, c, save.PlayerID, "diplomacy_negotiation_open", targetID, 5*time.Minute) {
			writeWorldResponse(c, nil, conflictError("COOLDOWN_ACTIVE", "Négociation en cooldown, réessayez plus tard.", map[string]any{"cooldownSeconds": 300}))
			return
		}
		actionID := stableActionID(payload)
		now := time.Now().UTC()
		duration := 4 * time.Hour
		payload["startedAt"] = now.Format(time.RFC3339)
		payload["finishesAt"] = now.Add(duration).Format(time.RFC3339)
		payload["durationSeconds"] = int(duration.Seconds())
		err = world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "diplomacy_negotiation_open", "diplomacy", actionID, "accepted", "", payload)
		response := worldOperationResponse(actionID, "Négociation diplomatique", "diplomacy", "in_progress", now, duration)
		response["started"] = true
		writeWorldResponse(c, response, err)
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
		now := time.Now().UTC()
		duration := 45 * time.Minute
		payload["startedAt"] = now.Format(time.RFC3339)
		payload["finishesAt"] = now.Add(duration).Format(time.RFC3339)
		payload["durationSeconds"] = int(duration.Seconds())
		err = world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "diplomacy_negotiation_action", "diplomacy", c.Param("id"), "accepted", "", payload)
		response := worldOperationResponse(c.Param("id"), "Action de négociation", "diplomacy", "in_progress", now, duration)
		response["applied"] = true
		writeWorldResponse(c, response, err)
	})

	private.GET("/diplomacy/emissaries", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		available, total := emissaryAvailability(database, c.Request.Context(), save)
		var missionLogs []models.PlayerActionLog
		_ = database.WithContext(c.Request.Context()).
			Where("player_id = ? AND world_id = ? AND action = ? AND status = ? AND created_at >= ?", save.PlayerID, save.WorldID, "diplomacy_emissary_send", "accepted", time.Now().UTC().Add(-2*time.Hour)).
			Order("created_at DESC").
			Limit(total).
			Find(&missionLogs).Error
		items := make([]gin.H, 0, total)
		for i := 0; i < total; i++ {
			status := "available"
			var mission any
			if i < len(missionLogs) {
				status = "on_mission"
				mission = worldOperationFromLog(missionLogs[i], "Émissaire en mission", "diplomacy", 2*time.Hour)
			} else if i >= available {
				status = "on_mission"
			}
			items = append(items, gin.H{
				"id":               i + 1,
				"name":             "Émissaire " + strconv.Itoa(i+1),
				"level":            1,
				"experience":       0,
				"specialty":        "paix",
				"status":           status,
				"currentMission":   mission,
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
		now := time.Now().UTC()
		duration := 2 * time.Hour
		payload["startedAt"] = now.Format(time.RFC3339)
		payload["finishesAt"] = now.Add(duration).Format(time.RFC3339)
		payload["durationSeconds"] = int(duration.Seconds())
		err = world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "diplomacy_emissary_send", "emissary", actionID, "accepted", "", payload)
		response := worldOperationResponse(actionID, "Émissaire en mission", "diplomacy", "on_mission", now, duration)
		response["sent"] = true
		writeWorldResponse(c, response, err)
	})

	private.GET("/diplomacy/reports", func(c *gin.Context) {
		save, saveErr := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if saveErr != nil {
			writeWorldResponse(c, nil, saveErr)
			return
		}
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
		operations := recentWorldOperations(database, c, save, map[string]worldOperationSpec{
			"diplomacy_negotiation_open":   {Title: "Négociation diplomatique", Domain: "diplomacy", Duration: 4 * time.Hour},
			"diplomacy_negotiation_action": {Title: "Action de négociation", Domain: "diplomacy", Duration: 45 * time.Minute},
			"diplomacy_emissary_send":      {Title: "Émissaire en mission", Domain: "diplomacy", Duration: 2 * time.Hour},
			"diplomacy_treaty_accept":      {Title: "Traité accepté", Domain: "diplomacy", Duration: 15 * time.Minute},
			"diplomacy_treaty_reject":      {Title: "Traité refusé", Domain: "diplomacy", Duration: 15 * time.Minute},
			"diplomacy_treaty_break":       {Title: "Traité rompu", Domain: "diplomacy", Duration: 15 * time.Minute},
		}, 12)
		writeWorldResponse(c, gin.H{"reports": reports, "operations": operations}, nil)
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
				"id":     "r_" + strconv.FormatUint(uint64(ct.Id), 10),
				"rawId":  ct.Id,
				"name":   defaultText(ct.Name, "Région"),
				"type":   "region",
				"score":  score,
				"status": diplomacyStance(score),
			})
		}

		// Guilds from the player's current continent only.
		guilds, _ := world.ListGuilds(ctx, currentUserID(c), save.ContinentID, 50)
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
		now := time.Now().UTC()
		duration := 15 * time.Minute
		err = world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "diplomacy_treaty_"+action, "treaty", c.Param("id"), "accepted", "", gin.H{
			"startedAt":       now.Format(time.RFC3339),
			"finishesAt":      now.Add(duration).Format(time.RFC3339),
			"durationSeconds": int(duration.Seconds()),
		})
		response := worldOperationResponse(c.Param("id"), "Décision de traité", "diplomacy", "in_progress", now, duration)
		response["status"] = action
		response["treatyId"] = c.Param("id")
		writeWorldResponse(c, response, err)
	}
}

func registerTradeRoutes(private *gin.RouterGroup, database *gorm.DB, world *service.WorldGameService) {
	private.GET("/trade/overview", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
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
			"operations": recentWorldOperations(database, c, save, map[string]worldOperationSpec{
				"commerce_create_route":    {Title: "Création route commerciale", Domain: "commerce", Duration: 45 * time.Minute},
				"commerce_create_export":   {Title: "Création export commercial", Domain: "commerce", Duration: 45 * time.Minute},
				"commerce_create_import":   {Title: "Création import commercial", Domain: "commerce", Duration: 45 * time.Minute},
				"commerce_routes_optimize": {Title: "Optimisation des routes", Domain: "commerce", Duration: 45 * time.Minute},
			}, 12),
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
		writeWorldResponse(c, gin.H{
			"routes":  items,
			"summary": "Routes commerciales connues par le moteur IA.",
			"impact":  "Les routes modifient les crédits 24h, les importations et la dépendance extérieure.",
			"rewards": worldOperationRewards("commerce"),
		}, nil)
	})

	private.POST("/trade/routes/create", func(c *gin.Context) {
		payload := bindOptionalMap(c)
		actionType := "commerce_create_route"
		if t := toString(payload["type"]); t == "export" {
			actionType = "commerce_create_export"
		} else if t == "import" {
			actionType = "commerce_create_import"
		}

		save, _ := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		actionID := stableActionID(payload)
		if actionID == "" || actionID == "manual" {
			actionID = "route_" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
		}
		now := time.Now().UTC()
		duration := 45 * time.Minute
		payload["startedAt"] = now.Format(time.RFC3339)
		payload["finishesAt"] = now.Add(duration).Format(time.RFC3339)
		payload["durationSeconds"] = int(duration.Seconds())
		if save != nil {
			_ = world.LogPlayerWorldAction(
				c.Request.Context(),
				currentUserID(c),
				save.WorldID,
				save.ContinentID,
				actionType,
				"trade_route",
				actionID,
				"accepted",
				"",
				payload,
			)
		}

		response := worldOperationResponse(actionID, "Création de route commerciale", "commerce", "negotiating", now, duration)
		response["created"] = true
		response["type"] = payload["type"]
		writeWorldResponse(c, response, nil)
	})
	private.POST("/trade/routes/optimize", func(c *gin.Context) {
		payload := bindOptionalMap(c)
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		actionID := "optimize_" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
		now := time.Now().UTC()
		duration := 45 * time.Minute
		payload["startedAt"] = now.Format(time.RFC3339)
		payload["finishesAt"] = now.Add(duration).Format(time.RFC3339)
		payload["durationSeconds"] = int(duration.Seconds())
		err = world.LogPlayerWorldAction(c.Request.Context(), save.PlayerID, save.WorldID, save.ContinentID, "commerce_routes_optimize", "trade_route", actionID, "accepted", "", payload)
		response := worldOperationResponse(actionID, "Optimisation des routes", "commerce", "in_progress", now, duration)
		response["optimized"] = err == nil
		writeWorldResponse(c, response, err)
	})
	private.POST("/trade/routes/:id/optimize", func(c *gin.Context) {
		if err := validateRouteOwnership(database, world, c, c.Param("id")); err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		payload := bindOptionalMap(c)
		save, _ := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		now := time.Now().UTC()
		duration := 45 * time.Minute
		payload["startedAt"] = now.Format(time.RFC3339)
		payload["finishesAt"] = now.Add(duration).Format(time.RFC3339)
		payload["durationSeconds"] = int(duration.Seconds())
		if save != nil {
			_ = world.LogPlayerWorldAction(c.Request.Context(), save.PlayerID, save.WorldID, save.ContinentID, "commerce_routes_optimize", "trade_route", c.Param("id"), "accepted", "", payload)
		}
		response := worldOperationResponse(c.Param("id"), "Optimisation de route", "commerce", "in_progress", now, duration)
		response["optimized"] = true
		writeWorldResponse(c, response, nil)
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
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		// Synthesize visible agreements from recent player commerce actions (newAgreement etc.)
		agreements := synthesizePlayerAgreementsFromLogs(database, c)
		operations := recentWorldOperations(database, c, save, map[string]worldOperationSpec{
			"commerce_agreement_create":      {Title: "Nouvel accord commercial", Domain: "commerce", Duration: 45 * time.Minute},
			"commerce_agreement_negotiating": {Title: "Renégociation commerciale", Domain: "commerce", Duration: 45 * time.Minute},
			"commerce_agreement_accepted":    {Title: "Accord commercial accepté", Domain: "commerce", Duration: 15 * time.Minute},
			"commerce_agreement_rejected":    {Title: "Accord commercial refusé", Domain: "commerce", Duration: 15 * time.Minute},
		}, 12)
		writeWorldResponse(c, gin.H{"agreements": agreements, "operations": operations}, nil)
	})
	private.POST("/trade/agreements/create", tradeAgreementCreateHandler(database, world))
	private.POST("/trade/agreements/new", tradeAgreementCreateHandler(database, world))
	private.POST("/trade/agreements/:id/accept", tradeAgreementDecisionHandler(database, world, "accepted"))
	private.POST("/trade/agreements/:id/reject", tradeAgreementDecisionHandler(database, world, "rejected"))
	private.POST("/trade/agreements/:id/negotiate", tradeAgreementDecisionHandler(database, world, "negotiating"))
	private.POST("/trade/agreement/:id/accept", tradeAgreementDecisionHandler(database, world, "accepted"))
	private.POST("/trade/agreement/:id/reject", tradeAgreementDecisionHandler(database, world, "rejected"))
	private.POST("/trade/agreement/:id/negotiate", tradeAgreementDecisionHandler(database, world, "negotiating"))
	private.POST("/trade/agreements/accept", tradeAgreementDecisionHandler(database, world, "accepted"))
	private.POST("/trade/agreements/reject", tradeAgreementDecisionHandler(database, world, "rejected"))
	private.POST("/trade/agreements/negotiate", tradeAgreementDecisionHandler(database, world, "negotiating"))

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
		forecast := []gin.H{}
		if len(weather) > 0 {
			for i := 0; i < 24; i += 4 {
				sev := clamp(avg+(i-8)*2, 0, 100)
				eventDesc := "projection stable"
				if sev > 60 {
					eventDesc = "projection de risque élevé"
				}
				forecast = append(forecast, gin.H{
					"hour":     i,
					"severity": sev,
					"risk":     riskLabel(sev),
					"event":    eventDesc,
					"source":   "projection_from_active_weather",
				})
			}
		}

		writeWorldResponse(c, gin.H{
			"activeEvents":    len(weather),
			"averageSeverity": avg,
			"globalRisk":      riskLabel(avg),
			"forecast24h":     forecast,
		}, nil)
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
		writeWorldResponse(c, gin.H{
			"risks":   risks,
			"summary": "Risques météo évalués sur les événements actifs.",
			"impact":  "Un risque élevé peut réduire population, nourriture, énergie et bâtiments.",
			"rewards": worldOperationRewards("weather"),
			"recommendations": []string{
				"Démarrer un plan météo si la sévérité dépasse 60.",
				"Prépositionner des ressources avant les risques longs.",
			},
		}, nil)
	})

	private.GET("/weather/plans", weatherPlansHandler(database, world))
	private.POST("/weather/plans/start", weatherPlanStartHandler(database, world))
	private.POST("/weather/plan/start", weatherPlanStartHandler(database, world))

	private.POST("/weather/plans/:id/start", weatherPlanStartHandler(database, world))
	private.POST("/weather/plan/:id/start", weatherPlanStartHandler(database, world))

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
}

func weatherPlanStartHandler(database *gorm.DB, world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload := bindOptionalMap(c)
		raw := firstNonEmptyPhase2(strings.TrimSpace(c.Param("id")), toString(payload["planId"]), toString(payload["id"]), toString(payload["actionKey"]))
		actionKey := normalizeWeatherActionKey(raw)
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
		if cooldownActive(database, c, save.PlayerID, "weather_plan_start", actionKey, 30*time.Second) {
			writeWorldResponse(c, nil, conflictError("COOLDOWN_ACTIVE", "Plan météo en cooldown (30s).", map[string]any{"cooldownSeconds": 30}))
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
		now := time.Now().UTC()
		durationMinutes, _ := plan["durationMinutes"].(int)
		duration := time.Duration(durationMinutes) * time.Minute
		if duration <= 0 {
			duration = 90 * time.Minute
		}
		payload["startedAt"] = now.Format(time.RFC3339)
		payload["finishesAt"] = now.Add(duration).Format(time.RFC3339)
		payload["durationSeconds"] = int(duration.Seconds())
		err = database.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.PlayerSave{}).Where("id = ?", save.Id).Updates(map[string]any{
				"credits":        save.Credits - creditsCost,
				"food":           save.Food - foodCost,
				"energy":         save.Energy - energyCost,
				"version":        save.Version + 1,
				"last_synced_at": now,
			}).Error; err != nil {
				return err
			}
			return world.LogPlayerWorldAction(c.Request.Context(), currentUserID(c), save.WorldID, save.ContinentID, "weather_plan_start", "weather_plan", actionKey, "accepted", "", payload)
		})
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		response := worldOperationResponse(actionKey, toString(plan["title"]), "weather", "in_progress", now, duration)
		response["plan"] = plan
		response["accepted"] = true
		writeWorldResponse(c, response, nil)
	}
}

func weatherPlansHandler(database *gorm.DB, world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		plans := []gin.H{
			weatherActionPlan("deploy-aid"),
			weatherActionPlan("preposition-resources"),
			weatherActionPlan("activate-defense-protocol"),
		}
		now := time.Now().UTC()
		for _, plan := range plans {
			actionKey := toString(plan["id"])
			durationMinutes, _ := plan["durationMinutes"].(int)
			var logEntry models.PlayerActionLog
			result := database.WithContext(c.Request.Context()).
				Where("player_id = ? AND action = ? AND target_id = ? AND status = ?", save.PlayerID, "weather_plan_start", actionKey, "accepted").
				Order("created_at DESC").
				Limit(1).
				Find(&logEntry)
			if result.Error != nil || result.RowsAffected == 0 || durationMinutes <= 0 {
				plan["status"] = "available"
				continue
			}
			finishesAt := logEntry.CreatedAt.UTC().Add(time.Duration(durationMinutes) * time.Minute)
			operation := worldOperationFromLog(logEntry, toString(plan["title"]), "weather", time.Duration(durationMinutes)*time.Minute)
			for key, value := range operation {
				plan[key] = value
			}
			if finishesAt.After(now) {
				plan["status"] = "in_progress"
				plan["startsAt"] = logEntry.CreatedAt.UTC()
				plan["finishesAt"] = finishesAt
				plan["remainingSeconds"] = int(finishesAt.Sub(now).Seconds())
			} else {
				plan["status"] = "completed"
				plan["startsAt"] = logEntry.CreatedAt.UTC()
				plan["finishesAt"] = finishesAt
				plan["canCancel"] = false
				plan["canClaim"] = true
			}
		}
		writeWorldResponse(c, gin.H{"plans": plans}, nil)
	}
}

func normalizeWeatherActionKey(raw string) string {
	switch strings.TrimSpace(raw) {
	case "preposition", "preposition-resources":
		return "preposition-resources"
	case "defense-protocol", "activate-defense-protocol":
		return "activate-defense-protocol"
	case "deploy-aid", "deployAid":
		return "deploy-aid"
	default:
		return strings.TrimSpace(raw)
	}
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

type worldOperationSpec struct {
	Title    string
	Domain   string
	Duration time.Duration
}

func worldOperationResponse(actionID string, title string, domain string, status string, startedAt time.Time, duration time.Duration) gin.H {
	now := time.Now().UTC()
	if startedAt.IsZero() {
		startedAt = now
	}
	if duration <= 0 {
		duration = 30 * time.Minute
	}
	endsAt := startedAt.UTC().Add(duration)
	remaining := int(endsAt.Sub(now).Seconds())
	if remaining < 0 {
		remaining = 0
	}
	progress := 1.0
	if duration > 0 && now.Before(endsAt) {
		elapsed := now.Sub(startedAt.UTC()).Seconds()
		if elapsed < 0 {
			elapsed = 0
		}
		progress = elapsed / duration.Seconds()
		if progress < 0 {
			progress = 0
		}
		if progress > 1 {
			progress = 1
		}
	}
	effectiveStatus := status
	if effectiveStatus == "" {
		effectiveStatus = "in_progress"
	}
	if remaining == 0 && (effectiveStatus == "in_progress" || effectiveStatus == "on_mission" || effectiveStatus == "negotiating" || effectiveStatus == "negocie") {
		effectiveStatus = "completed"
	}
	actionID = strings.TrimSpace(actionID)
	if actionID == "" {
		actionID = "manual"
	}
	return gin.H{
		"id":               actionID,
		"actionId":         actionID,
		"title":            title,
		"domain":           domain,
		"status":           effectiveStatus,
		"startedAt":        startedAt.UTC().Format(time.RFC3339),
		"startsAt":         startedAt.UTC().Format(time.RFC3339),
		"endsAt":           endsAt.Format(time.RFC3339),
		"finishesAt":       endsAt.Format(time.RFC3339),
		"durationSeconds":  int(duration.Seconds()),
		"remainingSeconds": remaining,
		"progress":         progress,
		"canCancel":        remaining > 0,
		"canClaim":         remaining == 0,
		"cancelEndpoint":   "/api/v1/world/actions/" + actionID + "/cancel",
		"claimEndpoint":    "/api/v1/world/actions/" + actionID + "/claim",
		"serverNow":        now.Format(time.RFC3339),
		"rewards":          worldOperationRewards(domain),
		"impact":           worldOperationImpact(domain),
	}
}

func worldOperationRewards(domain string) gin.H {
	switch strings.TrimSpace(strings.ToLower(domain)) {
	case "conflicts":
		return gin.H{"xp": 90, "credits": 350, "stability": 4, "description": "Gain tactique si l'action est menée à terme."}
	case "diplomacy":
		return gin.H{"xp": 60, "reputation": 5, "diplomacyScore": 3, "description": "Améliore l'influence diplomatique et débloque de meilleurs accords."}
	case "commerce":
		return gin.H{"xp": 45, "credits": 500, "tradeEfficiency": 3, "description": "Améliore les flux commerciaux et les gains sur 24h."}
	case "weather":
		return gin.H{"xp": 55, "riskReduction": 8, "populationProtection": 4, "description": "Réduit les dégâts météo et protège population/production."}
	default:
		return gin.H{"xp": 25, "description": "Récompense monde calculée à la finalisation."}
	}
}

func worldOperationImpact(domain string) string {
	switch strings.TrimSpace(strings.ToLower(domain)) {
	case "conflicts":
		return "Réduction de tension et meilleure stabilité militaire si l'opération réussit."
	case "diplomacy":
		return "Influence les relations IA, les traités et les futures négociations."
	case "commerce":
		return "Influence routes, volume d'échange, balance commerciale et revenus."
	case "weather":
		return "Influence risques météo, pertes civiles, dégâts bâtiments et continuité production."
	default:
		return "Influence le score mondial et le journal de progression."
	}
}

func worldOperationFromLog(log models.PlayerActionLog, title string, domain string, duration time.Duration) gin.H {
	actionID := strings.TrimSpace(log.TargetID)
	if actionID == "" {
		actionID = strconv.FormatUint(uint64(log.Id), 10)
	}
	return worldOperationResponse(actionID, title, domain, "in_progress", log.CreatedAt.UTC(), duration)
}

func recentWorldOperations(database *gorm.DB, c *gin.Context, save *models.PlayerSave, specs map[string]worldOperationSpec, limit int) []gin.H {
	if database == nil || save == nil || len(specs) == 0 {
		return []gin.H{}
	}
	actions := make([]string, 0, len(specs))
	for action := range specs {
		actions = append(actions, action)
	}
	var logs []models.PlayerActionLog
	since := time.Now().UTC().Add(-7 * 24 * time.Hour)
	if limit <= 0 {
		limit = 20
	}
	_ = database.WithContext(c.Request.Context()).
		Where("player_id = ? AND world_id = ? AND action IN ? AND status = ? AND created_at >= ?", save.PlayerID, save.WorldID, actions, "accepted", since).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	out := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		spec, ok := specs[log.Action]
		if !ok {
			continue
		}
		out = append(out, worldOperationFromLog(log, spec.Title, spec.Domain, spec.Duration))
	}
	return out
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
		writeWorldResponse(c, gin.H{
			"reports": formatConflictItems(database, c.Request.Context(), conflicts),
			"operations": recentWorldOperations(database, c, save, map[string]worldOperationSpec{
				"conflict_action": {Title: "Intervention tactique", Domain: "conflicts", Duration: 4 * time.Hour},
			}, 12),
		}, nil)
	})

	private.GET("/world/events/:id", getOwnedWorldEntity[models.GameEvent](database, world, "game event", "world_id = ? AND (continent_id IS NULL OR continent_id = ?)"))
	private.GET("/world/conflicts/:id", getOwnedWorldEntity[models.Conflict](database, world, "conflict", "world_id = ? AND (continent_id IS NULL OR continent_id = ?)"))
}

func tradeAgreementDecisionHandler(database *gorm.DB, world *service.WorldGameService, status string) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload := bindOptionalMap(c)
		agreementID := strings.TrimSpace(c.Param("id"))
		if agreementID == "" {
			agreementID = firstNonEmptyPhase2(toString(payload["agreementId"]), toString(payload["id"]))
		}
		if agreementID == "" || agreementID == "Partenaire inconnu" {
			writeWorldResponse(c, nil, badRequestError("INVALID_AGREEMENT_ID", "Identifiant d'accord invalide.", nil))
			return
		}

		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		sourceLogID := uint(0)
		rawLogID := strings.TrimPrefix(agreementID, "agr_")
		if parsed, parseErr := strconv.ParseUint(rawLogID, 10, 64); parseErr == nil {
			sourceLogID = uint(parsed)
		}
		if sourceLogID != 0 {
			var logEntry models.PlayerActionLog
			result := database.WithContext(c.Request.Context()).
				Where("id = ? AND player_id = ? AND action = ?", sourceLogID, save.PlayerID, "commerce_agreement_create").
				Limit(1).
				Find(&logEntry)
			if result.Error == nil && result.RowsAffected > 0 {
				meta := parseJSONObject(logEntry.MetadataJSON)
				meta["agreementStatus"] = status
				meta["decisionAt"] = time.Now().UTC().Format(time.RFC3339)
				if toString(payload["mode"]) != "" {
					meta["decisionMode"] = toString(payload["mode"])
				}
				if encoded, marshalErr := json.Marshal(meta); marshalErr == nil {
					_ = database.WithContext(c.Request.Context()).
						Model(&models.PlayerActionLog{}).
						Where("id = ?", logEntry.Id).
						Update("metadata_json", datatypes.JSON(encoded)).Error
				}
			}
		}
		now := time.Now().UTC()
		duration := 15 * time.Minute
		if status == "negotiating" {
			duration = 45 * time.Minute
		}
		payload["startedAt"] = now.Format(time.RFC3339)
		payload["finishesAt"] = now.Add(duration).Format(time.RFC3339)
		payload["durationSeconds"] = int(duration.Seconds())
		logErr := world.LogPlayerWorldAction(
			c.Request.Context(),
			save.PlayerID,
			save.WorldID,
			save.ContinentID,
			"commerce_agreement_"+status,
			"commerce_agreement",
			agreementID,
			"accepted",
			"",
			gin.H{"agreementId": agreementID, "agreementStatus": status, "payload": payload},
		)
		response := worldOperationResponse(agreementID, "Décision accord commercial", "commerce", "in_progress", now, duration)
		response["status"] = status
		response["updated"] = logErr == nil
		writeWorldResponse(c, response, logErr)
	}
}

func tradeAgreementCreateHandler(database *gorm.DB, world *service.WorldGameService) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload := bindOptionalMap(c)
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		payload["agreementStatus"] = "negocie"
		actionID := stableActionID(payload)
		if actionID == "" || actionID == "manual" {
			actionID = "agreement_" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
		}
		now := time.Now().UTC()
		duration := 45 * time.Minute
		payload["startedAt"] = now.Format(time.RFC3339)
		payload["finishesAt"] = now.Add(duration).Format(time.RFC3339)
		payload["durationSeconds"] = int(duration.Seconds())
		err = world.LogPlayerWorldAction(c.Request.Context(), save.PlayerID, save.WorldID, save.ContinentID, "commerce_agreement_create", "commerce_agreement", actionID, "accepted", "", payload)
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		var logEntry models.PlayerActionLog
		result := database.WithContext(c.Request.Context()).
			Where("player_id = ? AND action = ? AND target_id = ?", save.PlayerID, "commerce_agreement_create", actionID).
			Order("created_at DESC").
			Limit(1).
			Find(&logEntry)
		agreementID := actionID
		if result.Error == nil && result.RowsAffected > 0 && logEntry.Id != 0 {
			agreementID = "agr_" + strconv.FormatUint(uint64(logEntry.Id), 10)
		}
		response := worldOperationResponse(agreementID, "Nouvel accord commercial", "commerce", "negocie", now, duration)
		response["created"] = true
		response["sourceActionId"] = actionID
		response["agreement"] = gin.H{"id": agreementID, "status": "negocie", "source": "player"}
		writeWorldResponse(c, response, nil)
	}
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
		Where("player_id = ? AND action = ? AND created_at >= ?",
			playerID, "commerce_agreement_create", since).
		Order("created_at DESC").Limit(15).Find(&logs)

	out := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		meta := parseJSONObject(log.MetadataJSON)
		partner := firstNonEmptyPhase2(toString(meta["target"]), toString(meta["faction"]), log.TargetID)
		mode := toString(meta["mode"])
		if mode == "" {
			mode = "create"
		}
		risk := toInt64OrDefault(meta["risk"], 40)
		title := toString(meta["title"])
		if title == "" {
			title = "Accord " + partner
		}
		status := firstNonEmptyPhase2(toString(meta["agreementStatus"]), "negocie")
		aiDecision := "En cours d'analyse"
		if risk <= 25 {
			aiDecision = "Accord favorable"
		} else if risk >= 60 {
			aiDecision = "Risqué - négociation conseillée"
		}
		out = append(out, gin.H{
			"id":            "agr_" + strconv.FormatUint(uint64(log.Id), 10),
			"title":         title,
			"partner":       partner,
			"mode":          mode,
			"status":        status,
			"risk":          risk,
			"durationH":     toInt64OrDefault(meta["durationHours"], toInt64OrDefault(meta["duration"], 24)),
			"source":        "player",
			"createdAt":     log.CreatedAt.UTC().Format(time.RFC3339),
			"cost":          toInt64OrDefault(meta["cost"], 0),
			"estimatedGain": toInt64OrDefault(meta["estimatedGain"], 0),
			"aiDecision":    aiDecision,
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
