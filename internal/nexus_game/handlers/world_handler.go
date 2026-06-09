package handlers

import (
	"net/http"
	"strconv"

	"cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/nexus_game/models"
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
	event, err := h.aiSvc.GenerateWorldEvent(c.Request.Context(), state, uint(worldID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"world_id": worldID, "proposed_event": event, "note": "Prompt v1.1 optimized for cost/speed + enriching lore. Evolves with state."})
}

// Prompt CRUD - fully modifiable in "gestion des world" UI.
// Allows admin to create/update prompts for IA serveur (versioned, optimized).
func (h *WorldHandler) ListPrompts(c *gin.Context) {
	domain := c.Query("domain")
	prompts, err := h.svc.ListPrompts(c.Request.Context(), domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"prompts": prompts})
}

// TriggerWorldTick - for testing/integration in world gestion. Calls the tick which uses Server AI.
func (h *WorldHandler) TriggerWorldTick(c *gin.Context) {
	worldID, _ := strconv.Atoi(c.Param("id"))
	ts := services.NewWorldTickService(h.db, h.redis)
	if err := ts.RunWorldTick(c.Request.Context(), uint(worldID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tick executed", "world_id": worldID})
}

// ListAIOutputs - for admin to see textual outputs of server IA (persisted in Redis).
func (h *WorldHandler) ListAIOutputs(c *gin.Context) {
	outputs, err := h.aiSvc.GetAIOutputs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"outputs": outputs})
}

func (h *WorldHandler) CreatePrompt(c *gin.Context) {
	var p models.Prompt
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.CreatePrompt(c.Request.Context(), &p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"prompt": p})
}

func (h *WorldHandler) UpdatePrompt(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdatePrompt(c.Request.Context(), uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "prompt updated"})
}

// Note: Assignment logic is called internally from profile/faction handlers (see updates below).
// For full "gestion", expose more admin endpoints as needed (e.g. force assign, view capacities).
// Server AI prompts (in service) are versioned, logged, respect limits (no bypass policies).

// RunAIGeneration - endpoint admin pour générer quêtes (quest seeds), événements, living lore, cas tribunal, etc.
// avec les prompts créés/manuellement modifiés dans l'UI Prompts.
// Body: { world_id, feature: "quest_seed"|"world_event"|"living_lore"|"tribunal_case"|"world_summary", prompt_id?, prompt_version?, extra? }
// Réponse inclut l'output structuré (title, summary, details + meta prompt utilisé) + persistance auto (DB+Redis).
// Progression/erreurs/succès sont gérés côté admin UI (appels visibles, steps, result panel).
func (h *WorldHandler) RunAIGeneration(c *gin.Context) {
	var req struct {
		WorldID       uint                   `json:"world_id"`
		Feature       string                 `json:"feature"`
		PromptID      string                 `json:"prompt_id,omitempty"`
		PromptVersion string                 `json:"prompt_version,omitempty"`
		Extra         map[string]interface{} `json:"extra,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body: " + err.Error()})
		return
	}
	if req.Feature == "" || req.WorldID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "world_id and feature are required"})
		return
	}

	var manualPrompt *models.Prompt
	if req.PromptID != "" {
		if p, err := h.svc.GetPrompt(c.Request.Context(), req.PromptID, req.PromptVersion); err == nil && p != nil {
			manualPrompt = p
		}
	}

	out, err := h.aiSvc.RunAIGeneration(c.Request.Context(), req.Feature, req.WorldID, manualPrompt, req.Extra)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"feature":     req.Feature,
		"world_id":    req.WorldID,
		"output":      out,
		"used_prompt": manualPrompt, // may be null if internal fallback
		"note":        "Output persisted to ai_outputs (GORM DB + Redis). Visible in IA Outputs admin page and in-tab history. Uses the exact manual prompt SystemPrompt when provided.",
	})
}
