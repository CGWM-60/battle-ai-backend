package handlers

import (
	"net/http"
	"strconv"

	"cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/nexus_game/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// WorldHandler provides endpoints for "gestion des world" (world management card/UI).
// All world creation, capacity, faction/player assignment logic lives here + service.
// Uses Redis heavily for counters, locks, assignments.
// Includes server AI triggers for events, summaries (visible/modifiable via these pages).
// Prompts optimized (cost/speed + detailed/enriching), evolve with world state.
type WorldHandler struct {
	db    *gorm.DB
	redis *cache.RedisService
	svc   *services.WorldService
	aiSvc *services.ServerAIService
}

func NewWorldHandler(db *gorm.DB, redis *cache.RedisService) *WorldHandler {
	return &WorldHandler{
		db:    db,
		redis: redis,
		svc:   services.NewWorldService(db, redis),
		aiSvc: services.NewServerAIService(db, redis),
	}
}

// ListWorlds - entry "card" data for world management UI.
// Returns worlds with continent capacities (Redis real-time).
func (h *WorldHandler) ListWorlds(c *gin.Context) {
	worlds, err := h.svc.ListWorlds(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"worlds": worlds})
}

// CreateWorld - admin can force new world (auto creates 5 continents).
func (h *WorldHandler) CreateWorld(c *gin.Context) {
	w, err := h.svc.CreateWorld(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"world": w})
}

// GetWorld detail.
func (h *WorldHandler) GetWorld(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	// For demo, return basic; extend with Redis capacities.
	c.JSON(http.StatusOK, gin.H{"world_id": id, "message": "extend with capacities from Redis"})
}

// ListContinents - for management.
func (h *WorldHandler) ListContinents(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "use ListWorlds for full view with Redis capacities"})
}

// GenerateWorldEvent - "page" endpoint in world gestion to trigger server IA for event proposal.
// Visible/modifiable: admin calls this, sees generated (optimized prompt), can approve.
func (h *WorldHandler) GenerateWorldEvent(c *gin.Context) {
	worldID, _ := strconv.Atoi(c.Param("id"))
	// Stub world state from Redis/DB.
	state := map[string]interface{}{"tensions": 4, "recent_events": 2}
	event, err := h.aiSvc.GenerateWorldEvent(c.Request.Context(), state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"world_id": worldID, "proposed_event": event, "note": "Prompt v1.1 optimized for cost/speed + enriching lore. Evolves with state."})
}

// Note: Assignment logic is called internally from profile/faction handlers (see updates below).
// For full "gestion", expose more admin endpoints as needed (e.g. force assign, view capacities).
// Server AI prompts (in service) are versioned, logged, respect limits (no bypass policies).
