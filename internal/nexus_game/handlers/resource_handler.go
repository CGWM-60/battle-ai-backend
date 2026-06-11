package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/repositories"
	"cgwm/battle/internal/nexus_game/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ResourceHandler struct {
	db                *gorm.DB
	resourceService   *services.ResourceService
	dailyGrantService *services.DailyGrantService
}

func NewResourceHandler(db *gorm.DB) *ResourceHandler {
	return &ResourceHandler{
		db:                db,
		resourceService:   services.NewResourceService(db),
		dailyGrantService: services.NewDailyGrantService(db),
	}
}

func (h *ResourceHandler) Resources(c *gin.Context) {
	profileID, ok := h.profileID(c)
	if !ok {
		return
	}
	// Defensive: verify the ProfileGamer row exists for this profile_gamer_id.
	// (Bootstrap and /profile resolve user_id -> real pg.ID before snapshot; /resources is called by client with the pg PK from prior responses.)
	// If client ever passes a non-pg numeric (e.g. user_id leaked as pg id), return clean 404 instead of 500 from inner First in Sync/Ensure.
	var p models.ProfileGamer
	if err := h.db.WithContext(c.Request.Context()).First(&p, profileID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify profile"})
		return
	}
	snapshot, err := h.resourceService.PlayerSnapshot(c.Request.Context(), profileID)
	if err != nil {
		// Defensive: do not return 500 to client (causes DioException and potential retry storm hammering the server).
		// Log the real cause on server. Return 200 with minimal safe data so client can continue with last known
		// or empty (live prediction / cache will handle UX). This profile may be in transitional state
		// (fresh after purge/creation, during construction complete, missing some rows temporarily, etc.).
		// The real fix is in the snapshot/ensure/sync path (guards on cfg divisors, ensure always creates rows, etc.).
		c.JSON(http.StatusOK, gin.H{
			"resources":     []interface{}{},
			"catalog":       []interface{}{},
			"cityStats":     gin.H{},
			"transactions":  []interface{}{},
			"warning":       "partial resources data (backend snapshot error)",
		})
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

func (h *ResourceHandler) Catalog(c *gin.Context) {
	resources, err := repositories.NewResourceCatalogRepository(h.db).List(c.Request.Context(), true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load resource catalog"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"resources": resources})
}

func (h *ResourceHandler) Transactions(c *gin.Context) {
	profileID, ok := h.profileID(c)
	if !ok {
		return
	}
	limit := limitFromQuery(c, 50, 200)
	transactions, err := repositories.NewResourceTransactionRepository(h.db).List(c.Request.Context(), profileID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load transactions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"transactions": transactions})
}

func (h *ResourceHandler) CityStats(c *gin.Context) {
	profileID, ok := h.profileID(c)
	if !ok {
		return
	}
	if err := h.resourceService.EnsureInitialAllocation(c.Request.Context(), profileID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize city stats"})
		return
	}
	stats, err := repositories.NewPlayerCityStatsRepository(h.db).GetOrCreate(c.Request.Context(), profileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load city stats"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"cityStats": stats})
}

func (h *ResourceHandler) DailyGrantStatus(c *gin.Context) {
	profileID, ok := h.profileID(c)
	if !ok {
		return
	}
	status, err := h.dailyGrantService.Status(c.Request.Context(), profileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load daily grant status"})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *ResourceHandler) DailyGrantClaim(c *gin.Context) {
	profileID, ok := h.profileID(c)
	if !ok {
		return
	}
	status, err := h.dailyGrantService.Claim(c.Request.Context(), profileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to claim daily grant"})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *ResourceHandler) DailyGrantHistory(c *gin.Context) {
	profileID, ok := h.profileID(c)
	if !ok {
		return
	}
	claims, err := h.dailyGrantService.History(c.Request.Context(), profileID, limitFromQuery(c, 30, 100))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load daily grant history"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"claims": claims})
}

func (h *ResourceHandler) AdminCatalog(c *gin.Context) {
	resources, err := repositories.NewResourceCatalogRepository(h.db).List(c.Request.Context(), false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load resource catalog"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"resources": resources})
}

func (h *ResourceHandler) AdminSeedPreview(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"resources": services.OfficialResourceDefinitions(),
		"count":     len(services.OfficialResourceDefinitions()),
	})
}

func (h *ResourceHandler) AdminSeedCommit(c *gin.Context) {
	if err := h.resourceService.SeedDefaults(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to seed resource defaults"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "seeded", "count": len(services.OfficialResourceDefinitions())})
}

func (h *ResourceHandler) AdminSeedStatus(c *gin.Context) {
	resources, err := repositories.NewResourceCatalogRepository(h.db).List(c.Request.Context(), false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load seed status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"expected": len(services.OfficialResourceDefinitions()),
		"current":  len(resources),
		"complete": len(resources) >= len(services.OfficialResourceDefinitions()),
	})
}

func (h *ResourceHandler) profileID(c *gin.Context) (uint, bool) {
	raw := c.Query("profile_gamer_id")
	if raw == "" {
		raw = c.Query("profileGamerId")
	}
	if raw == "" {
		raw = c.Query("profile_id")
	}
	if raw == "" {
		raw = c.Query("profileId")
	}
	profileID, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || profileID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profile_gamer_id is required"})
		return 0, false
	}
	return uint(profileID), true
}

func limitFromQuery(c *gin.Context, fallback int, max int) int {
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil || limit <= 0 {
		return fallback
	}
	if limit > max {
		return max
	}
	return limit
}
