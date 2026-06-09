package handlers

import (
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
			"exists": false,
			"profile": models.ProfileGamer{
				UserID: uint(uid),
				Level:  1,
				Power:  0,
			},
		})
		return
	}

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
		"exists":       true,
		"profile":      p,
		"avatar_url":   avatarURL,
		"world_name":   worldName,
		"faction_name": factionName,
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
		UserID        uint   `json:"user_id" form:"user_id"`
		AvatarID      uint   `json:"avatar_id" form:"avatar_id"`
		FactionID     uint   `json:"faction_id" form:"faction_id"`
		IACompanionID uint   `json:"ia_companion_id" form:"ia_companion_id"`
		Pseudo        string `json:"pseudo" form:"pseudo"`
		CityName      string `json:"city_name" form:"city_name"`
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
			CreatedAt:     now,
			UpdatedAt:     now,
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
	} else {
		// update existing (only the gamer fields; never touch other tables here)
		p.AvatarID = req.AvatarID
		p.FactionID = req.FactionID
		p.IACompanionID = req.IACompanionID
		p.Pseudo = req.Pseudo
		p.CityName = req.CityName
		p.UpdatedAt = now
		if err := h.db.Save(&p).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile_gamer"})
			return
		}
	}

	// Enrich for MmoEntryScreen: avatar URL + world name (so we can show real chosen avatar, pseudo, lvl, power, world name)
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
		"profile":      p,
		"exists":       true,
		"avatar_url":   avatarURL,
		"world_name":   worldName,
		"faction_name": factionName,
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
