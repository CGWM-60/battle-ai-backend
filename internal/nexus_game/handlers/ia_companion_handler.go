package handlers

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"cgwm/battle/internal/nexus_game/models"

	"github.com/chai2010/webp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IACompanionHandler struct {
	db *gorm.DB
}

func NewIACompanionHandler(db *gorm.DB) *IACompanionHandler {
	return &IACompanionHandler{db: db}
}

func companionBaseDir() string {
	return filepath.Join(assetsBaseDir(), "companion")
}

func companionBaseURL() string {
	return path.Join(assetsBaseURL(), "companion")
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

// Create a new IA companion for a player (admin or in-game) - name + image WebP
func (h *IACompanionHandler) Create(c *gin.Context) {
	baseDir := companionBaseDir()
	baseURL := companionBaseURL()
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create directory"})
		return
	}

	name := c.PostForm("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	file, _, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image file is required"})
		return
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read image"})
		return
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image format"})
		return
	}

	var webpBuf bytes.Buffer
	if err := webp.Encode(&webpBuf, img, &webp.Options{Quality: 80}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to convert to webp"})
		return
	}

	filename := uuid.New().String() + ".webp"
	fullPath := filepath.Join(baseDir, filename)
	if err := os.WriteFile(fullPath, webpBuf.Bytes(), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save"})
		return
	}

	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	fullURL := fmt.Sprintf("%s://%s%s/%s", scheme, c.Request.Host, baseURL, filename)

	cpn := models.IACompanion{
		PlayerID:  1, // demo
		Name:      name,
		Role:      c.PostForm("role"),
		Level:     1,
		Filename:  filename,
		URL:       fullURL,
		CreatedAt: time.Now(),
	}

	if err := h.db.Create(&cpn).Error; err != nil {
		os.Remove(fullPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save to database"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ia_companion": cpn})
}

// Update
func (h *IACompanionHandler) Update(c *gin.Context) {
	baseDir := companionBaseDir()
	baseURL := companionBaseURL()
	id := c.Param("id")
	var cpn models.IACompanion
	if err := h.db.First(&cpn, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "companion not found"})
		return
	}

	name := c.PostForm("name")
	if name != "" {
		cpn.Name = name
	}
	role := c.PostForm("role")
	if role != "" {
		cpn.Role = role
	}

	if file, err := c.FormFile("image"); err == nil {
		f2, _ := file.Open()
		defer f2.Close()
		imgBytes, _ := io.ReadAll(f2)
		img, _, _ := image.Decode(bytes.NewReader(imgBytes))
		var webpBuf bytes.Buffer
		webp.Encode(&webpBuf, img, &webp.Options{Quality: 80})
		filename := uuid.New().String() + ".webp"
		fullPath := filepath.Join(baseDir, filename)
		os.WriteFile(fullPath, webpBuf.Bytes(), 0644)
		os.Remove(filepath.Join(baseDir, cpn.Filename))
		scheme := "https"
		if c.Request.TLS == nil {
			scheme = "http"
		}
		cpn.Filename = filename
		cpn.URL = fmt.Sprintf("%s://%s%s/%s", scheme, c.Request.Host, baseURL, filename)
	}

	if err := h.db.Save(&cpn).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ia_companion": cpn})
}

// Delete
func (h *IACompanionHandler) Delete(c *gin.Context) {
	baseDir := companionBaseDir()
	id := c.Param("id")
	var cpn models.IACompanion
	if err := h.db.First(&cpn, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "companion not found"})
		return
	}
	os.Remove(filepath.Join(baseDir, cpn.Filename))
	h.db.Delete(&cpn)
	c.JSON(http.StatusOK, gin.H{"success": true})
}
