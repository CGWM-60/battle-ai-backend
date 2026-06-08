package handlers

import (
	"net/http"
	"time"

	"cgwm/battle/internal/nexus_game/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type IACompanionHandler struct {
	db *gorm.DB
}

func NewIACompanionHandler(db *gorm.DB) *IACompanionHandler {
	return &IACompanionHandler{db: db}
}

// List all companions (for admin, or filter by player later)
func (h *IACompanionHandler) List(c *gin.Context) {
	var companions []models.IACompanion
	if err := h.db.Order("created_at desc").Find(&companions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ia_companions": companions})
}

// Create a new IA companion for a player (admin or in-game)
func (h *IACompanionHandler) Create(c *gin.Context) {
	var cpn models.IACompanion
	if err := c.ShouldBindJSON(&cpn); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cpn.ID = 0
	cpn.CreatedAt = time.Now()
	if err := h.db.Create(&cpn).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ia_companion": cpn})
}

// Update
func (h *IACompanionHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var cpn models.IACompanion
	if err := h.db.First(&cpn, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "companion not found"})
		return
	}
	var input models.IACompanion
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cpn.Name = input.Name
	cpn.Role = input.Role
	cpn.Level = input.Level
	if err := h.db.Save(&cpn).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ia_companion": cpn})
}

// Delete
func (h *IACompanionHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&models.IACompanion{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}