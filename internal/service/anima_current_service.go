package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	cgwmmodels "cgwm/battle/internal/cgwm/models"

	"gorm.io/gorm"
)

type AnimaCurrentView struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Avatar     string         `json:"avatar,omitempty"`
	Appearance map[string]any `json:"appearance,omitempty"`
	Mood       string         `json:"mood,omitempty"`
	Energy     int            `json:"energy,omitempty"`
	Affection  int            `json:"affection,omitempty"`
	Status     string         `json:"status,omitempty"`
}

type AnimaCurrentResult struct {
	Exists bool              `json:"exists"`
	Alive  bool              `json:"alive"`
	Anima  *AnimaCurrentView `json:"anima"`
}

type AnimaCurrentService struct {
	db *gorm.DB
}

func NewAnimaCurrentService(db *gorm.DB) *AnimaCurrentService {
	return &AnimaCurrentService{db: db}
}

func (s *AnimaCurrentService) GetCurrent(ctx context.Context, userID uint) (*AnimaCurrentResult, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("anima current service unavailable")
	}
	if userID == 0 {
		return nil, fmt.Errorf("user id is required")
	}

	ownerID := fmt.Sprintf("%d", userID)
	var profile cgwmmodels.AnimaProfile
	err := s.db.WithContext(ctx).
		Where("owner_user_id = ?", ownerID).
		Order("updated_at DESC").
		First(&profile).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &AnimaCurrentResult{Exists: false, Alive: false, Anima: nil}, nil
		}
		return nil, err
	}

	alive, appearance, energy, affection, status := resolveLivingState(s.db.WithContext(ctx), profile)

	view := &AnimaCurrentView{
		ID:         firstNonEmpty(profile.ID, profile.PublicAnimaID),
		Name:       strings.TrimSpace(profile.Name),
		Avatar:     strings.TrimSpace(profile.AvatarPreview),
		Appearance: appearance,
		Mood:       strings.TrimSpace(profile.Mood),
		Energy:     energy,
		Affection:  affection,
		Status:     status,
	}
	if view.Name == "" {
		view.Name = "Amina"
	}

	return &AnimaCurrentResult{
		Exists: true,
		Alive:  alive,
		Anima:  view,
	}, nil
}

func resolveLivingState(db *gorm.DB, profile cgwmmodels.AnimaProfile) (
	alive bool,
	appearance map[string]any,
	energy int,
	affection int,
	status string,
) {
	appearance = map[string]any{
		"stage": profile.Stage,
		"mood":  profile.Mood,
		"style": "anima_module",
	}
	if strings.TrimSpace(profile.AvatarPreview) != "" {
		appearance["avatar"] = profile.AvatarPreview
	}

	var snapshot cgwmmodels.AnimaCloudSnapshot
	err := db.
		Where("anima_id = ? OR owner_user_id = ?", profile.ID, profile.OwnerUserID).
		Order("updated_at DESC").
		First(&snapshot).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return true, appearance, 80, 55, "active"
		}
		return false, appearance, 0, 0, "unknown"
	}

	payload := map[string]any{}
	if strings.TrimSpace(snapshot.SnapshotJSON) != "" {
		_ = json.Unmarshal([]byte(snapshot.SnapshotJSON), &payload)
	}

	if nested, ok := payload["anima"].(map[string]any); ok {
		payload = nested
	}

	if dead, ok := payload["isDead"].(bool); ok && dead {
		return false, mergeAppearance(appearance, payload), 0, 0, "inactive"
	}
	if dead, ok := payload["is_dead"].(bool); ok && dead {
		return false, mergeAppearance(appearance, payload), 0, 0, "inactive"
	}

	energy = animaIntFromAny(payload["energy"], 80)
	affection = animaIntFromAny(payload["affection"], 55)
	if genome, ok := payload["visualGenome"].(map[string]any); ok {
		appearance = mergeAppearance(appearance, genome)
	} else if genome, ok := payload["visual_genome"].(map[string]any); ok {
		appearance = mergeAppearance(appearance, genome)
	}
	appearance = mergeAppearance(appearance, payload)

	return true, appearance, energy, affection, "active"
}

func mergeAppearance(base map[string]any, extra map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range extra {
		out[key] = value
	}
	return out
}

func animaIntFromAny(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return fallback
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}