package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"
	"cgwm/battle/internal/repository"

	"gorm.io/datatypes"
)

type RolePlaySessionInput struct {
	TemplateID     uint
	Title          string
	Mode           string
	ScenarioPrompt string
	Snapshot       map[string]any
	ProviderName   string
	ModelName      string
	APIKey         string
}

type RolePlayActionInput struct {
	AuthorName string
	Content    string
	Payload    map[string]any
}

type RolePlayService struct {
	roleplay *repository.RolePlayRepository
	quests   *repository.QuestRepository
	usage    *repository.AIUsageRepository
}

func NewRolePlayService(roleplay *repository.RolePlayRepository, quests *repository.QuestRepository, usage *repository.AIUsageRepository) *RolePlayService {
	return &RolePlayService{roleplay: roleplay, quests: quests, usage: usage}
}

func (s *RolePlayService) CreateSession(ctx context.Context, ownerID uint, input RolePlaySessionInput) (*models.RolePlaySession, error) {
	now := time.Now()
	title := input.Title
	prompt := input.ScenarioPrompt
	providerName := strings.TrimSpace(input.ProviderName)
	modelName := strings.TrimSpace(input.ModelName)
	apiKey := strings.TrimSpace(input.APIKey)
	if providerName != "" || modelName != "" || apiKey != "" {
		if providerName == "" || modelName == "" || apiKey == "" {
			return nil, fmt.Errorf("providerName, modelName and apiKey are required to launch roleplay with IA")
		}
		if _, err := ProviderURL(providerName); err != nil {
			return nil, fmt.Errorf("providerName invalide")
		}
	}
	if input.Snapshot == nil {
		input.Snapshot = map[string]any{}
	}
	if providerName != "" {
		input.Snapshot["providerName"] = providerName
		input.Snapshot["modelName"] = modelName
	}
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
	if providerName != "" {
		if err := s.appendInitialNarration(ctx, session, providerName, modelName, apiKey); err != nil {
			failedAt := time.Now()
			_ = s.roleplay.UpdateSessionFields(ctx, session.Id, ownerID, map[string]any{
				"status":           constants.RolePlayStatusFailed,
				"finished_at":      &failedAt,
				"last_activity_at": &failedAt,
			})
			return nil, err
		}
	}
	return session, nil
}

func (s *RolePlayService) appendInitialNarration(ctx context.Context, session *models.RolePlaySession, providerName string, modelName string, apiKey string) error {
	url, err := ProviderURL(providerName)
	if err != nil {
		return fmt.Errorf("providerName invalide")
	}

	callCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	sessionID := session.Id
	ai := attachUsageRecorder(s.usage, usageSessionRef{
		OwnerID:           session.OwnerID,
		SessionMode:       constants.ModeRolePlayIA,
		RolePlaySessionID: &sessionID,
		BillingSource:     billingSourceClientKey,
		ProviderName:      providerName,
		ModelName:         modelName,
	}, provider.NewsProvider(apiKey, url, modelName))
	if ai != nil {
		ai.WithUsageMetadata(provider.UsageMetadata{
			Mode:      constants.ModeRolePlayIA,
			Operation: "roleplay_narration",
			Phase:     "opening",
			Round:     1,
			ActorName: "Narrateur",
		})
	}
	response, err := ai.Chat(callCtx, []provider.ProviderMessage{
		{
			Role: "system",
			Content: `Tu es le maitre du jeu d'une quete roleplay IA.
Tu dois lancer la scene d'ouverture pour le joueur.
Ecris en francais, sois immersif, concret et jouable.
Ne termine pas la quete. Termine par une situation qui appelle une action du joueur.`,
		},
		{
			Role: "user",
			Content: fmt.Sprintf(`Titre: %s

Scenario:
%s

Lance maintenant la premiere scene.`, session.Title, session.ScenarioPrompt),
		},
	})
	if err != nil {
		return fmt.Errorf("cannot launch roleplay provider: %w", err)
	}

	sequence, err := s.roleplay.NextTurnSequence(ctx, session.Id)
	if err != nil {
		return err
	}

	payload, _ := json.Marshal(map[string]any{
		"providerName": providerName,
		"modelName":    modelName,
		"phase":        "opening",
	})
	turn := &models.RolePlaySessionTurn{
		SessionID:  session.Id,
		Turn:       sequence,
		AuthorType: "narrateur",
		AuthorName: "Narrateur",
		Content:    response,
		Payload:    datatypes.JSON(payload),
		Sequence:   sequence,
	}
	if err := s.roleplay.AppendTurn(ctx, turn); err != nil {
		return err
	}

	now := time.Now()
	_ = s.roleplay.UpdateSessionFields(ctx, session.Id, session.OwnerID, map[string]any{
		"current_turn":     sequence,
		"current_scene":    response,
		"last_activity_at": &now,
	})
	session.CurrentTurn = sequence
	session.CurrentScene = response
	session.LastActivityAt = &now

	return nil
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
