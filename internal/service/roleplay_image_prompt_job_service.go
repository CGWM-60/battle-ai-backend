package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"cgwm/battle/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	ImagePromptJobPending     = "pending"
	ImagePromptJobRunning     = "running"
	ImagePromptJobCompleted   = "completed"
	ImagePromptJobFailed      = "failed"
	ImagePromptJobCancelled   = "cancelled"
	ImagePromptJobInterrupted = "interrupted"

	ImagePromptJobItemPending   = "pending"
	ImagePromptJobItemRunning   = "running"
	ImagePromptJobItemCompleted = "completed"
	ImagePromptJobItemFailed    = "failed"
	ImagePromptJobItemSkipped   = "skipped"
)

type ImagePromptJobError struct {
	QuestID uint   `json:"questId"`
	Title   string `json:"title"`
	Error   string `json:"error"`
}

type StartImagePromptJobInput struct {
	Scope           string `json:"scope"`
	QuestIDs        []uint `json:"questIds"`
	OnlyMissing     bool   `json:"onlyMissing"`
	ForceRegenerate bool   `json:"forceRegenerate"`
	SceneMode       string `json:"sceneMode"`
	SceneCount      int    `json:"sceneCount"`
	BatchSize       int    `json:"batchSize"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	APIKey          string `json:"apiKey"`
}

type ImagePromptJobStatus struct {
	JobID             uint                   `json:"jobId"`
	Status            string                 `json:"status"`
	TotalQuests       int                    `json:"totalQuests"`
	ProcessedQuests   int                    `json:"processedQuests"`
	UpdatedQuests     int                    `json:"updatedQuests"`
	CreatedScenes     int                    `json:"createdScenes"`
	UpdatedPrompts    int                    `json:"updatedPrompts"`
	FailedQuests      int                    `json:"failedQuests"`
	Percent           int                    `json:"percent"`
	CurrentQuestID    *uint                  `json:"currentQuestId,omitempty"`
	CurrentQuestTitle string                 `json:"currentQuestTitle,omitempty"`
	StartedAt         *time.Time             `json:"startedAt,omitempty"`
	FinishedAt        *time.Time             `json:"finishedAt,omitempty"`
	Errors            []ImagePromptJobError  `json:"errors"`
}

type RolePlayImagePromptJobService struct {
	db      *gorm.DB
	visual  *RolePlayQuestVisualService
	running sync.Map
}

func NewRolePlayImagePromptJobService(db *gorm.DB) *RolePlayImagePromptJobService {
	return &RolePlayImagePromptJobService{
		db:     db,
		visual: NewRolePlayQuestVisualService(db),
	}
}

func (s *RolePlayImagePromptJobService) RecoverInterruptedJobs(ctx context.Context) error {
	now := time.Now()
	return s.db.WithContext(ctx).
		Model(&models.RolePlayImagePromptJob{}).
		Where("status = ?", ImagePromptJobRunning).
		Updates(map[string]any{
			"status":      ImagePromptJobInterrupted,
			"finished_at": now,
		}).Error
}

func (s *RolePlayImagePromptJobService) StartJob(ctx context.Context, input StartImagePromptJobInput) (*ImagePromptJobStatus, error) {
	if strings.TrimSpace(input.SceneMode) == "" {
		input.SceneMode = SceneModePerChapter
	}
	if input.BatchSize <= 0 {
		input.BatchSize = 5
	}
	if input.BatchSize > 20 {
		input.BatchSize = 20
	}

	questIDs, err := s.resolveQuestIDs(ctx, input)
	if err != nil {
		return nil, err
	}
	if len(questIDs) == 0 {
		return nil, fmt.Errorf("no quests to process")
	}

	questIDsJSON, _ := json.Marshal(questIDs)
	job := models.RolePlayImagePromptJob{
		Status:          ImagePromptJobPending,
		Scope:           defaultString(input.Scope, "all"),
		TotalQuests:     len(questIDs),
		OnlyMissing:     input.OnlyMissing,
		ForceRegenerate: input.ForceRegenerate,
		SceneCount:      input.SceneCount,
		BatchSize:       input.BatchSize,
		Provider:        strings.TrimSpace(input.Provider),
		Model:           strings.TrimSpace(input.Model),
		QuestIDs:        datatypes.JSON(questIDsJSON),
		Errors:          datatypes.JSON([]byte("[]")),
	}
	if err := s.db.WithContext(ctx).Create(&job).Error; err != nil {
		return nil, err
	}

	items := make([]models.RolePlayImagePromptJobItem, 0, len(questIDs))
	for _, questID := range questIDs {
		var quest models.RolePlayQuestTemplate
		title := fmt.Sprintf("Quest %d", questID)
		if err := s.db.WithContext(ctx).Select("id", "title").First(&quest, questID).Error; err == nil {
			title = quest.Title
		}
		items = append(items, models.RolePlayImagePromptJobItem{
			JobID:      job.Id,
			QuestID:    questID,
			QuestTitle: title,
			Status:     ImagePromptJobItemPending,
		})
	}
	if err := s.db.WithContext(ctx).Create(&items).Error; err != nil {
		return nil, err
	}

	status := s.jobToStatus(job, nil)
	go s.runJob(job.Id, input)
	return &status, nil
}

func (s *RolePlayImagePromptJobService) resolveQuestIDs(ctx context.Context, input StartImagePromptJobInput) ([]uint, error) {
	scope := strings.ToLower(strings.TrimSpace(input.Scope))
	if scope == "ids" && len(input.QuestIDs) > 0 {
		return input.QuestIDs, nil
	}
	var quests []models.RolePlayQuestTemplate
	if err := s.db.WithContext(ctx).Order("id ASC").Select("id").Find(&quests).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(quests))
	for _, quest := range quests {
		ids = append(ids, quest.Id)
	}
	return ids, nil
}

func (s *RolePlayImagePromptJobService) runJob(jobID uint, input StartImagePromptJobInput) {
	ctx := context.Background()
	cancel := make(chan struct{})
	s.running.Store(jobID, cancel)
	defer s.running.Delete(jobID)

	now := time.Now()
	_ = s.db.WithContext(ctx).Model(&models.RolePlayImagePromptJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":     ImagePromptJobRunning,
		"started_at": now,
	}).Error

	var items []models.RolePlayImagePromptJobItem
	_ = s.db.WithContext(ctx).Where("job_id = ?", jobID).Order("id ASC").Find(&items).Error

	genInput := GenerateImagePromptsInput{
		OnlyMissing:     input.OnlyMissing,
		ForceRegenerate: input.ForceRegenerate,
		SceneMode:       input.SceneMode,
		SceneCount:      input.SceneCount,
		Provider:        input.Provider,
		Model:           input.Model,
		APIKey:          input.APIKey,
	}

	errors := make([]ImagePromptJobError, 0)
	for i := 0; i < len(items); i += input.BatchSize {
		select {
		case <-cancel:
			_ = s.finishJob(ctx, jobID, ImagePromptJobCancelled, errors)
			return
		default:
		}

		end := i + input.BatchSize
		if end > len(items) {
			end = len(items)
		}
		batch := items[i:end]
		for _, item := range batch {
			select {
			case <-cancel:
				_ = s.finishJob(ctx, jobID, ImagePromptJobCancelled, errors)
				return
			default:
			}

			_ = s.db.WithContext(ctx).Model(&models.RolePlayImagePromptJob{}).Where("id = ?", jobID).Updates(map[string]any{
				"current_quest_id":    item.QuestID,
				"current_quest_title": item.QuestTitle,
			}).Error
			_ = s.db.WithContext(ctx).Model(&models.RolePlayImagePromptJobItem{}).Where("id = ?", item.Id).
				Update("status", ImagePromptJobItemRunning).Error

			result, err := s.visual.GenerateImagePromptsForQuest(ctx, item.QuestID, genInput)
			itemUpdates := map[string]any{"status": ImagePromptJobItemCompleted}
			if err != nil {
				itemUpdates["status"] = ImagePromptJobItemFailed
				itemUpdates["error"] = err.Error()
				errors = append(errors, ImagePromptJobError{QuestID: item.QuestID, Title: item.QuestTitle, Error: err.Error()})
				_ = s.db.WithContext(ctx).Model(&models.RolePlayImagePromptJob{}).Where("id = ?", jobID).
					UpdateColumn("failed_quests", gorm.Expr("failed_quests + 1")).Error
			} else if result.Skipped {
				itemUpdates["status"] = ImagePromptJobItemSkipped
			} else {
				itemUpdates["created_scenes"] = result.CreatedScenes
				itemUpdates["updated_prompts"] = result.UpdatedPrompts
				updates := map[string]any{}
				if result.CreatedScenes > 0 {
					updates["created_scenes"] = gorm.Expr("created_scenes + ?", result.CreatedScenes)
				}
				if result.UpdatedPrompts > 0 {
					updates["updated_prompts"] = gorm.Expr("updated_prompts + ?", result.UpdatedPrompts)
				}
				if result.UpdatedPrompts > 0 || result.CreatedScenes > 0 {
					updates["updated_quests"] = gorm.Expr("updated_quests + 1")
				}
				if len(updates) > 0 {
					_ = s.db.WithContext(ctx).Model(&models.RolePlayImagePromptJob{}).Where("id = ?", jobID).Updates(updates).Error
				}
			}
			_ = s.db.WithContext(ctx).Model(&models.RolePlayImagePromptJobItem{}).Where("id = ?", item.Id).Updates(itemUpdates).Error
			_ = s.db.WithContext(ctx).Model(&models.RolePlayImagePromptJob{}).Where("id = ?", jobID).
				UpdateColumn("processed_quests", gorm.Expr("processed_quests + 1")).Error
		}
		if end < len(items) {
			time.Sleep(150 * time.Millisecond)
		}
	}

	_ = s.finishJob(ctx, jobID, ImagePromptJobCompleted, errors)
}

func (s *RolePlayImagePromptJobService) finishJob(ctx context.Context, jobID uint, status string, errors []ImagePromptJobError) error {
	errorsJSON, _ := json.Marshal(errors)
	now := time.Now()
	return s.db.WithContext(ctx).Model(&models.RolePlayImagePromptJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":              status,
		"finished_at":         now,
		"errors":              datatypes.JSON(errorsJSON),
		"current_quest_id":    nil,
		"current_quest_title": "",
	}).Error
}

func (s *RolePlayImagePromptJobService) GetJob(ctx context.Context, jobID uint) (*ImagePromptJobStatus, error) {
	var job models.RolePlayImagePromptJob
	if err := s.db.WithContext(ctx).First(&job, jobID).Error; err != nil {
		return nil, err
	}
	errors := decodeJobErrors(job.Errors)
	status := s.jobToStatus(job, errors)
	return &status, nil
}

func (s *RolePlayImagePromptJobService) ListJobs(ctx context.Context, limit int) ([]ImagePromptJobStatus, error) {
	if limit <= 0 {
		limit = 20
	}
	var jobs []models.RolePlayImagePromptJob
	if err := s.db.WithContext(ctx).Order("id DESC").Limit(limit).Find(&jobs).Error; err != nil {
		return nil, err
	}
	out := make([]ImagePromptJobStatus, 0, len(jobs))
	for _, job := range jobs {
		out = append(out, s.jobToStatus(job, decodeJobErrors(job.Errors)))
	}
	return out, nil
}

func (s *RolePlayImagePromptJobService) CancelJob(ctx context.Context, jobID uint) error {
	if ch, ok := s.running.Load(jobID); ok {
		if cancel, ok := ch.(chan struct{}); ok {
			close(cancel)
		}
	}
	var job models.RolePlayImagePromptJob
	if err := s.db.WithContext(ctx).First(&job, jobID).Error; err != nil {
		return err
	}
	if job.Status != ImagePromptJobRunning && job.Status != ImagePromptJobPending {
		return fmt.Errorf("job is not cancellable")
	}
	now := time.Now()
	return s.db.WithContext(ctx).Model(&job).Updates(map[string]any{
		"status":      ImagePromptJobCancelled,
		"finished_at": now,
	}).Error
}

func (s *RolePlayImagePromptJobService) jobToStatus(job models.RolePlayImagePromptJob, errors []ImagePromptJobError) ImagePromptJobStatus {
	percent := 0
	if job.TotalQuests > 0 {
		percent = int(float64(job.ProcessedQuests) / float64(job.TotalQuests) * 100)
	}
	if errors == nil {
		errors = []ImagePromptJobError{}
	}
	return ImagePromptJobStatus{
		JobID:             job.Id,
		Status:            job.Status,
		TotalQuests:       job.TotalQuests,
		ProcessedQuests:   job.ProcessedQuests,
		UpdatedQuests:     job.UpdatedQuests,
		CreatedScenes:     job.CreatedScenes,
		UpdatedPrompts:    job.UpdatedPrompts,
		FailedQuests:      job.FailedQuests,
		Percent:           percent,
		CurrentQuestID:    job.CurrentQuestID,
		CurrentQuestTitle: job.CurrentQuestTitle,
		StartedAt:         job.StartedAt,
		FinishedAt:        job.FinishedAt,
		Errors:            errors,
	}
}

func decodeJobErrors(raw datatypes.JSON) []ImagePromptJobError {
	if len(raw) == 0 {
		return []ImagePromptJobError{}
	}
	var errors []ImagePromptJobError
	if err := json.Unmarshal(raw, &errors); err != nil {
		return []ImagePromptJobError{}
	}
	return errors
}

