package services

import (
	"context"

	"cgwm/battle/internal/nexus_tribunal/models"
	"gorm.io/gorm"
)

// NarrativeCaseService handles scenarized narrative cases (acts/scenes/progression).
type NarrativeCaseService struct {
	db *gorm.DB
}

func NewNarrativeCaseService(db *gorm.DB) *NarrativeCaseService {
	return &NarrativeCaseService{db: db}
}

func (s *NarrativeCaseService) CreateNarrativeCaseFromGeneratedCase(ctx context.Context, gen *models.TribunalGeneratedCase) (*models.TribunalNarrativeCase, error) {
	// Stub: in full parse ScenesJSON etc and persist acts/scenes/rules
	nc := &models.TribunalNarrativeCase{
		GeneratedCaseID: &gen.ID,
		Title:           gen.Title,
		Synopsis:        gen.Summary,
		RealTruth:       gen.RealTruth,
		PublicTruth:     gen.PublicTruth,
		FinalReveal:     gen.FinalReveal,
		DifficultyLevel: gen.Level,
		StoryTone:       gen.Tone,
		Status:          "ready",
	}
	if err := s.db.WithContext(ctx).Create(nc).Error; err != nil {
		return nil, err
	}
	return nc, nil
}

func (s *NarrativeCaseService) GetCurrentScene(ctx context.Context, narrativeCaseID uint) (*models.TribunalScene, error) {
	var sc models.TribunalScene
	if err := s.db.WithContext(ctx).Where("narrative_case_id = ? AND status = ?", narrativeCaseID, "active").First(&sc).Error; err != nil {
		return nil, err
	}
	return &sc, nil
}

func (s *NarrativeCaseService) BuildSceneResponse(ctx context.Context, scene *models.TribunalScene) (map[string]any, error) {
	// Stub full response shape per corrective
	return map[string]any{
		"sceneId": scene.SceneID,
		"title":   scene.Title,
		"objective": scene.Objective,
		"sceneType": scene.SceneType,
		"narrativeText": scene.NarrativeText,
		"allowedActions": []string{"press", "present_evidence", "objection"},
	}, nil
}

// Add other required: AdvanceScene, Unlock*, RecordStoryEvent, GetSceneState...
