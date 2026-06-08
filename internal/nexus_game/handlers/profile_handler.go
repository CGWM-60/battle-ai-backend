package handlers

import (
	"net/http"
	"strconv"
	"time"

	"cgwm/battle/internal/nexus_game/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ProfileHandler manages the ProfileGamer (may be empty for new players).
// All saves go through here. Server validates + applies. Flutter displays result.
type ProfileHandler struct {
	db *gorm.DB
}

func NewProfileHandler(db *gorm.DB) *ProfileHandler {
	return &ProfileHandler{db: db}
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
		// not found -> empty profile (allowed)
		c.JSON(http.StatusOK, gin.H{
			"exists": false,
			"profile": models.ProfileGamer{
				UserID: uint(uid),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"exists": true, "profile": p})
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
		// create new
		p = models.ProfileGamer{
			UserID:        req.UserID,
			AvatarID:      req.AvatarID,
			FactionID:     req.FactionID,
			IACompanionID: req.IACompanionID,
			Pseudo:        req.Pseudo,
			CityName:      req.CityName,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := h.db.Create(&p).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create profile_gamer"})
			return
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

	c.JSON(http.StatusOK, gin.H{"profile": p, "exists": true})
}
