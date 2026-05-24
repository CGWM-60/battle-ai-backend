package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/repository"

	"gorm.io/datatypes"
)

type RolePlaySessionInput struct {
	TemplateID     uint
	Title          string
	Mode           string
	ScenarioPrompt string
	Snapshot       map[string]any
}

type RolePlayActionInput struct {
	AuthorName string
	Content    string
	Payload    map[string]any
}

type RolePlayService struct {
	roleplay *repository.RolePlayRepository
	quests   *repository.QuestRepository
}

func NewRolePlayService(roleplay *repository.RolePlayRepository, quests *repository.QuestRepository) *RolePlayService {
	return &RolePlayService{roleplay: roleplay, quests: quests}
}

func (s *RolePlayService) CreateSession(ctx context.Context, ownerID uint, input RolePlaySessionInput) (*models.RolePlaySession, error) {
	now := time.Now()
	title := input.Title
	prompt := input.ScenarioPrompt
	snapshotBytes, _ := json.Marshal(input.Snapshot)
	if len(snapshotBytes) == 0 || string(snapshotBytes) == "null" {
		snapshotBytes = []byte("{}")
	}
	var templateID *uint
	if input.TemplateID != 0 {
		template, err := s.quests.GetRolePlayQuestByID(ctx, input.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("roleplay quest not found")
		}
		templateID = &template.Id
		title = defaultString(title, template.Title)
		prompt = defaultString(prompt, template.Prompt)
	}
	if title == "" || prompt == "" {
		return nil, fmt.Errorf("title and scenarioPrompt are required")
	}
	session := &models.RolePlaySession{
		OwnerID:        ownerID,
		Mode:           defaultString(input.Mode, "solo"),
		Title:          title,
		Status:         constants.RolePlayStatusLive,
		ScenarioPrompt: prompt,
		CurrentScene:   "start",
		CurrentTurn:    0,
		Snapshot:       datatypes.JSON(snapshotBytes),
		StartedAt:      &now,
		LastActivityAt: &now,
	}
	if err := s.roleplay.CreateSession(ctx, session); err != nil {
		return nil, err
	}
	if templateID != nil {
		run := &models.RolePlayQuestRun{
			TemplateID:     templateID,
			UserID:         ownerID,
			SessionID:      &session.Id,
			Title:          title,
			Status:         constants.RolePlayStatusLive,
			CurrentStep:    1,
			TotalSteps:     0,
			Journal:        "",
			State:          datatypes.JSON([]byte("{}")),
			StartedAt:      &now,
			LastActivityAt: &now,
		}
		_ = s.roleplay.CreateQuestRun(ctx, run)
	}
	return session, nil
}

func (s *RolePlayService) ListSessions(ctx context.Context, ownerID uint, limit int) ([]models.RolePlaySession, error) {
	return s.roleplay.ListSessionsByOwner(ctx, ownerID, limit)
}

func (s *RolePlayService) GetSession(ctx context.Context, id uint, ownerID uint) (*models.RolePlaySession, error) {
	return s.roleplay.GetSessionOwnedByID(ctx, id, ownerID)
}

func (s *RolePlayService) Turns(ctx context.Context, id uint, ownerID uint) ([]models.RolePlaySessionTurn, error) {
	session, err := s.GetSession(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}
	return s.roleplay.ListTurns(ctx, session.Id)
}

func (s *RolePlayService) AppendAction(ctx context.Context, id uint, ownerID uint, input RolePlayActionInput) (*models.RolePlaySessionTurn, error) {
	session, err := s.GetSession(ctx, id, ownerID)
	if err != nil {
		return nil, err
	}
	if input.Content == "" {
		return nil, fmt.Errorf("content is required")
	}
	sequence, err := s.roleplay.NextTurnSequence(ctx, session.Id)
	if err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(input.Payload)
	turn := &models.RolePlaySessionTurn{
		SessionID:  session.Id,
		Turn:       sequence,
		AuthorType: constants.AuthorTypePlayer,
		AuthorName: defaultString(input.AuthorName, "player"),
		Content:    input.Content,
		Payload:    datatypes.JSON(payload),
		Sequence:   sequence,
	}
	if err := s.roleplay.AppendTurn(ctx, turn); err != nil {
		return nil, err
	}
	now := time.Now()
	_ = s.roleplay.UpdateSessionFields(ctx, session.Id, ownerID, map[string]any{
		"current_turn":     sequence,
		"current_scene":    input.Content,
		"last_activity_at": &now,
	})
	return turn, nil
}

func (s *RolePlayService) Resume(ctx context.Context, id uint, ownerID uint) (*models.RolePlaySession, []models.RolePlaySessionTurn, error) {
	session, err := s.GetSession(ctx, id, ownerID)
	if err != nil {
		return nil, nil, err
	}
	turns, err := s.roleplay.ListTurns(ctx, session.Id)
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	_ = s.roleplay.UpdateSessionFields(ctx, session.Id, ownerID, map[string]any{
		"status":           constants.RolePlayStatusLive,
		"last_activity_at": &now,
	})
	return session, turns, nil
}

func (s *RolePlayService) End(ctx context.Context, id uint, ownerID uint) error {
	now := time.Now()
	return s.roleplay.UpdateSessionFields(ctx, id, ownerID, map[string]any{
		"status":           constants.RolePlayStatusFinished,
		"finished_at":      &now,
		"last_activity_at": &now,
	})
}
