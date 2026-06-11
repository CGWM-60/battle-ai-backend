package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// BootstrapHandler handles the initial data loading for a Nexus Game player.
// This endpoint is meant to return:
// - Assets manifest / CDN urls
// - Player city / resources / population info
// - Available quests (world + personal)
// - World state (regions, events, etc.)
// - Pre-assigned IA agents status
// - Living Lore summary, etc.
// - Avatar (name + full URL) - Etape 2
type BootstrapHandler struct {
	db *gorm.DB
}

func NewBootstrapHandler(db *gorm.DB) *BootstrapHandler {
	return &BootstrapHandler{db: db}
}

func (h *BootstrapHandler) Load(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}

	profile, err := h.bootstrapProfile(c)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	resourceSnapshot, err := services.NewResourceService(h.db).PlayerSnapshot(c.Request.Context(), profile.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.db.First(&profile, profile.ID).Error

	var avatar models.Avatar
	avatarInfo := gin.H{}
	if profile.AvatarID > 0 {
		if err := h.db.First(&avatar, profile.AvatarID).Error; err == nil {
			avatarInfo = gin.H{"id": avatar.ID, "name": avatar.Name, "url": avatar.URL}
		}
	}
	if len(avatarInfo) == 0 {
		if err := h.db.Where("player_id = ?", profile.UserID).Order("created_at desc").First(&avatar).Error; err == nil {
			avatarInfo = gin.H{"id": avatar.ID, "name": avatar.Name, "url": avatar.URL}
		}
	}

	var faction models.Faction
	factionInfo := gin.H{}
	if profile.FactionID > 0 {
		if err := h.db.First(&faction, profile.FactionID).Error; err == nil {
			factionInfo = gin.H{"id": faction.ID, "name": faction.Name, "description": faction.Description, "color": faction.Color, "url": faction.URL}
		}
	}

	var companion models.IACompanion
	companionInfo := gin.H{}
	if profile.IACompanionID > 0 {
		if err := h.db.First(&companion, profile.IACompanionID).Error; err == nil {
			companionInfo = gin.H{"id": companion.ID, "name": companion.Name, "role": companion.Role, "level": companion.Level, "url": companion.URL}
		}
	}
	var companions []models.IACompanion
	_ = h.db.Where("player_id = ?", profile.UserID).Order("created_at desc").Find(&companions).Error

	var world models.World
	worldInfo := gin.H{}
	if profile.WorldID > 0 {
		if err := h.db.First(&world, profile.WorldID).Error; err == nil {
			worldInfo = gin.H{"id": world.ID, "name": world.Name, "isActive": world.IsActive}
		}
	}
	var continent models.Continent
	continentInfo := gin.H{}
	if profile.ContinentID > 0 {
		if err := h.db.First(&continent, profile.ContinentID).Error; err == nil {
			continentInfo = gin.H{"id": continent.ID, "worldId": continent.WorldID, "name": continent.Name, "maxPlayers": continent.MaxPlayers, "maxFactions": continent.MaxFactions}
		}
	}

	var constructionQueue []models.PlayerBuilding
	_ = h.db.Where("profile_gamer_id = ? AND is_constructing = ?", profile.ID, true).Order("construction_ends_at asc").Find(&constructionQueue).Error

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Bootstrap Nexus ready",
		"profile": profile,
		"player": gin.H{
			"id":                 profile.ID,
			"userId":             profile.UserID,
			"pseudo":             profile.Pseudo,
			"cityName":           profile.CityName,
			"level":              profile.Level,
			"population":         profile.Population,
			"populationCapacity": profile.PopulationCapacity,
			"energyProduction":   profile.EnergyProduction,
			"energyConsumption":  profile.EnergyConsumption,
			"energyBalance":      profile.EnergyBalance,
			"avatar":             avatarInfo,
			"faction":            factionInfo,
			"iaCompanion":        companionInfo,
		},
		"assets": gin.H{
			"version": "v1",
			"baseUrl": "/nexus-assets",
		},
		"resources":         resourceSnapshot["resources"],
		"resourceCatalog":   resourceSnapshot["catalog"],
		"resourceHistory":   resourceSnapshot["transactions"],
		"cityStats":         resourceSnapshot["cityStats"],
		"constructionQueue": constructionQueue,
		"quests":            []gin.H{},
		"world": gin.H{
			"current":      worldInfo,
			"continent":    continentInfo,
			"activeEvents": []gin.H{},
		},
		"iaCompanions": companions,
	})
}

func (h *BootstrapHandler) bootstrapProfile(c *gin.Context) (models.ProfileGamer, error) {
	var profile models.ProfileGamer
	profileID := firstBootstrapUint(c, "profileGamerId", "profileId", "profile_gamer_id", "profile_id")
	if profileID > 0 {
		err := h.db.First(&profile, profileID).Error
		return profile, err
	}
	userID := firstBootstrapUint(c, "user_id", "userId")
	if userID == 0 {
		return profile, errors.New("profileGamerId or user_id is required")
	}
	err := h.db.Where("user_id = ?", userID).First(&profile).Error
	return profile, err
}

func firstBootstrapUint(c *gin.Context, keys ...string) uint {
	for _, key := range keys {
		raw := c.Query(key)
		if raw == "" {
			continue
		}
		value, err := strconv.ParseUint(raw, 10, 64)
		if err == nil && value > 0 {
			return uint(value)
		}
	}
	return 0
}
