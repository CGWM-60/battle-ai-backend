package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ProfileHandler manages the ProfileGamer (may be empty for new players).
// All saves go through here. Server validates + applies. Flutter displays result.
type ProfileHandler struct {
	db    *gorm.DB
	redis *cache.RedisService
}

func NewProfileHandler(db *gorm.DB, redis *cache.RedisService) *ProfileHandler {
	return &ProfileHandler{db: db, redis: redis}
}

// GetProfile returns the profile for a given user_id (from main app user).
// Query param: ?user_id=123
// If no row exists, returns exists:false + a zeroed profile with the user_id.
func (h *ProfileHandler) GetProfile(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusOK, gin.H{"exists": false, "profile": nil})
		return
	}

	uidStr := c.Query("user_id")
	if uidStr == "" {
		// also accept json body for flexibility
		var body struct {
			UserID uint `json:"user_id"`
		}
		_ = c.ShouldBindJSON(&body)
		if body.UserID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required (query or body)"})
			return
		}
		uidStr = strconv.FormatUint(uint64(body.UserID), 10)
	}

	uid, err := strconv.ParseUint(uidStr, 10, 64)
	if err != nil || uid == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var p models.ProfileGamer
	if err := h.db.Where("user_id = ?", uint(uid)).First(&p).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load profile"})
			return
		}
		// not found -> empty profile (allowed). This First() is expected for new users and was producing the "record not found" log.
		c.JSON(http.StatusOK, gin.H{
			"exists":             false,
			"starter_allocation": services.Level1StarterAllocation(),
			"profile": models.ProfileGamer{
				UserID:             uint(uid),
				Level:              1,
				Power:              0,
				Population:         0,
				PopulationCapacity: 0,
				Morale:             50,
				EnergyProduction:   0,
				EnergyConsumption:  0,
				EnergyBalance:      0,
				EnergyStored:       0,
				Security:           50,
			},
			"daily_plan_context": buildDailyPlanContextWithResources(models.ProfileGamer{
				UserID:             uint(uid),
				Level:              1,
				Power:              0,
				Population:         0,
				PopulationCapacity: 0,
				Morale:             50,
				EnergyProduction:   0,
				EnergyConsumption:  0,
				EnergyBalance:      0,
				EnergyStored:       0,
				Security:           50,
			}, services.Level1StarterAllocation(), nil),
		})
		return
	}
	if err := services.NewResourceService(h.db).EnsureInitialAllocation(c.Request.Context(), p.ID); err == nil {
		_ = h.db.First(&p, p.ID).Error
	}

	resourceBalances, cityStatsPayload := h.profileResourcePayload(c, p.ID)

	// Enrich for MmoEntryScreen (same as SaveProfile)
	avatarURL := ""
	if p.AvatarID > 0 {
		var av models.Avatar
		if err := h.db.Select("url").First(&av, p.AvatarID).Error; err == nil {
			avatarURL = av.URL
		}
	}
	worldName := ""
	if p.WorldID > 0 {
		var w models.World
		if err := h.db.Select("name").First(&w, p.WorldID).Error; err == nil {
			worldName = w.Name
		}
	}
	factionName := ""
	if p.FactionID > 0 {
		var f models.Faction
		if err := h.db.Select("name").First(&f, p.FactionID).Error; err == nil {
			factionName = f.Name
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"exists":             true,
		"profile":            p,
		"avatar_url":         avatarURL,
		"world_name":         worldName,
		"faction_name":       factionName,
		"starter_allocation": services.Level1StarterAllocation(),
		"resources":          resourceBalances,
		"city_stats":         cityStatsPayload,
		// Daily plan context sent on first load for player's AI (provider/local/governor > server fallback).
		"daily_plan_context": buildDailyPlanContextWithResources(p, resourceBalances, cityStatsPayload),
	})
}

// SaveProfile creates or updates the ProfileGamer for the user.
// Body (JSON or form):
// { "user_id": 123, "avatar_id": 5, "faction_id": 2, "ia_companion_id": 7, "pseudo": "Neo", "city_name": "Neon Spire" }
func (h *ProfileHandler) SaveProfile(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db not available"})
		return
	}

	var req struct {
		UserID             uint   `json:"user_id" form:"user_id"`
		AvatarID           uint   `json:"avatar_id" form:"avatar_id"`
		FactionID          uint   `json:"faction_id" form:"faction_id"`
		IACompanionID      uint   `json:"ia_companion_id" form:"ia_companion_id"`
		Pseudo             string `json:"pseudo" form:"pseudo"`
		CityName           string `json:"city_name" form:"city_name"`
		Population         int    `json:"population" form:"population"`
		PopulationCapacity int    `json:"population_capacity" form:"population_capacity"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if req.UserID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	var p models.ProfileGamer
	now := time.Now().UTC()

	tx := h.db.Where("user_id = ?", req.UserID).First(&p)
	if tx.Error != nil {
		// Expected for brand new users (first profile creation). Do not treat as hard error.
		// GORM logs "record not found" here by default — this is the source of the log the user saw.
		if !errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			// Real DB error -> fail
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check existing profile"})
			return
		}
		// create new

		p = models.ProfileGamer{
			UserID:        req.UserID,
			AvatarID:      req.AvatarID,
			FactionID:     req.FactionID,
			IACompanionID: req.IACompanionID,
			Pseudo:        req.Pseudo,
			CityName:      req.CityName,
			Level:         1,
			Power:         0,
			// Initial evolutionary city stats (Population, Morale, Energy, Security).
			// These will evolve via ticks, actions, events per the detailed rules.
			Population:         0,
			PopulationCapacity: 0,
			Morale:             50,
			EnergyProduction:   0,
			EnergyConsumption:  0,
			EnergyBalance:      0,
			EnergyStored:       0,
			Security:           50,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		// Allow client-driven resync of advanced pop (from local persist when server snapshot was older).
		// Clamp to sane values (cap >=0, pop 0..cap if cap>0).
		if req.PopulationCapacity > 0 {
			p.PopulationCapacity = req.PopulationCapacity
		}
		if req.Population > 0 {
			if p.PopulationCapacity > 0 && req.Population > p.PopulationCapacity {
				p.Population = p.PopulationCapacity
			} else {
				p.Population = req.Population
			}
		}
		if err := h.db.Create(&p).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create profile_gamer"})
			return
		}

		// Auto-assign continent based on faction (max 500 players per continent, proportional).
		// Best-effort: if assignment fails (e.g. "faction not assigned to continent" because Redis keys missing
		// for an old/admin-created faction), we still return success with the created profile.
		// This fixes the 400 on first save that sent the user back to the creation screen.
		// The profile row exists; continent can be repaired on next load or by admin.
		// (The GORM "record not found" for the initial First() on a brand new user is expected and should not be fatal.)
		if h.redis != nil {
			ws := services.NewWorldService(h.db, h.redis)
			wID, cID, aErr := ws.AssignPlayerToContinent(c.Request.Context(), req.UserID, req.FactionID)
			if aErr == nil {
				p.WorldID = wID
				p.ContinentID = cID
				h.db.Save(&p)
			}
			// Do not return 400. Player can enter the game; assignment is enriched when possible.
		}

		// Additional robust fallback: always ensure the profile gets ContinentID/WorldID from the faction's DB record
		// if it doesn't have one yet. This guarantees the profile appears in "Liste des Joueurs par Monde"
		// in the admin (which queries ProfileGamer by continent_id) and shows the assigned world.
		if p.ContinentID == 0 && req.FactionID > 0 {
			var f models.Faction
			if err := h.db.First(&f, req.FactionID).Error; err == nil && f.ContinentID != 0 {
				p.WorldID = f.WorldID
				p.ContinentID = f.ContinentID
				h.db.Save(&p)
				// Also populate Redis so Assign and counts work next time
				if h.redis != nil {
					_ = h.redis.SetString(c.Request.Context(), fmt.Sprintf("nexus:faction:%d:continent", req.FactionID), fmt.Sprintf("%d", f.ContinentID), 0)
					_ = h.redis.SetString(c.Request.Context(), fmt.Sprintf("nexus:faction:%d:world", req.FactionID), fmt.Sprintf("%d", f.WorldID), 0)
				}
			}
		}
		if err := services.NewResourceService(h.db).EnsureInitialAllocation(c.Request.Context(), p.ID); err == nil {
			_ = h.db.First(&p, p.ID).Error
		}
	} else {
		// update existing (only the gamer fields; never touch other tables here)
		p.AvatarID = req.AvatarID
		p.FactionID = req.FactionID
		p.IACompanionID = req.IACompanionID
		p.Pseudo = req.Pseudo
		p.CityName = req.CityName
		p.UpdatedAt = now
		// Support client resync push for population when local advanced state (offline growth per client prediction + persist) is ahead of server snapshot.
		// Server remains authoritative for future ticks, but accepts the higher recent value to avoid reset on relaunch.
		if req.PopulationCapacity > 0 {
			p.PopulationCapacity = req.PopulationCapacity
		}
		if req.Population > 0 {
			cap := p.PopulationCapacity
			if cap <= 0 {
				cap = req.PopulationCapacity
			}
			if cap > 0 && req.Population > cap {
				p.Population = cap
			} else {
				p.Population = req.Population
			}
		}
		if err := h.db.Save(&p).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile_gamer"})
			return
		}
	}

	// Enrich for MmoEntryScreen: avatar URL + world name (so we can show real chosen avatar, pseudo, lvl, power, world name)
	resourceBalances, cityStatsPayload := h.profileResourcePayload(c, p.ID)

	avatarURL := ""
	if p.AvatarID > 0 {
		var av models.Avatar
		if err := h.db.Select("url").First(&av, p.AvatarID).Error; err == nil {
			avatarURL = av.URL
		}
	}
	worldName := ""
	if p.WorldID > 0 {
		var w models.World
		if err := h.db.Select("name").First(&w, p.WorldID).Error; err == nil {
			worldName = w.Name
		}
	}
	factionName := ""
	if p.FactionID > 0 {
		var f models.Faction
		if err := h.db.Select("name").First(&f, p.FactionID).Error; err == nil {
			factionName = f.Name
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"profile":            p,
		"exists":             true,
		"avatar_url":         avatarURL,
		"world_name":         worldName,
		"faction_name":       factionName,
		"starter_allocation": services.Level1StarterAllocation(),
		"resources":          resourceBalances,
		"city_stats":         cityStatsPayload,
		// Daily plan context sent on first load for player's AI (provider/local/governor > server fallback).
		"daily_plan_context": buildDailyPlanContextWithResources(p, resourceBalances, cityStatsPayload),
	})
}

// SaveIAAgent creates an IA agent or companion for a ProfileGamer.
// Rules: multiple agents OK, but only 1 companion (is_companion=true) per profile.
// The companion can reference the IA companion chosen at profile creation (ProfileGamer.IACompanionID).
// Supports avatar for agents.
// Endpoint used by MmoCreationAgentIAScreen.
func (h *ProfileHandler) SaveIAAgent(c *gin.Context) {
	var req struct {
		ProfileGamerID uint   `json:"profile_gamer_id"`
		Name           string `json:"name"`
		Role           string `json:"role"`
		Personality    string `json:"personality"`
		Provider       string `json:"provider"`
		Model          string `json:"model"`
		AvatarID       uint   `json:"avatar_id"`
		IsCompanion    bool   `json:"is_companion"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if req.ProfileGamerID == 0 || req.Name == "" || req.Role == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profile_gamer_id, name and role are required"})
		return
	}

	// Enforce only 1 companion per profile
	if req.IsCompanion {
		var count int64
		h.db.Model(&models.MmoIAAgent{}).Where("profile_gamer_id = ? AND is_companion = ?", req.ProfileGamerID, true).Count(&count)
		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "only one companion IA allowed per profile"})
			return
		}
	}

	agent := models.MmoIAAgent{
		ProfileGamerID: req.ProfileGamerID,
		Name:           req.Name,
		Role:           req.Role,
		Personality:    req.Personality,
		Provider:       req.Provider,
		Model:          req.Model,
		AvatarID:       req.AvatarID,
		IsCompanion:    req.IsCompanion,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := h.db.Create(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save ia agent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agent": agent, "message": "ia agent/companion saved"})
}

// ListIAAgents for a profile (used for sync on load)
func (h *ProfileHandler) ListIAAgents(c *gin.Context) {
	profileIDStr := c.Param("id")
	profileID, err := strconv.ParseUint(profileIDStr, 10, 64)
	if err != nil || profileID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}

	var agents []models.MmoIAAgent
	if err := h.db.Where("profile_gamer_id = ?", uint(profileID)).Order("created_at desc").Find(&agents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

// GetDailyPlanContext returns the safe official context for the player's AI provider (or fallback).
// This is sent to Flutter on first load / profile bootstrap.
// Flutter then sends it to the player's configured provider (or local model / governor agent).
// Server rules are included so the AI knows it cannot apply changes directly.
func (h *ProfileHandler) GetDailyPlanContext(c *gin.Context) {
	profileIDStr := c.Param("id")
	profileID, err := strconv.ParseUint(profileIDStr, 10, 64)
	if err != nil || profileID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}

	var p models.ProfileGamer
	if err := h.db.First(&p, profileID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}
	resourceBalances, cityStatsPayload := h.profileResourcePayload(c, p.ID)
	foodBalance := cityStatFloat(cityStatsPayload, "foodBalance")
	if foodBalance == 0 {
		foodBalance = cityStatFloat(cityStatsPayload, "food_balance")
	}

	ctx := models.DailyPlanContext{
		ProfileGamerID: p.ID,
		PlayerStyle:    "balanced",
		City: map[string]interface{}{
			"population":         p.Population,
			"populationCapacity": p.PopulationCapacity,
			"morale":             p.Morale,
			"security":           p.Security,
			"energyBalance":      p.EnergyBalance,
			"foodBalance":        foodBalance,
		},
		Resources: map[string]interface{}{
			"credits": resourceBalances["credits"],
			"metal":   resourceBalances["metal"],
			"energy":  resourceBalances["energy"],
			"food":    resourceBalances["food"],
		},
		ActiveQueues: map[string]interface{}{
			"construction": []interface{}{},
			"research":     []interface{}{},
			"training":     []interface{}{},
		},
		AvailableActions: []string{"build", "upgrade", "train_unit", "start_research", "collect", "explore"},
		ServerRules: []string{
			"The AI cannot apply actions directly.",
			"Costs and impacts are validated by the server via /actions/validate.",
			"All changes require /actions/resolve after player confirmation.",
			"Rewards cannot be invented by the AI.",
		},
		GeneratedAt: time.Now().UTC(),
	}

	c.JSON(http.StatusOK, gin.H{"context": ctx, "note": "Send this context to the player's AI provider (or local/governor fallback). Recommendations must be validated by server."})
}

// buildDailyPlanContext is the retrieval function for data sent to Flutter on first load (profile/bootstrap).
// It prepares the safe context for the player's AI Daily Plan generation.
// Implements the rules: Player provider > local > Governor > server fallback > algorithmic.
func buildDailyPlanContext(p models.ProfileGamer) map[string]interface{} {
	return buildDailyPlanContextWithResources(p, map[string]int64{}, nil)
}

func buildDailyPlanContextWithResources(p models.ProfileGamer, resources map[string]int64, cityStats map[string]any) map[string]interface{} {
	foodBalance := cityStatFloat(cityStats, "foodBalance")
	if foodBalance == 0 {
		foodBalance = cityStatFloat(cityStats, "food_balance")
	}

	energyValue := resources["energy"]
	if energyValue == 0 {
		energyValue = int64(p.EnergyStored)
	}

	return map[string]interface{}{
		"profile_gamer_id": p.ID,
		"player_style":     "balanced",
		"city": map[string]interface{}{
			"population":         p.Population,
			"populationCapacity": p.PopulationCapacity,
			"morale":             p.Morale,
			"security":           p.Security,
			"energyBalance":      p.EnergyBalance,
			"foodBalance":        foodBalance,
		},
		"resources": map[string]interface{}{
			"credits": resources["credits"],
			"metal":   resources["metal"],
			"energy":  energyValue,
			"food":    resources["food"],
		},
		"active_queues": map[string]interface{}{
			"construction": []interface{}{},
			"research":     []interface{}{},
			"training":     []interface{}{},
		},
		"available_actions": []string{"build", "upgrade", "train_unit", "start_research", "collect", "explore"},
		"server_rules": []string{
			"The AI cannot apply actions directly.",
			"Costs and impacts are validated by the server via /actions/validate.",
			"All changes require /actions/resolve after player confirmation.",
			"Rewards cannot be invented by the AI.",
		},
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}
}

// Note: generate/accept etc. will be added next. For now the context retrieval is the first load data sent to Flutter.

// GetDailyPlan returns (or initializes) the DailyPlan for today for this player.
// Used by the city dashboard card to show the current plan + recommendations.
func (h *ProfileHandler) GetDailyPlan(c *gin.Context) {
	profileIDStr := c.Param("id")
	profileID, err := strconv.ParseUint(profileIDStr, 10, 64)
	if err != nil || profileID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}

	today := time.Now().UTC().Format("2006-01-02")

	var plan models.DailyPlan
	err = h.db.Where("profile_gamer_id = ? AND DATE(generated_at) = ?", profileID, today).First(&plan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create a shell plan with fresh context for today
			ctx := h.buildDailyPlanContextForID(uint(profileID))
			ctxBytes, _ := json.Marshal(ctx)
			plan = models.DailyPlan{
				ProfileGamerID:  uint(profileID),
				Context:         string(ctxBytes),
				Recommendations: "[]",
				GeneratedAt:     time.Now().UTC(),
			}
			_ = h.db.Create(&plan).Error
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Parse for response
	var recs []models.DailyPlanRecommendation
	_ = json.Unmarshal([]byte(plan.Recommendations), &recs)

	c.JSON(http.StatusOK, gin.H{
		"plan":            plan,
		"recommendations": recs,
		"today":           today,
	})
}

// SaveDailyPlanRecommendations saves the AI-generated recommendations for the day.
// Called by Flutter after the player's companion / provider processed the context.
func (h *ProfileHandler) SaveDailyPlanRecommendations(c *gin.Context) {
	profileIDStr := c.Param("id")
	profileID, err := strconv.ParseUint(profileIDStr, 10, 64)
	if err != nil || profileID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}

	var body struct {
		Recommendations []models.DailyPlanRecommendation `json:"recommendations"`
		GeneratedBy     string                           `json:"generated_by"` // "player_provider", "governor_agent", ...
		Summary         string                           `json:"summary"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	recsJSON, _ := json.Marshal(body.Recommendations)

	today := time.Now().UTC().Format("2006-01-02")

	var plan models.DailyPlan
	if err := h.db.Where("profile_gamer_id = ? AND DATE(generated_at) = ?", profileID, today).First(&plan).Error; err != nil {
		// create new
		ctx := h.buildDailyPlanContextForID(uint(profileID))
		ctxBytes, _ := json.Marshal(ctx)
		plan = models.DailyPlan{
			ProfileGamerID:  uint(profileID),
			Context:         string(ctxBytes),
			Recommendations: string(recsJSON),
			Summary:         body.Summary,
			GeneratedBy:     body.GeneratedBy,
			GeneratedAt:     time.Now().UTC(),
		}
		h.db.Create(&plan)
	} else {
		plan.Recommendations = string(recsJSON)
		plan.GeneratedBy = body.GeneratedBy
		plan.Summary = body.Summary
		plan.UpdatedAt = time.Now().UTC()
		h.db.Save(&plan)
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "plan_id": plan.ID})
}

// ApplyDailyPlan applies selected recommendations from the daily plan.
// This is the "Appliquer le plan" action from the city dashboard.
// It performs the effects server-side (local + persisted) and returns the updated stats.
// Design is kept open: new types are easy to add in the switch.
func (h *ProfileHandler) ApplyDailyPlan(c *gin.Context) {
	profileIDStr := c.Param("id")
	profileID, err := strconv.ParseUint(profileIDStr, 10, 64)
	if err != nil || profileID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}

	var body struct {
		Indices []int `json:"indices"` // which recommendations from the saved plan to apply
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body, expected indices"})
		return
	}

	var p models.ProfileGamer
	if err := h.db.First(&p, profileID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}

	today := time.Now().UTC().Format("2006-01-02")
	var plan models.DailyPlan
	if err := h.db.Where("profile_gamer_id = ? AND DATE(generated_at) = ?", profileID, today).First(&plan).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no daily plan for today"})
		return
	}

	var recs []models.DailyPlanRecommendation
	if err := json.Unmarshal([]byte(plan.Recommendations), &recs); err != nil {
		recs = []models.DailyPlanRecommendation{}
	}

	applied := []map[string]interface{}{}
	impacts := map[string]interface{}{}

	for _, idx := range body.Indices {
		if idx < 0 || idx >= len(recs) {
			continue
		}
		rec := recs[idx]

		// Apply effect based on type. This is the server application.
		// Later this should delegate to a central ActionResolver that also handles queues, costs, validation.
		switch rec.Type {
		case "train_unit", "train":
			// Simple effect: increase power / "trained" feeling + small security
			p.Power += 8
			p.Security = min(100, p.Security+3)
			impacts["power"] = p.Power
			impacts["security"] = p.Security

		case "upgrade", "build":
			// Building/upgrade effect on city
			p.EnergyProduction += 12
			p.EnergyBalance += 8
			p.Morale = min(100, p.Morale+2)
			p.PopulationCapacity += 5
			impacts["energyProduction"] = p.EnergyProduction
			impacts["morale"] = p.Morale
			impacts["populationCapacity"] = p.PopulationCapacity

		case "start_research", "research":
			p.Morale = min(100, p.Morale+4)
			p.Power += 3
			impacts["morale"] = p.Morale

		case "collect":
			p.EnergyStored += 25
			impacts["energyStored"] = p.EnergyStored

		default:
			// Generic positive nudge so the feature is always useful while we add more types
			p.Morale = min(100, p.Morale+1)
			p.Security = min(100, p.Security+1)
		}

		applied = append(applied, map[string]interface{}{
			"priority": rec.Priority,
			"type":     rec.Type,
			"title":    rec.Title,
			"reason":   rec.Reason,
		})
	}

	// Persist profile changes
	if err := h.db.Save(&p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist profile after apply"})
		return
	}

	// Mark applied in the plan (simple: append to a field or just return the list)
	// For openness we just return what was applied; full history can be added later.
	plan.UpdatedAt = time.Now().UTC()
	h.db.Save(&plan)

	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"applied": applied,
		"impacts": impacts,
		"profile_snapshot": gin.H{
			"level":              p.Level,
			"power":              p.Power,
			"population":         p.Population,
			"populationCapacity": p.PopulationCapacity,
			"morale":             p.Morale,
			"security":           p.Security,
			"energyProduction":   p.EnergyProduction,
			"energyBalance":      p.EnergyBalance,
			"energyStored":       p.EnergyStored,
		},
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (h *ProfileHandler) buildDailyPlanContextForID(profileID uint) map[string]interface{} {
	var p models.ProfileGamer
	h.db.First(&p, profileID)
	resourceBalances, cityStatsPayload := h.profileResourcePayload(nil, p.ID)
	return buildDailyPlanContextWithResources(p, resourceBalances, cityStatsPayload)
}

func (h *ProfileHandler) profileResourcePayload(c *gin.Context, profileID uint) (map[string]int64, map[string]any) {
	balances := map[string]int64{}
	if profileID == 0 {
		return balances, map[string]any{}
	}

	snapshot, err := services.NewResourceService(h.db).PlayerSnapshot(contextForRequest(c), profileID)
	if err != nil {
		return balances, map[string]any{}
	}

	resourcesRaw, ok := snapshot["resources"]
	if ok {
		switch list := resourcesRaw.(type) {
		case []models.PlayerResource:
			for _, item := range list {
				balances[item.ResourceCode] = item.Amount
			}
		case []interface{}:
			for _, item := range list {
				entry, isMap := item.(map[string]interface{})
				if !isMap {
					continue
				}
				code, _ := entry["resourceCode"].(string)
				if code == "" {
					code, _ = entry["resource_code"].(string)
				}
				if code == "" {
					continue
				}
				if amount, ok := entry["amount"].(float64); ok {
					balances[code] = int64(amount)
				}
			}
		}
	}

	cityStats := map[string]any{}
	if raw, ok := snapshot["cityStats"]; ok {
		switch typed := raw.(type) {
		case map[string]any:
			cityStats = typed
		case models.PlayerCityStats:
			cityStats = map[string]any{
				"storageCapacity": typed.StorageCapacity,
				"foodProduction":  typed.FoodProduction,
				"foodConsumption": typed.FoodConsumption,
				"foodBalance":     typed.FoodBalance,
			}
		}
	}

	return balances, cityStats
}

func contextForRequest(c *gin.Context) context.Context {
	if c == nil || c.Request == nil {
		return context.Background()
	}
	return c.Request.Context()
}

func cityStatFloat(stats map[string]any, key string) float64 {
	if stats == nil {
		return 0
	}
	value, ok := stats[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}
