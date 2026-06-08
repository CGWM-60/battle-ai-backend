package handlers

import (
	"net/http"

	"cgwm/battle/internal/nexus_game/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FactionHandler struct {
	db *gorm.DB
}

func NewFactionHandler(db *gorm.DB) *FactionHandler {
	return &FactionHandler{db: db}
}

// List all factions
func (h *FactionHandler) List(c *gin.Context) {
	var factions []models.Faction
	if err := h.db.Find(&factions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"factions": factions})
}

// Create a new faction (admin)
func (h *FactionHandler) Create(c *gin.Context) {
	var f models.Faction
	if err := c.ShouldBindJSON(&f); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	f.ID = 0 // ensure new
	if err := h.db.Create(&f).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"faction": f})
}

// Update faction
func (h *FactionHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var f models.Faction
	if err := h.db.First(&f, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "faction not found"})
		return
	}
	var input models.Faction
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	f.Name = input.Name
	f.Description = input.Description
	f.Color = input.Color
	if err := h.db.Save(&f).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"faction": f})
}

// Delete faction
func (h *FactionHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&models.Faction{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}