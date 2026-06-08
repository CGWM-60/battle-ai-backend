package handlers

import (
	"net/http"
	"time"

	"cgwm/battle/internal/nexus_game/models"
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
//
// For now: simulation with 10s delay + TODO.
type BootstrapHandler struct {
	db *gorm.DB
}

func NewBootstrapHandler(db *gorm.DB) *BootstrapHandler {
	return &BootstrapHandler{db: db}
}

// Load is the main bootstrap endpoint.
// TODO: real implementation
// - Load player data from DB (city, resources, agents)
// - Load asset manifest (or return references to asset service)
// - Load active quests + world quests
// - Load world conditions / events
// - etc.
func (h *BootstrapHandler) Load(c *gin.Context) {
	// Simulation: 10 seconds delay as requested for Etape 1
	time.Sleep(10 * time.Second)

	avatarInfo := gin.H{
		"name": "",
		"url":  "",
	}

	// Etape 2: try to load latest avatar for demo player (PlayerID=1)
	if h.db != nil {
		var avatar models.Avatar
		if err := h.db.Where("player_id = ?", 1).Order("created_at desc").First(&avatar).Error; err == nil {
			avatarInfo = gin.H{
				"name": avatar.Name,
				"url":  avatar.URL,
			}
		}
	}

	// Simulated response structure (will be enriched later)
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Bootstrap simulation (10s delay)",
		"player": gin.H{
			"id":           1,
			"name":         "VOIDKAT_77",
			"city_level":   3,
			"population":   142,
			"energy":       87,
			"resources": gin.H{
				"metal":   1240,
				"quantum": 320,
			},
			// Etape 2: avatar included in base load
			"avatar": avatarInfo,
		},
		"assets": gin.H{
			"version":  "v0.1-sim",
			"base_url": "https://assets.example.com/nexus",
		},
		"quests": []gin.H{
			{"id": 101, "title": "Secure the northern relay", "type": "world"},
			{"id": 102, "title": "Upgrade habitat to level 4", "type": "city"},
		},
		"world": gin.H{
			"current_tick":  12847,
			"active_events": 2,
		},
		// TODO: add agents, lore summary, faction rep, etc.
	})
}