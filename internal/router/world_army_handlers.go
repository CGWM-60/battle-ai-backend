package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func registerArmyRoutes(private *gin.RouterGroup, database *gorm.DB, world *service.WorldGameService) {
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
			writeWorldResponse(c, nil, badRequestError("INVALID_PAYLOAD", "Payload d'affectation invalide.", nil))
			return
		}

		if input.ConflictID == 0 {
			writeWorldResponse(c, nil, badRequestError("INVALID_CONFLICT", "Conflit invalide.", nil))
			return
		}
		if len(input.SoldierIDs) == 0 {
			writeWorldResponse(c, nil, badRequestError("NO_SOLDIERS_SELECTED", "Aucun soldat sélectionné.", nil))
			return
		}
		strategy := strings.ToLower(strings.TrimSpace(input.Strategy))
		if !isAllowedConflictStrategy(strategy) {
			writeWorldResponse(c, nil, badRequestError("INVALID_STRATEGY", "Stratégie invalide.", map[string]any{"allowed": []string{"defensive", "balanced", "aggressive", "support", "extraction"}}))
			return
		}

		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}

		var conflict models.Conflict
		err = database.WithContext(c.Request.Context()).Where("id = ? AND world_id = ?", input.ConflictID, save.WorldID).First(&conflict).Error
		if err != nil {
			writeWorldResponse(c, nil, notFoundError("CONFLICT_NOT_FOUND", "Conflit introuvable.", map[string]any{"conflictId": input.ConflictID}))
			return
		}
		if strings.ToLower(strings.TrimSpace(conflict.Status)) != "active" {
			writeWorldResponse(c, nil, conflictError("CONFLICT_NOT_ACTIVE", "Le conflit n'est pas actif.", map[string]any{"status": conflict.Status}))
			return
		}

		minBarracksLevel := requiredBarracksLevelForConflict(conflict)
		playerBarracksLevel := playerBuildingLevelFromSaveJSON(save.BuildingsJSON, "barracks")
		if playerBarracksLevel < minBarracksLevel {
			writeWorldResponse(c, nil, conflictError("BARRACKS_LEVEL_REQUIRED", "Niveau de caserne insuffisant pour ce conflit.", map[string]any{"required": minBarracksLevel, "current": playerBarracksLevel}))
			return
		}

		var units []models.ArmyUnit
		err = database.WithContext(c.Request.Context()).Where("player_id = ? AND id IN ?", save.PlayerID, input.SoldierIDs).Find(&units).Error
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		if len(units) != len(input.SoldierIDs) {
			writeWorldResponse(c, nil, forbiddenError("SOLDIER_OWNERSHIP_MISMATCH", "Certains soldats n'appartiennent pas au joueur.", nil))
			return
		}
		for _, unit := range units {
			if strings.ToLower(strings.TrimSpace(unit.Status)) != "available" {
				writeWorldResponse(c, nil, conflictError("SOLDIER_NOT_AVAILABLE", "Un ou plusieurs soldats ne sont pas disponibles.", map[string]any{"soldierId": unit.Id, "status": unit.Status}))
				return
			}
		}

		estimatedFood := int64(len(units) * 2)
		estimatedEnergy := int64(len(units))
		estimatedCredits := int64(len(units) * 2)
		if save.Food < estimatedFood || save.Energy < estimatedEnergy || save.Credits < estimatedCredits {
			writeWorldResponse(c, nil, conflictError("NOT_ENOUGH_RESOURCES", "Ressources insuffisantes pour intervenir.", map[string]any{
				"missingFood":    maxInt64(0, estimatedFood-save.Food),
				"missingEnergy":  maxInt64(0, estimatedEnergy-save.Energy),
				"missingCredits": maxInt64(0, estimatedCredits-save.Credits),
			}))
			return
		}

		err = database.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.ArmyUnit{}).
				Where("player_id = ? AND id IN ?", save.PlayerID, input.SoldierIDs).
				Updates(map[string]any{"status": "assigned", "assigned_conflict_id": conflict.Id}).Error; err != nil {
				return err
			}
			return tx.Create(&models.ConflictAction{ConflictID: conflict.Id, PlayerID: save.PlayerID, ActionType: "intervene", Payload: []byte(`{}`)}).Error
		})
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}

		writeWorldResponse(c, gin.H{
			"interventionId":  strconv.FormatUint(uint64(conflict.Id), 10) + "-" + strconv.FormatUint(uint64(time.Now().UTC().Unix()), 10),
			"status":          "in_progress",
			"startsAt":        time.Now().UTC(),
			"finishesAt":      time.Now().UTC().Add(30 * time.Minute),
			"successChance":   0.62,
			"estimatedLosses": gin.H{"units": maxInt(1, len(units)/5)},
			"lockedUnits":     input.SoldierIDs,
			"cost":            gin.H{"food": estimatedFood, "energy": estimatedEnergy, "credits": estimatedCredits},
			"expectedRewards": gin.H{"credits": 400, "reputation": 2},
			"strategy":        strategy,
		}, nil)
	})

	private.POST("/army/recall", func(c *gin.Context) {
		var input struct {
			SoldierIDs []uint `json:"soldierIds"`
		}
		if err := bindPayload(c, &input); err != nil {
			writeWorldResponse(c, nil, badRequestError("INVALID_PAYLOAD", "Payload de rappel invalide.", nil))
			return
		}
		if len(input.SoldierIDs) == 0 {
			writeWorldResponse(c, nil, badRequestError("NO_SOLDIERS_SELECTED", "Aucun soldat sélectionné.", nil))
			return
		}
		playerID := currentUserID(c)
		err := database.WithContext(c.Request.Context()).Model(&models.ArmyUnit{}).
			Where("player_id = ? AND id IN ? AND status IN ?", playerID, input.SoldierIDs, []string{"assigned", "injured", "exhausted", "returning"}).
			Updates(map[string]any{"status": "returning", "assigned_conflict_id": nil}).Error
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		writeWorldResponse(c, gin.H{"recalled": true, "soldierCount": len(input.SoldierIDs), "status": "returning"}, nil)
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

func isAllowedConflictStrategy(strategy string) bool {
	switch strategy {
	case "defensive", "balanced", "aggressive", "support", "extraction":
		return true
	default:
		return false
	}
}

func requiredBarracksLevelForConflict(conflict models.Conflict) int {
	if strings.Contains(strings.ToLower(conflict.Title+" "+conflict.Description), "mond") {
		return 22
	}
	switch {
	case conflict.Intensity >= 85:
		return 22
	case conflict.Intensity >= 70:
		return 15
	case conflict.Intensity >= 50:
		return 8
	case conflict.Intensity >= 30:
		return 3
	default:
		return 1
	}
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func playerBuildingLevelFromSaveJSON(raw datatypes.JSON, key string) int {
	if len(raw) == 0 {
		return 0
	}
	items := make([]map[string]any, 0)
	if err := json.Unmarshal(raw, &items); err != nil {
		return 0
	}
	target := strings.ToLower(strings.TrimSpace(key))
	for _, item := range items {
		buildingKey := strings.ToLower(strings.TrimSpace(fmt.Sprint(item["buildingKey"])))
		if buildingKey != target {
			continue
		}
		switch level := item["level"].(type) {
		case float64:
			return int(level)
		case int:
			return level
		}
	}
	return 0
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
