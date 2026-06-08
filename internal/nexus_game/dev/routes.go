package dev

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	envEnabled = "NEXUS_DEV_BACKDOOR_ENABLED"
	envToken   = "NEXUS_DEV_BACKDOOR_TOKEN"
)

func RegisterRoutes(router *gin.Engine) {
	group := router.Group("/api/nexus-game/dev")

	group.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"enabled":        isEnabled(),
			"token_required": true,
			"environment":    gin.Mode(),
			"safe_in_prod":   !isEnabled(),
			"now":            time.Now().UTC().Format(time.RFC3339),
		})
	})

	group.POST("/login", protected("login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"dev_session": gin.H{
				"player_id":  "dev-player",
				"role":       "nexus-dev",
				"expires_at": time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339),
			},
		})
	}))

	group.POST("/reset-player", protected("reset-player", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "reset_queued", "player_id": devPlayerID(c)})
	}))

	group.POST("/seed-player", protected("seed-player", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "seeded", "player_id": devPlayerID(c)})
	}))

	group.POST("/grant-resources", protected("grant-resources", func(c *gin.Context) {
		var req struct {
			PlayerID  string         `json:"player_id"`
			Resources map[string]int `json:"resources"`
		}
		_ = c.ShouldBindJSON(&req)
		if req.PlayerID == "" {
			req.PlayerID = "dev-player"
		}
		if req.Resources == nil {
			req.Resources = map[string]int{"credits": 1000, "metal": 1000, "food": 1000}
		}
		c.JSON(http.StatusOK, gin.H{
			"status":    "granted",
			"player_id": req.PlayerID,
			"resources": req.Resources,
		})
	}))

	group.POST("/fast-forward", protected("fast-forward", func(c *gin.Context) {
		var req struct {
			Seconds int `json:"seconds"`
		}
		_ = c.ShouldBindJSON(&req)
		if req.Seconds <= 0 {
			req.Seconds = 60
		}
		c.JSON(http.StatusOK, gin.H{"status": "advanced", "seconds": req.Seconds})
	}))
}

func protected(action string, handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isEnabled() {
			log.Printf("nexus dev backdoor denied action=%s reason=disabled ip=%s", action, c.ClientIP())
			c.JSON(http.StatusForbidden, gin.H{"error": "nexus dev backdoor disabled"})
			return
		}
		if !validToken(c) {
			log.Printf("nexus dev backdoor denied action=%s reason=bad_token ip=%s", action, c.ClientIP())
			c.JSON(http.StatusUnauthorized, gin.H{"error": "nexus dev token required"})
			return
		}
		log.Printf("nexus dev backdoor action=%s ip=%s", action, c.ClientIP())
		handler(c)
	}
}

func isEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(envEnabled)))
	return value == "1" || value == "true" || value == "yes"
}

func validToken(c *gin.Context) bool {
	expected := strings.TrimSpace(os.Getenv(envToken))
	if expected == "" {
		return false
	}

	token := strings.TrimSpace(c.GetHeader("X-Nexus-Dev-Token"))
	if token == "" {
		auth := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			token = strings.TrimSpace(auth[len("bearer "):])
		}
	}
	return token != "" && token == expected
}

func devPlayerID(c *gin.Context) string {
	var req struct {
		PlayerID string `json:"player_id"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.PlayerID == "" {
		return "dev-player"
	}
	return req.PlayerID
}
