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

var (
	avatarBaseDir = getEnv("NEXUS_ASSET_DIR", "/nexus_game/assets/avatar")
	avatarBaseURL = getEnv("NEXUS_ASSET_BASE_URL", "/nexus_game/assets/avatar")
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

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

// List returns all avatars (for admin)
func (h *AvatarHandler) List(c *gin.Context) {
	var avatars []models.Avatar
	if err := h.db.Order("created_at desc").Find(&avatars).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"avatars": avatars})
}

// Update avatar name and/or image (CRUD popin)
func (h *AvatarHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var avatar models.Avatar
	if err := h.db.First(&avatar, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
		return
	}

	name := c.PostForm("name")
	if name != "" {
		avatar.Name = name
	}

	// if new image
	if file, err := c.FormFile("image"); err == nil {
		f, _ := file.Open()
		defer f.Close()
		imgBytes, err := io.ReadAll(f)
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
		fullPath := filepath.Join(avatarBaseDir, filename)
		if err := os.WriteFile(fullPath, webpBuf.Bytes(), 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save avatar"})
			return
		}
		// delete old
		oldPath := filepath.Join(avatarBaseDir, avatar.Filename)
		os.Remove(oldPath)

		avatar.Filename = filename
		scheme := "https"
		if c.Request.TLS == nil {
			scheme = "http"
		}
		avatar.URL = fmt.Sprintf("%s://%s%s/%s", scheme, c.Request.Host, avatarBaseURL, filename)
	}

	if err := h.db.Save(&avatar).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"avatar": avatar})
}

// Delete avatar (and file)
func (h *AvatarHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	var avatar models.Avatar
	if err := h.db.First(&avatar, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "avatar not found"})
		return
	}
	// delete file
	fullPath := filepath.Join(avatarBaseDir, avatar.Filename)
	os.Remove(fullPath)

	if err := h.db.Delete(&avatar).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}