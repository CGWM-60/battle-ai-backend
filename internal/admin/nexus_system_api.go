package admin

import (
	"net/http"
	"strings"

	nexusmodels "cgwm/battle/internal/nexus_game/models"
	serveraiservices "cgwm/battle/internal/nexus_game/server_ai/services"
	nexusservices "cgwm/battle/internal/nexus_game/services"

	"github.com/gin-gonic/gin"
)

func (s *Server) nexusSystemAPI(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return
	}
	ctx := c.Request.Context()
	contentSvc := nexusservices.NewContentService(s.db, nexusAdminContentAssetsDir())
	armySvc := nexusservices.NewArmyService(s.db, contentSvc)
	snapshot, err := armySvc.AdminSnapshot(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resources, _ := nexusservices.NewResourceService(s.db).PlayerSnapshot(ctx, firstProfileID(snapshot))
	jobs, _ := serveraiservices.NewService(s.db).ListJobs(ctx)
	c.JSON(http.StatusOK, gin.H{
		"snapshot":       snapshot,
		"resourceSample": resources,
		"jobs":           jobs,
		"routes": []gin.H{
			{"method": "GET", "path": "/admin/api/nexus-system", "label": "Console systeme"},
			{"method": "POST", "path": "/admin/api/nexus-system/resources/grant", "label": "Ajouter/debiter ressource joueur"},
			{"method": "POST", "path": "/admin/api/nexus-system/units/grant", "label": "Ajouter/debiter unite joueur"},
			{"method": "POST", "path": "/admin/api/nexus-system/ai/jobs/run-due", "label": "Executer jobs IA dus"},
		},
	})
}

func (s *Server) nexusSystemGrantResourceAPI(c *gin.Context) {
	var body struct {
		ProfileGamerID uint   `json:"profileGamerId"`
		ResourceCode   string `json:"resourceCode"`
		Amount         int64  `json:"amount"`
		Reason         string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	body.ResourceCode = strings.TrimSpace(body.ResourceCode)
	if body.ProfileGamerID == 0 || body.ResourceCode == "" || body.Amount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profileGamerId, resourceCode and non-zero amount are required"})
		return
	}
	rs := nexusservices.NewResourceService(s.db)
	if err := rs.EnsureInitialAllocation(c.Request.Context(), body.ProfileGamerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	source := "admin"
	if strings.TrimSpace(body.Reason) != "" {
		source = "admin:" + strings.TrimSpace(body.Reason)
	}
	resource, err := rs.ApplyResourceDelta(c.Request.Context(), body.ProfileGamerID, body.ResourceCode, body.Amount, "admin_grant", source)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"resource": resource})
}

func (s *Server) nexusSystemGrantUnitsAPI(c *gin.Context) {
	var body struct {
		ProfileGamerID uint   `json:"profileGamerId"`
		UnitCode       string `json:"unitCode"`
		Quantity       int    `json:"quantity"`
		Reason         string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.ProfileGamerID == 0 || strings.TrimSpace(body.UnitCode) == "" || body.Quantity == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profileGamerId, unitCode and non-zero quantity are required"})
		return
	}
	armySvc := nexusservices.NewArmyService(s.db, nexusservices.NewContentService(s.db, nexusAdminContentAssetsDir()))
	unit, err := armySvc.AdminGrantUnits(c.Request.Context(), body.ProfileGamerID, body.UnitCode, body.Quantity, body.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unit": unit})
}

func (s *Server) nexusSystemRunDueAIJobsAPI(c *gin.Context) {
	results, err := serveraiservices.NewService(s.db).RunDueJobs(c.Request.Context(), "admin_system", 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "results": results})
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

func nexusAdminContentAssetsDir() string {
	return env("NEXUS_ASSETS_BASE_DIR", "/nexus_game/assets") + "/content"
}

func firstProfileID(snapshot map[string]any) uint {
	players, ok := snapshot["players"].([]nexusmodels.ProfileGamer)
	if !ok || len(players) == 0 {
		return 0
	}
	return players[0].ID
}
