package router

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ConstructionJobDTO is the canonical queue item returned by construction endpoints.
type ConstructionJobDTO struct {
	ID          string     `json:"id"`
	BuildingKey string     `json:"buildingKey"`
	BuildingID  string     `json:"buildingId,omitempty"`
	FromLevel   int        `json:"fromLevel"`
	TargetLevel int        `json:"targetLevel"`
	Type        string     `json:"type"`   // construct | upgrade
	Status      string     `json:"status"` // queued | in_progress | completed | cancelled
	QueuedAt    *time.Time `json:"queuedAt,omitempty"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// ConstructionQueueResponse is shared by all 6 endpoints to keep Flutter sync simple.
type ConstructionQueueResponse struct {
	Jobs        []ConstructionJobDTO `json:"jobs"`
	MaxTeams    int                  `json:"maxTeams"`
	ActiveTeams int                  `json:"activeTeams"`
	ServerNow   time.Time            `json:"serverNow"`
	Message     string               `json:"message,omitempty"`
	PlayerSave  map[string]any       `json:"playerSave"`
}

// StartConstructionRequest starts a new building construction.
type StartConstructionRequest struct {
	BuildingKey string `json:"buildingKey" binding:"required"`
	TargetLevel int    `json:"targetLevel" binding:"required,min=1"`
	PositionX   *int   `json:"positionX,omitempty"`
	PositionY   *int   `json:"positionY,omitempty"`
}

// ConstructionActionPathParams represent :id path usage for upgrade/speedup/cancel/complete.
type ConstructionActionPathParams struct {
	ID string `uri:"id" binding:"required"`
}

// registerConstructionContractRoutes exposes the 6 construction endpoints with exact JSON shape.
func registerConstructionContractRoutes(private *gin.RouterGroup, database *gorm.DB, world *service.WorldGameService) {
	private.GET("/construction/queue", func(c *gin.Context) {
		save, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		jobs, err := decodeConstructionQueue(save.ConstructionQueueJSON)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid construction queue data"})
			return
		}
		buildings, _ := decodeBuildings(save.BuildingsJSON)
		activeEffects, _ := decodeObject(save.ActiveEffectsJSON)
		c.JSON(http.StatusOK, buildQueueResponse(save, jobs, buildings, activeEffects, "queue loaded"))
	})

	private.POST("/construction/start", func(c *gin.Context) {
		var input StartConstructionRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start construction payload"})
			return
		}
		input.BuildingKey = normalizeBuildingKey(input.BuildingKey)

		var response ConstructionQueueResponse
		err := database.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			save, jobs, buildings, activeEffects, err := loadSaveBundleForUpdate(c, tx, world)
			if err != nil {
				return err
			}

			duration, err := resolveConstructionDuration(tx, input.BuildingKey, input.TargetLevel, save.Satisfaction, activeEffects)
			if err != nil {
				return err
			}

			now := time.Now().UTC()
			maxTeams := maxTeamsFromEffects(activeEffects)
			activeTeams := countActiveJobs(jobs)
			status := "queued"
			var startedAt *time.Time
			var completedAt *time.Time
			if activeTeams < maxTeams {
				status = "in_progress"
				startedAt = ptrTime(now)
				end := now.Add(duration)
				completedAt = ptrTime(end)
			}

			fromLevel := input.TargetLevel - 1
			if fromLevel < 0 {
				fromLevel = 0
			}

			jobID := fmt.Sprintf("c_%d", now.UnixNano())
			job := ConstructionJobDTO{
				ID:          jobID,
				BuildingKey: strings.TrimSpace(input.BuildingKey),
				FromLevel:   fromLevel,
				TargetLevel: input.TargetLevel,
				Type:        "construct",
				Status:      status,
				QueuedAt:    ptrTime(now),
				StartedAt:   startedAt,
				CompletedAt: completedAt,
			}
			jobs = append(jobs, job)

			buildings = upsertBuildingForStart(buildings, input, job)

			if err := persistSaveBundle(tx, save, jobs, buildings); err != nil {
				return err
			}

			response = buildQueueResponse(save, jobs, buildings, activeEffects, "construction started")
			return nil
		})
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})

	private.POST("/construction/:id/upgrade", func(c *gin.Context) {
		var path ConstructionActionPathParams
		if err := c.ShouldBindUri(&path); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid construction id"})
			return
		}

		var response ConstructionQueueResponse
		err := database.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			save, jobs, buildings, activeEffects, err := loadSaveBundleForUpdate(c, tx, world)
			if err != nil {
				return err
			}

			idx := findJobIndex(jobs, path.ID)
			if idx < 0 {
				return gorm.ErrRecordNotFound
			}

			job := jobs[idx]
			if job.Status == "cancelled" {
				return fmt.Errorf("cannot upgrade a cancelled job")
			}

			now := time.Now().UTC()
			job.Type = "upgrade"
			job.Status = "in_progress"
			job.FromLevel = job.TargetLevel
			job.TargetLevel = job.TargetLevel + 1
			duration, err := resolveConstructionDuration(tx, job.BuildingKey, job.TargetLevel, save.Satisfaction, activeEffects)
			if err != nil {
				return err
			}
			job.StartedAt = ptrTime(now)
			end := now.Add(duration)
			job.CompletedAt = ptrTime(end)
			jobs[idx] = job

			buildings = upsertBuildingForJobProgress(buildings, job)

			if err := persistSaveBundle(tx, save, jobs, buildings); err != nil {
				return err
			}
			response = buildQueueResponse(save, jobs, buildings, activeEffects, "construction upgraded")
			return nil
		})
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})

	private.POST("/construction/:id/speedup", func(c *gin.Context) {
		var path ConstructionActionPathParams
		if err := c.ShouldBindUri(&path); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid construction id"})
			return
		}

		var response ConstructionQueueResponse
		err := database.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			save, jobs, buildings, activeEffects, err := loadSaveBundleForUpdate(c, tx, world)
			if err != nil {
				return err
			}
			idx := findJobIndex(jobs, path.ID)
			if idx < 0 {
				return gorm.ErrRecordNotFound
			}
			now := time.Now().UTC()
			job := jobs[idx]
			if strings.EqualFold(strings.TrimSpace(job.Status), "cancelled") {
				return fmt.Errorf("cannot speed up a cancelled construction")
			}
			if strings.EqualFold(strings.TrimSpace(job.Status), "completed") {
				return fmt.Errorf("construction already completed")
			}
			if job.CompletedAt == nil {
				return fmt.Errorf("construction cannot be speed up before it starts")
			}

			remaining := job.CompletedAt.Sub(now)
			if remaining <= 0 {
				return fmt.Errorf("construction is already ready to complete")
			}
			gemCost := calculateSpeedUpGemCost(remaining)
			if save.Gems < int64(gemCost) {
				return fmt.Errorf("insufficient gems: required %d, available %d", gemCost, save.Gems)
			}
			save.Gems -= int64(gemCost)

			jobs[idx].Status = "completed"
			jobs[idx].CompletedAt = ptrTime(now)

			if err := persistSaveBundle(tx, save, jobs, buildings); err != nil {
				return err
			}
			response = buildQueueResponse(save, jobs, buildings, activeEffects, fmt.Sprintf("construction speedup applied (-%d gems)", gemCost))
			return nil
		})
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})

	private.POST("/construction/:id/cancel", func(c *gin.Context) {
		var path ConstructionActionPathParams
		if err := c.ShouldBindUri(&path); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid construction id"})
			return
		}

		var response ConstructionQueueResponse
		err := database.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			save, jobs, buildings, activeEffects, err := loadSaveBundleForUpdate(c, tx, world)
			if err != nil {
				return err
			}
			idx := findJobIndex(jobs, path.ID)
			if idx < 0 {
				return gorm.ErrRecordNotFound
			}
			cancelledJob := jobs[idx]
			buildings = resetBuildingOnCancel(buildings, cancelledJob)
			jobs = removeJob(jobs, idx)
			jobs, err = reconcileConstructionQueueSchedule(tx, jobs, save, activeEffects)
			if err != nil {
				return err
			}

			if err := persistSaveBundle(tx, save, jobs, buildings); err != nil {
				return err
			}
			response = buildQueueResponse(save, jobs, buildings, activeEffects, "construction cancelled")
			return nil
		})
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})

	private.POST("/construction/:id/complete", func(c *gin.Context) {
		var path ConstructionActionPathParams
		if err := c.ShouldBindUri(&path); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid construction id"})
			return
		}

		var response ConstructionQueueResponse
		err := database.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			save, jobs, buildings, activeEffects, err := loadSaveBundleForUpdate(c, tx, world)
			if err != nil {
				return err
			}
			idx := findJobIndex(jobs, path.ID)
			if idx < 0 {
				return gorm.ErrRecordNotFound
			}

			job := jobs[idx]
			now := time.Now().UTC()
			if job.CompletedAt == nil || job.CompletedAt.After(now) {
				return fmt.Errorf("construction is not ready to complete")
			}

			buildings = applyCompletedBuilding(buildings, job, now)
			jobs = removeJob(jobs, idx)
			jobs, err = reconcileConstructionQueueSchedule(tx, jobs, save, activeEffects)
			if err != nil {
				return err
			}

			if err := persistSaveBundle(tx, save, jobs, buildings); err != nil {
				return err
			}
			response = buildQueueResponse(save, jobs, buildings, activeEffects, "construction completed")
			return nil
		})
		if err != nil {
			writeWorldResponse(c, nil, err)
			return
		}
		c.JSON(http.StatusOK, response)
	})
}

func loadSaveBundleForUpdate(c *gin.Context, tx *gorm.DB, world *service.WorldGameService) (*models.PlayerSave, []ConstructionJobDTO, []map[string]any, map[string]any, error) {
	baseSave, err := world.EnsurePlayerSave(c.Request.Context(), currentUserID(c))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	var save models.PlayerSave
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", baseSave.Id).First(&save).Error; err != nil {
		return nil, nil, nil, nil, err
	}
	jobs, err := decodeConstructionQueue(save.ConstructionQueueJSON)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	buildings, err := decodeBuildings(save.BuildingsJSON)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	activeEffects, err := decodeObject(save.ActiveEffectsJSON)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return &save, jobs, buildings, activeEffects, nil
}

func resolveConstructionDuration(tx *gorm.DB, buildingKey string, targetLevel int, satisfaction int, activeEffects map[string]any) (time.Duration, error) {
	key := normalizeBuildingKey(buildingKey)
	if key == "" {
		return 0, fmt.Errorf("buildingKey is required")
	}
	if targetLevel < 1 {
		return 0, fmt.Errorf("targetLevel must be greater than zero")
	}

	var def models.BuildingDefinition
	err := tx.Where("`key` = ? AND is_active = ?", key, true).First(&def).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("unknown or inactive buildingKey %q: refresh /api/v1/buildings/catalog and use the returned key", key)
		}
		return 0, err
	}
	if def.MaxLevel > 0 && targetLevel > def.MaxLevel {
		return 0, fmt.Errorf("targetLevel %d exceeds max level %d for building %s", targetLevel, def.MaxLevel, key)
	}

	return calculateConstructionDuration(key, targetLevel, satisfaction, activeEffects), nil
}

func calculateConstructionDuration(buildingKey string, targetLevel int, satisfaction int, activeEffects map[string]any) time.Duration {
	minutes := float64(baseConstructionMinutesForLevel(targetLevel))
	minutes *= buildingConstructionMultiplier(buildingKey)
	minutes *= satisfactionConstructionMultiplier(satisfaction)
	minutes *= activeConstructionMultiplier(activeEffects)

	roundedMinutes := int(math.Round(minutes))
	if roundedMinutes < 1 {
		roundedMinutes = 1
	}
	duration := time.Duration(roundedMinutes) * time.Minute
	if duration < time.Minute {
		return time.Minute
	}
	maxDuration := 30 * 24 * time.Hour
	if duration > maxDuration {
		return maxDuration
	}
	return duration
}

func baseConstructionMinutesForLevel(level int) int {
	switch {
	case level <= 1:
		return 5
	case level == 2:
		return 15
	case level == 3:
		return 30
	case level == 4:
		return 60
	case level == 5:
		return 120
	case level <= 10:
		return interpolateMinutes(level, 6, 10, 4*60, 12*60)
	case level <= 15:
		return interpolateMinutes(level, 11, 15, 12*60, 24*60)
	case level <= 20:
		return interpolateMinutes(level, 16, 20, 24*60, 2*24*60)
	case level <= 25:
		return interpolateMinutes(level, 21, 25, 2*24*60, 4*24*60)
	default:
		if level > 30 {
			level = 30
		}
		return interpolateMinutes(level, 26, 30, 4*24*60, 30*24*60)
	}
}

func interpolateMinutes(level int, firstLevel int, lastLevel int, firstMinutes int, lastMinutes int) int {
	if level <= firstLevel {
		return firstMinutes
	}
	if level >= lastLevel {
		return lastMinutes
	}
	ratio := float64(level-firstLevel) / float64(lastLevel-firstLevel)
	return int(math.Round(float64(firstMinutes) + ratio*float64(lastMinutes-firstMinutes)))
}

func buildingConstructionMultiplier(buildingKey string) float64 {
	switch normalizeBuildingKey(buildingKey) {
	case "habitation":
		return 0.85
	case "solar_park":
		return 1
	case "vertical_farm":
		return 1.10
	case "research_center":
		return 1.20
	case "ai_center":
		return 1.35
	default:
		return 1
	}
}

func satisfactionConstructionMultiplier(satisfaction int) float64 {
	switch {
	case satisfaction >= 90:
		return 0.95
	case satisfaction >= 75:
		return 1
	case satisfaction >= 60:
		return 1.08
	case satisfaction >= 40:
		return 1.18
	case satisfaction >= 20:
		return 1.35
	default:
		return 1.60
	}
}

func activeConstructionMultiplier(activeEffects map[string]any) float64 {
	if len(activeEffects) == 0 {
		return 1
	}
	multiplier := 1.0
	for _, key := range []string{"constructionDurationMultiplier", "constructionTimeMultiplier", "weatherConstructionMultiplier"} {
		if value, ok := floatFromAny(activeEffects[key]); ok && value > 0 {
			multiplier *= value
		}
	}
	for _, key := range []string{"constructionDurationBonusPercent", "constructionTimeReductionPercent"} {
		if value, ok := floatFromAny(activeEffects[key]); ok {
			multiplier *= clampFloat(1-(value/100), 0.1, 3)
		}
	}
	for _, key := range []string{"constructionDurationMalusPercent", "constructionTimePenaltyPercent"} {
		if value, ok := floatFromAny(activeEffects[key]); ok {
			multiplier *= clampFloat(1+(value/100), 0.1, 3)
		}
	}
	for _, key := range []string{"constructionSpeedMultiplier", "constructionSpeed"} {
		if value, ok := floatFromAny(activeEffects[key]); ok && value > 0 {
			multiplier /= value
		}
	}
	if value, ok := floatFromAny(activeEffects["constructionSpeedPercent"]); ok {
		multiplier /= clampFloat(1+(value/100), 0.1, 10)
	}
	return clampFloat(multiplier, 0.1, 10)
}

func normalizeBuildingKey(buildingKey string) string {
	key := strings.ToLower(strings.TrimSpace(buildingKey))
	key = strings.ReplaceAll(key, "-", "_")
	key = strings.ReplaceAll(key, " ", "_")
	switch key {
	case "housing", "house", "home", "residence", "residential", "habitation_basic":
		return "habitation"
	case "solar", "solarpark", "solar_panel", "solar_panels", "parc_solaire":
		return "solar_park"
	case "farm", "verticalfarm", "verticale_farm", "ferme_verticale":
		return "vertical_farm"
	case "recherche", "research", "researchcenter", "centre_recherche", "centre_de_recherche", "laboratory", "lab":
		return "research_center"
	case "ai", "aicenter", "centre_ia", "centre_ai":
		return "ai_center"
	case "defense", "defence", "defensegrid", "grille_defense":
		return "defense_grid"
	default:
		return key
	}
}

func decodeConstructionQueue(raw datatypes.JSON) ([]ConstructionJobDTO, error) {
	if len(raw) == 0 {
		return []ConstructionJobDTO{}, nil
	}
	trim := strings.TrimSpace(string(raw))
	if trim == "" || trim == "{}" || trim == "null" {
		return []ConstructionJobDTO{}, nil
	}

	var jobs []ConstructionJobDTO
	if err := json.Unmarshal(raw, &jobs); err == nil {
		return jobs, nil
	}

	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, err
	}
	if value, ok := wrapped["jobs"]; ok {
		if err := json.Unmarshal(value, &jobs); err != nil {
			return nil, err
		}
		return jobs, nil
	}
	if value, ok := wrapped["queue"]; ok {
		if err := json.Unmarshal(value, &jobs); err != nil {
			return nil, err
		}
		return jobs, nil
	}
	return []ConstructionJobDTO{}, nil
}

func decodeBuildings(raw datatypes.JSON) ([]map[string]any, error) {
	if len(raw) == 0 {
		return []map[string]any{}, nil
	}
	trim := strings.TrimSpace(string(raw))
	if trim == "" || trim == "{}" || trim == "null" {
		return []map[string]any{}, nil
	}

	var buildings []map[string]any
	if err := json.Unmarshal(raw, &buildings); err == nil {
		return buildings, nil
	}

	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, err
	}
	if value, ok := wrapped["buildings"]; ok {
		if err := json.Unmarshal(value, &buildings); err != nil {
			return nil, err
		}
		return buildings, nil
	}
	return []map[string]any{}, nil
}

func decodeObject(raw datatypes.JSON) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	trim := strings.TrimSpace(string(raw))
	if trim == "" || trim == "{}" || trim == "null" {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return map[string]any{}, nil
	}
	return out, nil
}

func persistSaveBundle(tx *gorm.DB, save *models.PlayerSave, jobs []ConstructionJobDTO, buildings []map[string]any) error {
	now := time.Now().UTC()
	jobsRaw, err := json.Marshal(jobs)
	if err != nil {
		return err
	}
	buildingsRaw, err := json.Marshal(buildings)
	if err != nil {
		return err
	}

	updates := map[string]any{
		"construction_queue_json": datatypes.JSON(jobsRaw),
		"buildings_json":          datatypes.JSON(buildingsRaw),
		"gems":                    save.Gems,
		"version":                 save.Version + 1,
		"last_synced_at":          &now,
	}

	if err := tx.Model(&models.PlayerSave{}).Where("id = ?", save.Id).Updates(updates).Error; err != nil {
		return err
	}
	if err := syncPlayerBuildingsTable(tx, save, buildings); err != nil {
		return err
	}
	save.Version = save.Version + 1
	save.LastSyncedAt = &now
	save.ConstructionQueueJSON = datatypes.JSON(jobsRaw)
	save.BuildingsJSON = datatypes.JSON(buildingsRaw)
	return nil
}

func syncPlayerBuildingsTable(tx *gorm.DB, save *models.PlayerSave, buildings []map[string]any) error {
	var existing []models.PlayerBuilding
	if err := tx.Where("player_id = ?", save.PlayerID).Find(&existing).Error; err != nil {
		return err
	}

	byKey := map[string]models.PlayerBuilding{}
	for _, item := range existing {
		key := strings.TrimSpace(item.BuildingKey)
		if key == "" {
			continue
		}
		if _, already := byKey[key]; !already {
			byKey[key] = item
		}
	}

	seen := map[string]bool{}
	for _, raw := range buildings {
		key := strings.TrimSpace(fmt.Sprintf("%v", raw["buildingKey"]))
		if key == "" {
			continue
		}
		seen[key] = true

		level, ok := intFromAny(raw["level"])
		if !ok || level < 0 {
			level = 0
		}
		positionX, ok := intFromAny(raw["positionX"])
		if !ok {
			positionX = 0
		}
		positionY, ok := intFromAny(raw["positionY"])
		if !ok {
			positionY = 0
		}
		state := strings.TrimSpace(fmt.Sprintf("%v", raw["state"]))
		if state == "" {
			state = "placed"
		}

		startedAt := timeFromAny(raw["startedAt"])
		completedAt := timeFromAny(raw["completedAt"])
		lastCollectedAt := timeFromAny(raw["lastCollectedAt"])

		if current, ok := byKey[key]; ok {
			updates := map[string]any{
				"level":             level,
				"position_x":        positionX,
				"position_y":        positionY,
				"state":             state,
				"started_at":        startedAt,
				"completed_at":      completedAt,
				"last_collected_at": lastCollectedAt,
			}
			if err := tx.Model(&models.PlayerBuilding{}).Where("id = ?", current.Id).Updates(updates).Error; err != nil {
				return err
			}
			continue
		}

		item := models.PlayerBuilding{
			PlayerID:        save.PlayerID,
			BuildingKey:     key,
			Level:           level,
			PositionX:       positionX,
			PositionY:       positionY,
			State:           state,
			StartedAt:       startedAt,
			CompletedAt:     completedAt,
			LastCollectedAt: lastCollectedAt,
		}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
	}

	if len(seen) == 0 {
		if err := tx.Where("player_id = ?", save.PlayerID).Delete(&models.PlayerBuilding{}).Error; err != nil {
			return err
		}
		return nil
	}

	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	if err := tx.Where("player_id = ? AND building_key NOT IN ?", save.PlayerID, keys).Delete(&models.PlayerBuilding{}).Error; err != nil {
		return err
	}
	return nil
}

func buildQueueResponse(save *models.PlayerSave, jobs []ConstructionJobDTO, buildings []map[string]any, activeEffects map[string]any, message string) ConstructionQueueResponse {
	if activeEffects == nil {
		activeEffects = map[string]any{}
	}
	return ConstructionQueueResponse{
		Jobs:        jobs,
		MaxTeams:    maxTeamsFromEffects(activeEffects),
		ActiveTeams: countActiveJobs(jobs),
		ServerNow:   time.Now().UTC(),
		Message:     message,
		PlayerSave: map[string]any{
			"id":                save.Id,
			"playerId":          save.PlayerID,
			"constructionQueue": jobs,
			"buildings":         buildings,
			"activeEffects":     activeEffects,
			"version":           save.Version,
		},
	}
}

func maxTeamsFromEffects(activeEffects map[string]any) int {
	if value, ok := intFromAny(activeEffects["maxTeams"]); ok && value > 0 {
		return value
	}
	return 1
}

func countActiveJobs(jobs []ConstructionJobDTO) int {
	count := 0
	for _, job := range jobs {
		if job.Status != "cancelled" && job.Status != "completed" {
			count++
		}
	}
	return count
}

func findJobIndex(jobs []ConstructionJobDTO, id string) int {
	for i, job := range jobs {
		if job.ID == id {
			return i
		}
	}
	return -1
}

func removeJob(jobs []ConstructionJobDTO, index int) []ConstructionJobDTO {
	if index < 0 || index >= len(jobs) {
		return jobs
	}
	return append(jobs[:index], jobs[index+1:]...)
}

func upsertBuildingForStart(buildings []map[string]any, input StartConstructionRequest, job ConstructionJobDTO) []map[string]any {
	for i := range buildings {
		if strings.TrimSpace(fmt.Sprintf("%v", buildings[i]["buildingKey"])) == job.BuildingKey {
			buildings[i]["state"] = "constructing"
			buildings[i]["startedAt"] = job.StartedAt
			buildings[i]["completedAt"] = job.CompletedAt
			return buildings
		}
	}

	positionX := 0
	positionY := 0
	if input.PositionX != nil {
		positionX = *input.PositionX
	}
	if input.PositionY != nil {
		positionY = *input.PositionY
	}

	return append(buildings, map[string]any{
		"id":          job.ID,
		"buildingKey": job.BuildingKey,
		"level":       job.FromLevel,
		"positionX":   positionX,
		"positionY":   positionY,
		"state":       "constructing",
		"startedAt":   job.StartedAt,
		"completedAt": job.CompletedAt,
	})
}

func upsertBuildingForJobProgress(buildings []map[string]any, job ConstructionJobDTO) []map[string]any {
	for i := range buildings {
		if strings.TrimSpace(fmt.Sprintf("%v", buildings[i]["buildingKey"])) == job.BuildingKey {
			buildings[i]["state"] = "upgrading"
			buildings[i]["startedAt"] = job.StartedAt
			buildings[i]["completedAt"] = job.CompletedAt
			return buildings
		}
	}
	return buildings
}

func resetBuildingOnCancel(buildings []map[string]any, job ConstructionJobDTO) []map[string]any {
	for i := 0; i < len(buildings); i++ {
		if strings.TrimSpace(fmt.Sprintf("%v", buildings[i]["buildingKey"])) != job.BuildingKey {
			continue
		}

		if strings.EqualFold(strings.TrimSpace(job.Type), "construct") && job.FromLevel <= 0 {
			return append(buildings[:i], buildings[i+1:]...)
		}

		buildings[i]["state"] = "placed"
		buildings[i]["startedAt"] = nil
		buildings[i]["completedAt"] = nil
		buildings[i]["level"] = job.FromLevel
		return buildings
	}
	return buildings
}

func reconcileConstructionQueueSchedule(tx *gorm.DB, jobs []ConstructionJobDTO, save *models.PlayerSave, activeEffects map[string]any) ([]ConstructionJobDTO, error) {
	if len(jobs) == 0 {
		return jobs, nil
	}

	now := time.Now().UTC()
	maxTeams := maxTeamsFromEffects(activeEffects)
	activeTeams := 0

	for i := range jobs {
		status := strings.ToLower(strings.TrimSpace(jobs[i].Status))
		switch status {
		case "completed", "cancelled":
			continue
		case "active":
			status = "in_progress"
		}

		if jobs[i].CompletedAt != nil && !jobs[i].CompletedAt.After(now) {
			jobs[i].Status = "completed"
			continue
		}

		if status == "in_progress" {
			if jobs[i].StartedAt == nil {
				jobs[i].StartedAt = ptrTime(now)
			}
			if jobs[i].CompletedAt == nil {
				duration, err := resolveConstructionDuration(tx, jobs[i].BuildingKey, jobs[i].TargetLevel, save.Satisfaction, activeEffects)
				if err != nil {
					return nil, err
				}
				end := jobs[i].StartedAt.UTC().Add(duration)
				jobs[i].CompletedAt = ptrTime(end)
			}
			jobs[i].Status = "in_progress"
			activeTeams++
			continue
		}

		jobs[i].Status = "queued"
		jobs[i].StartedAt = nil
		jobs[i].CompletedAt = nil
	}

	for i := range jobs {
		if activeTeams >= maxTeams {
			break
		}
		if strings.ToLower(strings.TrimSpace(jobs[i].Status)) != "queued" {
			continue
		}
		duration, err := resolveConstructionDuration(tx, jobs[i].BuildingKey, jobs[i].TargetLevel, save.Satisfaction, activeEffects)
		if err != nil {
			return nil, err
		}
		jobs[i].Status = "in_progress"
		jobs[i].StartedAt = ptrTime(now)
		end := now.Add(duration)
		jobs[i].CompletedAt = ptrTime(end)
		activeTeams++
	}

	return jobs, nil
}

func calculateSpeedUpGemCost(remaining time.Duration) int {
	if remaining <= 15*time.Minute {
		return 1
	}
	if remaining <= time.Hour {
		return 5
	}
	if remaining <= 6*time.Hour {
		return 15
	}
	if remaining <= 12*time.Hour {
		return 30
	}
	if remaining <= 24*time.Hour {
		return 60
	}
	if remaining <= 72*time.Hour {
		return 120
	}
	return 200
}

func applyCompletedBuilding(buildings []map[string]any, job ConstructionJobDTO, now time.Time) []map[string]any {
	for i := range buildings {
		if strings.TrimSpace(fmt.Sprintf("%v", buildings[i]["buildingKey"])) == job.BuildingKey {
			buildings[i]["level"] = job.TargetLevel
			buildings[i]["state"] = "placed"
			buildings[i]["startedAt"] = nil
			buildings[i]["completedAt"] = nil
			buildings[i]["lastCollectedAt"] = now
			return buildings
		}
	}
	return append(buildings, map[string]any{
		"id":              job.ID,
		"buildingKey":     job.BuildingKey,
		"level":           job.TargetLevel,
		"positionX":       0,
		"positionY":       0,
		"state":           "placed",
		"startedAt":       nil,
		"completedAt":     nil,
		"lastCollectedAt": now,
	})
}

func intFromAny(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return int(n), true
		}
	case string:
		var n int
		if _, err := fmt.Sscanf(strings.TrimSpace(v), "%d", &n); err == nil {
			return n, true
		}
	}
	return 0, false
}

func floatFromAny(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case json.Number:
		if n, err := v.Float64(); err == nil {
			return n, true
		}
	case string:
		var n float64
		if _, err := fmt.Sscanf(strings.TrimSpace(v), "%f", &n); err == nil {
			return n, true
		}
	}
	return 0, false
}

func clampFloat(value float64, min float64, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func timeFromAny(value any) *time.Time {
	if value == nil {
		return nil
	}
	if typed, ok := value.(time.Time); ok {
		v := typed.UTC()
		return &v
	}
	if typed, ok := value.(*time.Time); ok {
		if typed == nil {
			return nil
		}
		v := typed.UTC()
		return &v
	}
	text := strings.TrimSpace(fmt.Sprintf("%v", value))
	if text == "" || text == "<nil>" || text == "null" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return nil
	}
	v := parsed.UTC()
	return &v
}

func ptrTime(value time.Time) *time.Time {
	v := value.UTC()
	return &v
}
