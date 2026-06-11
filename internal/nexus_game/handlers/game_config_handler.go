package handlers

import (
	"net/http"

	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GameConfigHandler struct {
	service *services.GameBalanceConfigService
}

func NewGameConfigHandler(db *gorm.DB) *GameConfigHandler {
	return &GameConfigHandler{service: services.NewGameBalanceConfigService(db)}
}

func (h *GameConfigHandler) Get(c *gin.Context) {
	cfg, err := h.service.GetActive(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load game config"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"config":  cfg,
		"preview": gameConfigPreview(cfg),
	})
}

func (h *GameConfigHandler) Update(c *gin.Context) {
	var cfg models.GameBalanceConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cfg.UpdatedBy = "admin"
	saved, err := h.service.SaveActive(c.Request.Context(), cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save game config"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"config":  saved,
		"preview": gameConfigPreview(saved),
	})
}

func gameConfigPreview(cfg models.GameBalanceConfig) map[string]any {
	return map[string]any{
		"habitatCapacity": map[string]int{
			"1":  services.PopulationCapacityForHabitatLevel(cfg, 1),
			"2":  services.PopulationCapacityForHabitatLevel(cfg, 2),
			"3":  services.PopulationCapacityForHabitatLevel(cfg, 3),
			"10": services.PopulationCapacityForHabitatLevel(cfg, 10),
			"20": services.PopulationCapacityForHabitatLevel(cfg, 20),
			"30": services.PopulationCapacityForHabitatLevel(cfg, 30),
		},
		"example500Population": map[string]float64{
			"foodPerHour":   500 * cfg.FoodPerPopulationPerHour,
			"energyPerHour": float64(cfg.EnergyBaseConsumptionPerHour + 500/cfg.PopulationPerEnergy),
		},
	}
}
