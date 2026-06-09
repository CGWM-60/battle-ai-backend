package handlers

import (
	"io"
	"net/http"
	"strconv"

	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/services"

	"github.com/gin-gonic/gin"
)

// ContentHandler provides REST CRUD + asset upload for major content items (buildings first).
// Used by admin Next.js for tables + forms, and by Flutter for catalog.
// All images uploaded here are stored on server disk and served statically (configure router to /nexus-assets/content/* -> content/assets).
// The system is kept open: add similar methods for units/research.

type ContentHandler struct {
	contentSvc *services.ContentService
}

func NewContentHandler(contentSvc *services.ContentService) *ContentHandler {
	return &ContentHandler{contentSvc: contentSvc}
}

// === Buildings (table + CRUD for admin) ===

func (h *ContentHandler) ListBuildings(c *gin.Context) {
	published := c.Query("published") == "true"
	list, err := h.contentSvc.ListBuildings(published)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"buildings": list, "count": len(list)})
}

func (h *ContentHandler) GetBuilding(c *gin.Context) {
	id := c.Param("contentId")
	b, err := h.contentSvc.GetBuilding(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "building not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"building": b})
}

func (h *ContentHandler) CreateOrUpdateBuilding(c *gin.Context) {
	var def models.BuildingDefinition
	if err := c.ShouldBindJSON(&def); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if def.ContentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contentId required"})
		return
	}
	if err := h.contentSvc.CreateOrUpdateBuilding(&def); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "contentId": def.ContentID})
}

func (h *ContentHandler) DeleteBuilding(c *gin.Context) {
	id := c.Param("contentId")
	if err := h.contentSvc.DeleteBuilding(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// UploadAsset for a building (or other domain).
// Form: file (multipart), contentId, domain="building", tier="1"|"2"|"3"|"4" (optional)
func (h *ContentHandler) UploadAsset(c *gin.Context) {
	domain := c.PostForm("domain")
	contentID := c.PostForm("contentId")
	tier := c.PostForm("tier")
	if domain == "" || contentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain and contentId required"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read file failed"})
		return
	}

	savedName, err := h.contentSvc.UploadAsset(domain, contentID, tier, header.Filename, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":        true,
		"savedAs":   savedName,
		"urlHint":   "/nexus-assets/content/" + domain + "s/" + savedName, // configure static serving
	})
}

// === Player constructions (depends on buildings) ===

func (h *ContentHandler) ListPlayerBuildings(c *gin.Context) {
	pidStr := c.Param("profileId")
	pid, _ := strconv.ParseUint(pidStr, 10, 64)
	list, err := h.contentSvc.ListPlayerBuildings(uint(pid))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"buildings": list})
}

func (h *ContentHandler) StartConstruction(c *gin.Context) {
	// body: {profileId, contentId, targetLevel}
	var body struct {
		ProfileID   uint   `json:"profileId"`
		ContentID   string `json:"contentId"`
		TargetLevel int    `json:"targetLevel"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pb, err := h.contentSvc.StartConstruction(body.ProfileID, body.ContentID, body.TargetLevel)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"playerBuilding": pb, "note": "Construction started. Complete on tick or poll /complete."})
}

// Poll or call on load to complete ready constructions (server truth).
func (h *ContentHandler) CompleteReadyConstructions(c *gin.Context) {
	pidStr := c.Param("profileId")
	pid, _ := strconv.ParseUint(pidStr, 10, 64)

	list, _ := h.contentSvc.ListPlayerBuildings(uint(pid))
	completed := []uint{}
	for i := range list {
		pb := &list[i]
		done, _ := h.contentSvc.CompleteConstructionIfReady(pb)
		if done {
			completed = append(completed, pb.ID)
		}
	}
	c.JSON(http.StatusOK, gin.H{"completed": completed, "remaining": len(list) - len(completed)})
}
