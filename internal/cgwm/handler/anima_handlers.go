package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"cgwm/battle/internal/cgwm/models"
	"cgwm/battle/internal/cgwm/service"
)

// Cloud handlers (protected by auth middleware in real router)
func UploadSnapshot(c *gin.Context) {
	var snap models.AnimaCloudSnapshot
	if err := c.ShouldBindJSON(&snap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// TODO: owner check via JWT or device id
	if err := service.UploadSnapshot(&snap); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, snap)
}

func DownloadSnapshot(c *gin.Context) {
	animaID := c.Param("id")
	snap, err := service.DownloadSnapshot(animaID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, snap)
}

// Park handlers
func EnterPark(c *gin.Context) {
	var req struct {
		AnimaID string `json:"anima_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	visit, err := service.EnterPark(req.AnimaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, visit)
}

func LeaveAlone(c *gin.Context) {
	// similar binding + call service.LeaveAloneInPark
	c.JSON(http.StatusOK, gin.H{"status": "left alone"})
}

func Heartbeat(c *gin.Context) {
	// update presence lastHeartbeat
	c.JSON(http.StatusOK, gin.H{"status": "heartbeat received"})
}

func ReturnFromPark(c *gin.Context) {
	var req struct { AnimaID string `json:"anima_id"`; VisitID string `json:"visit_id"` }
	if err := c.ShouldBindJSON(&req); err != nil { /*...*/ }
	report, _ := service.ReturnFromPark(req.AnimaID, req.VisitID)
	c.JSON(http.StatusOK, report)
}

// Social
func SubmitSocialCard(c *gin.Context) {
	var card models.AnimaSocialLearningCard
	if err := c.ShouldBindJSON(&card); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Convert to map for the filter (simple for stub)
	raw := map[string]interface{}{
		"public_anima_id": card.SourcePublicAnimaID,
		"topic":           card.Topic,
		"lesson":          card.Lesson,
		"emotional_tone":  card.EmotionalTone,
		"confidence":      card.Confidence,
		"tags":            card.Tags,
	}
	safe, ok := service.SubmitSocialCard(raw)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsafe or low safety score"})
		return
	}
	c.JSON(http.StatusOK, safe)
}

// Admin (protected)
func AdminSyncState(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"syncs": []string{}, "errors": 0})
}

func AdminParkState(c *gin.Context, db *gorm.DB) {
	var activePlayers int64
	db.Model(&models.AnimaProfile{}).Where("cloud_enabled = ?", true).Distinct("owner_user_id").Count(&activePlayers)

	var activeAnimas int64
	db.Model(&models.AnimaProfile{}).Where("cloud_enabled = ?", true).Count(&activeAnimas)

	var aloneAnimas int64
	db.Model(&models.AnimaParkPresence{}).Where("is_alone = ?", true).Count(&aloneAnimas)

	var currentMeetings int64
	db.Model(&models.AnimaSocialEncounter{}).Count(&currentMeetings)

	// For simplicity, social events = recent encounters or 0
	socialEvents := int(currentMeetings / 2)

	atmosphere := "seedGarden"
	if activeAnimas > 5 {
		atmosphere = "livingPark"
	}
	if activeAnimas > 20 {
		atmosphere = "luminousForest"
	}

	c.JSON(http.StatusOK, gin.H{
		"activePlayers":        activePlayers,
		"activeAnimas":         activeAnimas,
		"aloneAnimas":          aloneAnimas,
		"currentMeetings":      currentMeetings,
		"socialLearningEvents": socialEvents,
		"atmosphereLevel":      atmosphere,
	})
}