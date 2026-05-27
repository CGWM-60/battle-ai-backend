package router

import (
	"encoding/json"
	"fmt"
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

		var response ConstructionQueueResponse
		err := database.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			save, jobs, buildings, activeEffects, err := loadSaveBundleForUpdate(c, tx, world)
			if err != nil {
				return err
			}

			duration, err := resolveConstructionDuration(tx, input.BuildingKey)
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

			duration, err := resolveConstructionDuration(tx, job.BuildingKey)
			if err != nil {
				return err
			}

			now := time.Now().UTC()
			job.Type = "upgrade"
			job.Status = "in_progress"
			job.FromLevel = job.TargetLevel
			job.TargetLevel = job.TargetLevel + 1
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
			jobs[idx].Status = "completed"
			jobs[idx].CompletedAt = ptrTime(now)

			if err := persistSaveBundle(tx, save, jobs, buildings); err != nil {
				return err
			}
			response = buildQueueResponse(save, jobs, buildings, activeEffects, "construction speedup applied")
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
			jobs[idx].Status = "cancelled"

			buildings = resetBuildingOnCancel(buildings, jobs[idx])

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

func resolveConstructionDuration(tx *gorm.DB, buildingKey string) (time.Duration, error) {
	key := strings.TrimSpace(buildingKey)
	if key == "" {
		return 0, fmt.Errorf("buildingKey is required")
	}

	var def models.BuildingDefinition
	err := tx.Where("key = ? AND is_active = ?", key, true).First(&def).Error
	if err != nil {
		return 0, err
	}

	baseCost, err := decodeObject(def.BaseCostJSON)
	if err != nil {
		return 0, fmt.Errorf("invalid base cost for building %s", key)
	}

	minutes, ok := intFromAny(baseCost["durationMinutes"])
	if !ok {
		minutes, ok = intFromAny(baseCost["buildTimeMinutes"])
	}
	if !ok {
		minutes, ok = intFromAny(baseCost["timeMinutes"])
	}
	if !ok || minutes <= 0 {
		return 0, fmt.Errorf("construction duration is not configured for building %s", key)
	}

	return time.Duration(minutes) * time.Minute, nil
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
	for i := range buildings {
		if strings.TrimSpace(fmt.Sprintf("%v", buildings[i]["buildingKey"])) == job.BuildingKey {
			buildings[i]["state"] = "placed"
			buildings[i]["startedAt"] = nil
			buildings[i]["completedAt"] = nil
		}
	}
	return buildings
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
