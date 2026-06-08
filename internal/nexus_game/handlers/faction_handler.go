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
	"path/filepath"

	"cgwm/battle/internal/nexus_game/models"
	"github.com/chai2010/webp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FactionHandler struct {
	db *gorm.DB
}

func NewFactionHandler(db *gorm.DB) *FactionHandler {
	return &FactionHandler{db: db}
}

var (
	factionBaseDir   = assetsBaseDir + "/faction"
	factionBaseURL   = assetsBaseURL + "/faction"
)

// List all factions
func (h *FactionHandler) List(c *gin.Context) {
	var factions []models.Faction
	if err := h.db.Find(&factions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"factions": factions})
}

// Create a new faction (admin) - name + image (WebP like avatar)
func (h *FactionHandler) Create(c *gin.Context) {
	if err := os.MkdirAll(factionBaseDir, 0755); err != nil {
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
	fullPath := filepath.Join(factionBaseDir, filename)
	if err := os.WriteFile(fullPath, webpBuf.Bytes(), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save"})
		return
	}

	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	fullURL := fmt.Sprintf("%s://%s%s/%s", scheme, c.Request.Host, factionBaseURL, filename)

	f := models.Faction{
		Name:     name,
		Filename: filename,
		URL:      fullURL,
	}

	if err := h.db.Create(&f).Error; err != nil {
		os.Remove(fullPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save to database"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"faction": f})
}

// Update faction (name + optional image)
func (h *FactionHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var f models.Faction
	if err := h.db.First(&f, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "faction not found"})
		return
	}

	name := c.PostForm("name")
	if name != "" {
		f.Name = name
	}

	if file, err := c.FormFile("image"); err == nil {
		f2, _ := file.Open()
		defer f2.Close()
		imgBytes, _ := io.ReadAll(f2)
		img, _, _ := image.Decode(bytes.NewReader(imgBytes))
		var webpBuf bytes.Buffer
		webp.Encode(&webpBuf, img, &webp.Options{Quality: 80})
		filename := uuid.New().String() + ".webp"
		fullPath := filepath.Join(factionBaseDir, filename)
		os.WriteFile(fullPath, webpBuf.Bytes(), 0644)
		os.Remove(filepath.Join(factionBaseDir, f.Filename))
		scheme := "https"
		if c.Request.TLS == nil { scheme = "http" }
		f.Filename = filename
		f.URL = fmt.Sprintf("%s://%s%s/%s", scheme, c.Request.Host, factionBaseURL, filename)
	}

	if err := h.db.Save(&f).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"faction": f})
}

// Delete faction
func (h *FactionHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	var f models.Faction
	if err := h.db.First(&f, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "faction not found"})
		return
	}
	os.Remove(filepath.Join(factionBaseDir, f.Filename))
	h.db.Delete(&f)
	c.JSON(http.StatusOK, gin.H{"success": true})
}