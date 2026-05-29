package router

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
)

func registerArmyRoutes(private *gin.RouterGroup, world *service.WorldGameService) {
	private.GET("/army/overview", func(c *gin.Context) {
		overview, err := world.ArmyOverview(c.Request.Context(), currentUserID(c))
		writeWorldResponse(c, overview, err)
	})

	private.GET("/army/units", func(c *gin.Context) {
		units, err := world.ListArmyUnits(c.Request.Context(), currentUserID(c), limitFromQuery(c))
		writeWorldResponse(c, gin.H{"units": units}, err)
	})

	private.POST("/army/train", func(c *gin.Context) {
		var input service.ArmyTrainInput
		if err := bindPayload(c, &input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid training payload"})
			return
		}
		result, err := world.TrainArmy(c.Request.Context(), currentUserID(c), input)
		writeWorldResponse(c, result, err)
	})

	private.POST("/army/assign-conflict", func(c *gin.Context) {
		var input struct {
			ConflictID uint   `json:"conflictId"`
			SoldierIDs []uint `json:"soldierIds"`
			Strategy   string `json:"strategy"`
		}
		if err := bindPayload(c, &input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid assign payload"})
			return
		}
		writeWorldResponse(c, gin.H{
			"assigned":     true,
			"conflictId":   input.ConflictID,
			"soldierCount": len(input.SoldierIDs),
			"strategy":     strings.ToLower(strings.TrimSpace(input.Strategy)),
		}, nil)
	})

	private.POST("/army/recall", func(c *gin.Context) {
		var input struct {
			SoldierIDs []uint `json:"soldierIds"`
		}
		if err := bindPayload(c, &input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recall payload"})
			return
		}
		writeWorldResponse(c, gin.H{"recalled": true, "soldierCount": len(input.SoldierIDs)}, nil)
	})

	private.GET("/army/reports", func(c *gin.Context) {
		overview, err := world.ArmyOverview(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		writeWorldResponse(c, gin.H{
			"reports": []gin.H{
				{"id": "army-daily", "type": "army_report", "title": "Rapport armée", "summary": "État des troupes et puissance.", "createdAt": time.Now().UTC(), "importance": "info", "metrics": overview},
			},
		}, nil)
	})
}

func registerWorldCompatibilityRoutes(private *gin.RouterGroup, world *service.WorldGameService) {
	private.GET("/world/overview", func(c *gin.Context) {
		state, err := world.PlayerState(c.Request.Context(), currentUserID(c))
		writeWorldResponse(c, state, err)
	})

	private.GET("/world/conflicts/:id/simulation", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conflict id"})
			return
		}
		strategy := strings.TrimSpace(strings.ToLower(c.DefaultQuery("strategy", "balanced")))
		if strategy == "" {
			strategy = "balanced"
		}
		writeWorldResponse(c, gin.H{
			"conflictId":         strconv.FormatUint(uint64(id), 10),
			"successChance":      0.62,
			"estimatedCost":      gin.H{"food": 120, "energy": 80, "credits": 150},
			"possibleLosses":     gin.H{"units": 4, "resources": 95},
			"estimatedDuration":  1800,
			"possibleGains":      gin.H{"credits": 400, "reputation": 2},
			"impactReputation":   2,
			"impactDiplomacy":    -1,
			"impactWorldTension": 3,
			"strategy":           strategy,
		}, nil)
	})

	private.GET("/buildings", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		writeWorldResponse(c, gin.H{"buildings": save.BuildingsJSON}, nil)
	})

	private.POST("/buildings/build", func(c *gin.Context) {
		var input struct {
			Type  string `json:"type"`
			Level int    `json:"level"`
		}
		if err := bindPayload(c, &input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid building payload"})
			return
		}
		if input.Level <= 0 {
			input.Level = 1
		}
		cost, err := world.CalculateBuildingUpgradeCost(input.Type, input.Level-1)
		writeWorldResponse(c, gin.H{"queued": true, "type": input.Type, "targetLevel": input.Level, "cost": cost}, err)
	})

	private.POST("/buildings/:id/upgrade", func(c *gin.Context) {
		var input struct {
			Type         string `json:"type"`
			CurrentLevel int    `json:"currentLevel"`
		}
		_ = bindPayload(c, &input)
		if input.CurrentLevel < 0 {
			input.CurrentLevel = 0
		}
		cost, err := world.CalculateBuildingUpgradeCost(input.Type, input.CurrentLevel)
		writeWorldResponse(c, gin.H{"buildingId": c.Param("id"), "upgrade": cost}, err)
	})

	private.GET("/buildings/:id", func(c *gin.Context) {
		writeWorldResponse(c, gin.H{"id": c.Param("id"), "maxLevel": 30}, nil)
	})
}
