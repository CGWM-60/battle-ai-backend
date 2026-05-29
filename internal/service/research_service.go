package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cgwm/battle/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ResearchCatalogPayload struct {
	Resources []models.ResourceDefinition     `json:"resources"`
	Trees     []models.ResearchTreeDefinition `json:"trees"`
	Progress  PlayerResearchProgress          `json:"progress"`
}

type PlayerResearchProgress struct {
	Nodes  map[string]ResearchNodeProgress `json:"nodes"`
	Active *ActiveResearchProgress         `json:"active,omitempty"`
}

type ResearchNodeProgress struct {
	Level       int        `json:"level"`
	Status      string     `json:"status"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

type ActiveResearchProgress struct {
	NodeKey     string    `json:"nodeKey"`
	TargetLevel int       `json:"targetLevel"`
	StartedAt   time.Time `json:"startedAt"`
	FinishAt    time.Time `json:"finishAt"`
}

type ResearchActionResult struct {
	Node     models.ResearchNodeDefinition `json:"node"`
	Progress PlayerResearchProgress        `json:"progress"`
}

func (s *WorldGameService) ResearchCatalog(ctx context.Context, playerID uint, buildingKey string) (ResearchCatalogPayload, error) {
	var resources []models.ResourceDefinition
	if err := s.db.WithContext(ctx).Where("is_active = ?", true).Order("sort_order ASC, id ASC").Find(&resources).Error; err != nil {
		return ResearchCatalogPayload{}, err
	}

	query := s.db.WithContext(ctx).Preload("Nodes", func(db *gorm.DB) *gorm.DB {
		return db.Where("is_active = ?", true).Order("sort_order ASC, id ASC")
	}).Where("is_active = ?", true)
	if aliases := normalizeResearchBuildingKeys(buildingKey); len(aliases) > 0 {
		query = query.Where("building_key IN ?", aliases)
	}
	var trees []models.ResearchTreeDefinition
	if err := query.Order("sort_order ASC, id ASC").Find(&trees).Error; err != nil {
		return ResearchCatalogPayload{}, err
	}

	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return ResearchCatalogPayload{}, err
	}
	progress := parseResearchProgress(save.ResearchJSON)
	return ResearchCatalogPayload{Resources: resources, Trees: trees, Progress: progress}, nil
}

func (s *WorldGameService) ResearchProgress(ctx context.Context, playerID uint) (PlayerResearchProgress, error) {
	save, err := s.EnsurePlayerSave(ctx, playerID)
	if err != nil {
		return PlayerResearchProgress{}, err
	}
	return parseResearchProgress(save.ResearchJSON), nil
}

func (s *WorldGameService) StartResearch(ctx context.Context, playerID uint, nodeKey string) (ResearchActionResult, error) {
	var result ResearchActionResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var node models.ResearchNodeDefinition
		if err := tx.Preload("ResearchTreeDefinition").Where("`key` = ? AND is_active = ?", nodeKey, true).First(&node).Error; err != nil {
			return err
		}
		if !node.ResearchTreeDefinition.IsActive {
			return errors.New("research tree is inactive")
		}

		save, err := playerSaveForResearch(tx, playerID)
		if err != nil {
			return err
		}
		progress := parseResearchProgress(save.ResearchJSON)
		if progress.Active != nil {
			return errors.New("another research is already active")
		}

		current := progress.Nodes[node.Key]
		targetLevel := current.Level + 1
		if node.MaxLevel > 0 && targetLevel > node.MaxLevel {
			return errors.New("research node already maxed")
		}
		if err := validateResearchParents(progress, node.ParentKeysJSON); err != nil {
			return err
		}

		duration := researchDuration(node.LevelProgressionJSON, targetLevel)
		now := time.Now()
		progress.Active = &ActiveResearchProgress{
			NodeKey:     node.Key,
			TargetLevel: targetLevel,
			StartedAt:   now,
			FinishAt:    now.Add(duration),
		}
		current.Status = "researching"
		current.StartedAt = &now
		progress.Nodes[node.Key] = current
		if err := updateResearchProgress(tx, save.Id, progress); err != nil {
			return err
		}
		result = ResearchActionResult{Node: node, Progress: progress}
		return nil
	})
	return result, err
}

func (s *WorldGameService) CompleteResearch(ctx context.Context, playerID uint, nodeKey string) (ResearchActionResult, error) {
	var result ResearchActionResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var node models.ResearchNodeDefinition
		if err := tx.Where("`key` = ?", nodeKey).First(&node).Error; err != nil {
			return err
		}
		save, err := playerSaveForResearch(tx, playerID)
		if err != nil {
			return err
		}
		progress := parseResearchProgress(save.ResearchJSON)
		if progress.Active == nil || progress.Active.NodeKey != node.Key {
			return errors.New("research node is not active")
		}
		now := time.Now()
		if progress.Active.FinishAt.After(now) {
			return fmt.Errorf("research is not finished")
		}
		current := progress.Nodes[node.Key]
		current.Level = progress.Active.TargetLevel
		current.Status = "completed"
		current.CompletedAt = &now
		progress.Nodes[node.Key] = current
		progress.Active = nil
		if err := updateResearchProgress(tx, save.Id, progress); err != nil {
			return err
		}
		result = ResearchActionResult{Node: node, Progress: progress}
		return nil
	})
	return result, err
}

func playerSaveForResearch(tx *gorm.DB, playerID uint) (*models.PlayerSave, error) {
	var save models.PlayerSave
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("player_id = ?", playerID).First(&save).Error; err != nil {
		return nil, err
	}
	return &save, nil
}

func parseResearchProgress(raw datatypes.JSON) PlayerResearchProgress {
	progress := PlayerResearchProgress{Nodes: map[string]ResearchNodeProgress{}}
	if len(raw) == 0 {
		return progress
	}
	if err := json.Unmarshal(raw, &progress); err != nil {
		return PlayerResearchProgress{Nodes: map[string]ResearchNodeProgress{}}
	}
	if progress.Nodes == nil {
		progress.Nodes = map[string]ResearchNodeProgress{}
	}
	return progress
}

func updateResearchProgress(tx *gorm.DB, saveID uint, progress PlayerResearchProgress) error {
	data, err := json.Marshal(progress)
	if err != nil {
		return err
	}
	return tx.Model(&models.PlayerSave{}).Where("id = ?", saveID).Update("research_json", datatypes.JSON(data)).Error
}

func validateResearchParents(progress PlayerResearchProgress, raw datatypes.JSON) error {
	var parents []string
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, &parents); err != nil {
		return err
	}
	for _, parent := range parents {
		if progress.Nodes[parent].Level <= 0 {
			return fmt.Errorf("missing prerequisite research: %s", parent)
		}
	}
	return nil
}

func researchDuration(raw datatypes.JSON, targetLevel int) time.Duration {
	var levels []researchSeedLevelDTO
	if len(raw) > 0 && json.Unmarshal(raw, &levels) == nil {
		for _, level := range levels {
			if level.Level == targetLevel && level.DurationMinutes > 0 {
				return time.Duration(level.DurationMinutes) * time.Minute
			}
		}
	}
	return time.Duration(targetLevel) * time.Hour
}

func normalizeResearchBuildingKeys(buildingKey string) []string {
	raw := strings.TrimSpace(strings.ToLower(buildingKey))
	if raw == "" {
		return nil
	}
	raw = strings.ReplaceAll(raw, "-", "_")

	canonical := raw
	switch raw {
	case "parc_solaire", "solar", "solarpark", "solar_panel", "solar_panels":
		canonical = "solar_park"
	case "ferme_verticale", "verticalfarm", "farm", "vertical_farm":
		canonical = "vertical_farm"
	case "recherche", "research", "researchcenter", "centre_recherche", "centre_de_recherche", "lab", "laboratory":
		canonical = "research_center"
	case "centre_ia", "aicenter", "centre_ai", "ai":
		canonical = "ai_center"
	}

	set := map[string]struct{}{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		set[value] = struct{}{}
	}

	add(raw)
	add(canonical)

	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	return out
}

type researchSeedLevelDTO struct {
	Level           int `json:"level"`
	DurationMinutes int `json:"durationMinutes"`
}
