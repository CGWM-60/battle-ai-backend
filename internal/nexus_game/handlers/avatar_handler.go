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
	"time"

	"cgwm/battle/internal/nexus_game/models"
	"github.com/chai2010/webp"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	avatarBaseDir = "/nexus_game/assets/avatar"
	avatarBaseURL = "/nexus_game/assets/avatar"
)

type AvatarHandler struct {
	db *gorm.DB
}

func NewAvatarHandler(db *gorm.DB) *AvatarHandler {
	return &AvatarHandler{db: db}
}

// Upload handles avatar creation: name + image file.
// - Creates dir if needed (volume mounted to survive builds)
// - Converts image to WebP (mandatory)
// - Saves file
// - Stores record in DB
// - Returns name + full URL
func (h *AvatarHandler) Upload(c *gin.Context) {
	// Ensure directory exists (persistent volume)
	if err := os.MkdirAll(avatarBaseDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create avatar directory"})
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

	// Read image
	imgBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read image"})
		return
	}

	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid image format (jpg/png/gif supported)"})
		return
	}

	// Convert to WebP (quality 80)
	var webpBuf bytes.Buffer
	if err := webp.Encode(&webpBuf, img, &webp.Options{Quality: 80}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to convert to webp"})
		return
	}

	// Unique filename
	filename := uuid.New().String() + ".webp"
	fullPath := filepath.Join(avatarBaseDir, filename)

	if err := os.WriteFile(fullPath, webpBuf.Bytes(), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save avatar"})
		return
	}

	// Build full URL (using request host for now; in prod use CDN or env)
	scheme := "https"
	if c.Request.TLS == nil {
		scheme = "http"
	}
	fullURL := fmt.Sprintf("%s://%s%s/%s", scheme, c.Request.Host, avatarBaseURL, filename)

	avatar := models.Avatar{
		PlayerID:  1, // TODO: get from auth context later
		Name:      name,
		Filename:  filename,
		URL:       fullURL,
		CreatedAt: time.Now(),
	}

	if err := h.db.Create(&avatar).Error; err != nil {
		// cleanup file?
		os.Remove(fullPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save avatar to database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"avatar": gin.H{
			"id":   avatar.ID,
			"name": avatar.Name,
			"url":  avatar.URL,
		},
	})
}

// GetCurrent returns the current avatar for the (fake) player.
// Used internally by bootstrap.
func (h *AvatarHandler) GetCurrent(playerID uint) (models.Avatar, error) {
	var avatar models.Avatar
	err := h.db.Where("player_id = ?", playerID).Order("created_at desc").First(&avatar).Error
	return avatar, err
}