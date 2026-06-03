package services

import (
	"context"

	"cgwm/battle/internal/nexus_tribunal/models"
	"gorm.io/gorm"
)

// LiveTrialStoryEngine resolves player actions against progression/failure rules (authoritative).
type LiveTrialStoryEngine struct {
	db *gorm.DB
}

func NewLiveTrialStoryEngine(db *gorm.DB) *LiveTrialStoryEngine { return &LiveTrialStoryEngine{db: db} }

func (e *LiveTrialStoryEngine) ResolvePlayerAction(ctx context.Context, narrativeCaseID uint, sceneID, actionType string, payload map[string]any) (map[string]any, error) {
	// Full impl: load rules, match, apply effects, decide advance or penalty, record event.
	return map[string]any{
		"success": true,
		"resultType": "minor_contradiction",
		"narrativeResult": "Action resolue par le moteur (stub).",
		"sceneAdvanced": false,
		"effects": map[string]int{"defenseScoreDelta": 5},
	}, nil
}

func (e *LiveTrialStoryEngine) MatchProgressionRule(sceneID, action string, evidenceID, statementID *string) (*models.TribunalProgressionRule, bool) {
	return nil, false
}

func (e *LiveTrialStoryEngine) ApplySuccessEffects(rule *models.TribunalProgressionRule) map[string]int { return map[string]int{} }
func (e *LiveTrialStoryEngine) ApplyFailureEffects(f *models.TribunalFailureRule) map[string]int { return map[string]int{} }
func (e *LiveTrialStoryEngine) ShouldAdvanceScene() bool { return false }
func (e *LiveTrialStoryEngine) BuildNarrativeResult() string { return "" }
func (e *LiveTrialStoryEngine) BuildHintIfNeeded() string { return "" }
