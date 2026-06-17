package admin

import (
	"mime/multipart"
	"net/http"
	"strconv"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
)

type generateImagePromptsRequest struct {
	OnlyMissing     bool   `json:"onlyMissing"`
	ForceRegenerate bool   `json:"forceRegenerate"`
	SceneCount      int    `json:"sceneCount"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	APIKey          string `json:"apiKey"`
}

func (s *Server) generateRolePlayQuestImagePromptsAdminAPI(c *gin.Context) {
	questID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || questID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest id"})
		return
	}
	var req generateImagePromptsRequest
	_ = c.ShouldBindJSON(&req)
	if req.SceneCount <= 0 {
		req.SceneCount = 3
	}
	visual := service.NewRolePlayQuestVisualService(s.db)
	result, err := visual.GenerateImagePromptsForQuest(c.Request.Context(), uint(questID), service.GenerateImagePromptsInput{
		OnlyMissing:     req.OnlyMissing,
		ForceRegenerate: req.ForceRegenerate,
		SceneCount:      req.SceneCount,
		Provider:        req.Provider,
		Model:           req.Model,
		APIKey:          req.APIKey,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	questData, _ := s.loadRolePlayQuestAdminItem(c, uint(questID))
	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"questId":        questID,
		"updatedPrompts": result.UpdatedPrompts,
		"createdScenes":  result.CreatedScenes,
		"skipped":        result.Skipped,
		"quest":          questData,
	})
}

func (s *Server) generateRolePlaySceneImagePromptAdminAPI(c *gin.Context) {
	questID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	sceneID, _ := strconv.ParseUint(c.Param("sceneId"), 10, 64)
	if questID == 0 || sceneID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ids"})
		return
	}
	var req generateImagePromptsRequest
	_ = c.ShouldBindJSON(&req)
	visual := service.NewRolePlayQuestVisualService(s.db)
	result, err := visual.GenerateImagePromptForScene(c.Request.Context(), uint(questID), uint(sceneID), service.GenerateImagePromptsInput{
		ForceRegenerate: req.ForceRegenerate,
		Provider:        req.Provider,
		Model:           req.Model,
		APIKey:          req.APIKey,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"questId":        questID,
		"sceneId":        sceneID,
		"updatedPrompts": result.UpdatedPrompts,
		"skipped":        result.Skipped,
	})
}

func (s *Server) startRolePlayImagePromptJobAdminAPI(c *gin.Context) {
	var req service.StartImagePromptJobInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	jobService := service.NewRolePlayImagePromptJobService(s.db)
	status, err := jobService.StartJob(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, status)
}

func (s *Server) getRolePlayImagePromptJobAdminAPI(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("jobId"), 10, 64)
	if err != nil || jobID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	status, err := service.NewRolePlayImagePromptJobService(s.db).GetJob(c.Request.Context(), uint(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (s *Server) listRolePlayImagePromptJobsAdminAPI(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	jobs, err := service.NewRolePlayImagePromptJobService(s.db).ListJobs(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

func (s *Server) cancelRolePlayImagePromptJobAdminAPI(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("jobId"), 10, 64)
	if err != nil || jobID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	if err := service.NewRolePlayImagePromptJobService(s.db).CancelJob(c.Request.Context(), uint(jobID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status, _ := service.NewRolePlayImagePromptJobService(s.db).GetJob(c.Request.Context(), uint(jobID))
	c.JSON(http.StatusOK, status)
}

func (s *Server) uploadRolePlaySceneImageAdminAPI(c *gin.Context) {
	questID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || questID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest id"})
		return
	}
	sceneID, err := strconv.ParseUint(c.Param("sceneId"), 10, 64)
	if err != nil || sceneID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scene id"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.RolePlaySceneMaxUploadBytes())
	if err := c.Request.ParseMultipartForm(service.RolePlaySceneMaxUploadBytes()); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "upload too large or invalid multipart form"})
		return
	}

	alt := c.PostForm("alt")
	visual := service.NewRolePlayQuestVisualService(s.db)
	files := collectRolePlayUploadFiles(c)
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image file is required"})
		return
	}

	images := make([]*models.RolePlayQuestSceneImage, 0, len(files))
	for index, file := range files {
		uploadAlt := alt
		if index > 0 {
			uploadAlt = ""
		}
		reader, openErr := file.Open()
		if openErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": openErr.Error()})
			return
		}
		image, uploadErr := visual.SaveSceneImageUpload(
			c.Request.Context(),
			uint(questID),
			uint(sceneID),
			file.Filename,
			reader,
			uploadAlt,
		)
		reader.Close()
		if uploadErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": uploadErr.Error()})
			return
		}
		images = append(images, image)
	}

	if len(images) == 1 {
		c.JSON(http.StatusCreated, gin.H{"image": images[0], "images": images, "count": 1})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"images": images, "count": len(images)})
}

func collectRolePlayUploadFiles(c *gin.Context) []*multipart.FileHeader {
	headers := make([]*multipart.FileHeader, 0)
	if form := c.Request.MultipartForm; form != nil {
		if multi, ok := form.File["images[]"]; ok {
			headers = append(headers, multi...)
		}
		if multi, ok := form.File["images"]; ok {
			headers = append(headers, multi...)
		}
	}
	if single, err := c.FormFile("image"); err == nil && single != nil {
		headers = append(headers, single)
	}
	out := make([]*multipart.FileHeader, 0, len(headers))
	for _, header := range headers {
		if header != nil {
			out = append(out, header)
		}
	}
	return out
}

func (s *Server) loadRolePlayQuestAdminItem(c *gin.Context, questID uint) (any, error) {
	data, err := s.rolePlayQuestsAdminData(c.Request.Context())
	if err != nil {
		return nil, err
	}
	for _, quest := range data.Quests {
		if quest.Id == questID {
			return quest, nil
		}
	}
	return nil, nil
}