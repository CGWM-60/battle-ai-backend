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
	var firstArcID *uint
	var firstChapterID *uint
	totalChapters := 0
	if input.TemplateID != 0 {
		template, err := s.quests.GetRolePlayQuestByID(ctx, input.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("roleplay quest not found")
		}
		templateID = &template.Id
		title = defaultString(title, template.Title)
		prompt = defaultString(prompt, template.Prompt)
		firstArcID, firstChapterID, totalChapters = rolePlayQuestProgressBounds(template)
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
			TemplateID:        templateID,
			UserID:            ownerID,
			SessionID:         &session.Id,
			Title:             title,
			Status:            constants.RolePlayStatusLive,
			CurrentStep:       1,
			TotalSteps:        totalChapters,
			CurrentArcID:      firstArcID,
			CurrentChapterID:  firstChapterID,
			CompletedChapters: datatypes.JSON([]byte("[]")),
			Journal:           "",
			State:             datatypes.JSON([]byte("{}")),
			StartedAt:         &now,
			LastActivityAt:    &now,
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
	if clientActionID := rolePlayPayloadString(input.Payload, "clientActionId"); clientActionID != "" {
		if existing := s.findExistingActionTurn(ctx, session.Id, clientActionID); existing != nil {
			return existing, nil
		}
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
	sessionSnapshot := mergeRolePlaySessionSnapshot(session.Snapshot, input.Payload)
	currentScene := rolePlayPayloadString(input.Payload, "scene")
	if currentScene == "" {
		currentScene = rolePlayPayloadString(input.Payload, "nextScene")
	}
	if currentScene == "" {
		currentScene = rolePlayPayloadString(input.Payload, "narration")
	}
	if currentScene == "" {
		currentScene = input.Content
	}
	_ = s.roleplay.UpdateSessionFields(ctx, session.Id, ownerID, map[string]any{
		"current_turn":     sequence,
		"current_scene":    currentScene,
		"snapshot":         sessionSnapshot,
		"last_activity_at": &now,
	})
	s.updateQuestRunProgressFromPayload(ctx, session, input.Payload, now)
	return turn, nil
}

func (s *RolePlayService) findExistingActionTurn(ctx context.Context, sessionID uint, clientActionID string) *models.RolePlaySessionTurn {
	turns, err := s.roleplay.ListTurns(ctx, sessionID)
	if err != nil {
		return nil
	}
	for index := range turns {
		var payload map[string]any
		if len(turns[index].Payload) == 0 {
			continue
		}
		if err := json.Unmarshal(turns[index].Payload, &payload); err != nil {
			continue
		}
		if rolePlayPayloadString(payload, "clientActionId") == clientActionID {
			return &turns[index]
		}
	}
	return nil
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
	if err := s.roleplay.UpdateSessionFields(ctx, id, ownerID, map[string]any{
		"status":           constants.RolePlayStatusFinished,
		"finished_at":      &now,
		"last_activity_at": &now,
	}); err != nil {
		return err
	}
	_ = s.roleplay.UpdateQuestRunBySession(ctx, id, map[string]any{
		"status":           constants.RolePlayStatusFinished,
		"finished_at":      &now,
		"last_activity_at": &now,
	})
	return nil
}

func rolePlayQuestProgressBounds(template *models.RolePlayQuestTemplate) (*uint, *uint, int) {
	if template == nil {
		return nil, nil, 0
	}
	var firstArcID *uint
	var firstChapterID *uint
	total := 0
	for arcIndex := range template.Arcs {
		arc := template.Arcs[arcIndex]
		if firstArcID == nil {
			id := arc.Id
			firstArcID = &id
		}
		for chapterIndex := range arc.Chapters {
			total++
			if firstChapterID == nil {
				id := arc.Chapters[chapterIndex].Id
				firstChapterID = &id
			}
		}
	}
	return firstArcID, firstChapterID, total
}

func (s *RolePlayService) updateQuestRunProgressFromPayload(ctx context.Context, session *models.RolePlaySession, payload map[string]any, now time.Time) {
	if session == nil || len(payload) == 0 {
		return
	}
	run, err := s.roleplay.GetQuestRunBySession(ctx, session.Id)
	if err != nil {
		return
	}
	fields := map[string]any{
		"last_activity_at": &now,
	}
	currentStep := run.CurrentStep
	if step := intFromPayload(payload["nextStep"]); step > 0 {
		fields["current_step"] = step
		currentStep = step
	} else if step := intFromPayload(payload["step"]); step > 0 {
		fields["current_step"] = step
		currentStep = step
	}
	if id := uintFromPayload(payload["nextArcId"]); id != nil {
		fields["current_arc_id"] = id
	} else if id := uintFromPayload(payload["currentArcId"]); id != nil {
		fields["current_arc_id"] = id
	}
	if id := uintFromPayload(payload["nextChapterId"]); id != nil {
		fields["current_chapter_id"] = id
	} else if id := uintFromPayload(payload["currentChapterId"]); id != nil {
		fields["current_chapter_id"] = id
	}
	completed := completedChapterSet(run.CompletedChapters)
	if ids := uintSliceFromPayload(payload["completedChapterIds"]); len(ids) > 0 {
		for _, id := range ids {
			completed[id] = true
		}
		data, _ := json.Marshal(sortedChapterIDs(completed))
		fields["completed_chapters"] = datatypes.JSON(data)
	} else if id := uintFromPayload(payload["completedChapterId"]); id != nil {
		completed[*id] = true
		data, _ := json.Marshal(sortedChapterIDs(completed))
		fields["completed_chapters"] = datatypes.JSON(data)
	}
	completedCount := len(completed)
	questDone := run.TotalSteps > 0 && (currentStep > run.TotalSteps || completedCount >= run.TotalSteps)
	if !questDone {
		_ = s.roleplay.UpdateQuestRunBySession(ctx, session.Id, fields)
		return
	}

	fields["status"] = constants.RolePlayStatusFinished
	fields["finished_at"] = &now
	sessionFields := map[string]any{
		"status":           constants.RolePlayStatusFinished,
		"finished_at":      &now,
		"last_activity_at": &now,
	}
	xpReward := 0
	coinReward := 0
	if run.Template != nil {
		xpReward = run.Template.Xp
		coinReward = run.Template.Coin
	}
	_ = s.roleplay.CompleteQuestRunAndSession(ctx, session.Id, session.OwnerID, fields, sessionFields, xpReward, coinReward)
}

func completedChapterSet(raw datatypes.JSON) map[uint]bool {
	completed := map[uint]bool{}
	var ids []uint
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &ids)
	}
	for _, id := range ids {
		if id > 0 {
			completed[id] = true
		}
	}
	return completed
}

func sortedChapterIDs(completed map[uint]bool) []uint {
	ids := make([]uint, 0, len(completed))
	for id := range completed {
		ids = append(ids, id)
	}
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			if ids[j] < ids[i] {
				ids[i], ids[j] = ids[j], ids[i]
			}
		}
	}
	return ids
}

func mergeRolePlaySessionSnapshot(current datatypes.JSON, payload map[string]any) datatypes.JSON {
	snapshot := map[string]any{}
	if len(current) > 0 {
		_ = json.Unmarshal(current, &snapshot)
	}
	if len(payload) == 0 {
		data, _ := json.Marshal(snapshot)
		return datatypes.JSON(data)
	}

	copyKeys := []string{
		"nextStep",
		"currentNodeIndex",
		"currentArcId",
		"nextArcId",
		"currentChapterId",
		"nextChapterId",
		"currentArcTitle",
		"currentChapterTitle",
		"currentArcPosition",
		"currentChapterPositionInArc",
		"currentArcChapterCount",
		"totalArcCount",
		"totalChapterCount",
		"arcTitles",
		"completedChapterId",
		"completedChapterIds",
		"chapterCompleted",
		"currentChapterTurn",
		"targetChapterTurns",
		"chapterPacingLabel",
		"objective",
		"narration",
		"scene",
		"nextOptions",
		"options",
	}
	for _, key := range copyKeys {
		if value, ok := payload[key]; ok {
			snapshot[key] = value
		}
	}
	if value, ok := payload["nextStep"]; ok {
		snapshot["currentStep"] = value
	}
	if value, ok := payload["nextArcId"]; ok && value != nil {
		snapshot["currentArcId"] = value
	}
	if value, ok := payload["nextChapterId"]; ok && value != nil {
		snapshot["currentChapterId"] = value
	}
	if value, ok := payload["nextOptions"]; ok {
		snapshot["currentOptions"] = value
	} else if value, ok := payload["options"]; ok {
		snapshot["currentOptions"] = value
	}
	snapshot["lastOptionId"] = payload["optionId"]
	snapshot["lastOptionLabel"] = payload["optionLabel"]
	snapshot["lastActionAt"] = time.Now().UTC().Format(time.RFC3339)

	data, _ := json.Marshal(snapshot)
	return datatypes.JSON(data)
}

func rolePlayPayloadString(payload map[string]any, key string) string {
	if len(payload) == 0 {
		return ""
	}
	text := strings.TrimSpace(fmt.Sprint(payload[key]))
	if text == "<nil>" {
		return ""
	}
	return text
}

func intFromPayload(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	case string:
		var parsed int
		_, _ = fmt.Sscanf(v, "%d", &parsed)
		return parsed
	default:
		return 0
	}
}

func uintFromPayload(value any) *uint {
	parsed := intFromPayload(value)
	if parsed <= 0 {
		return nil
	}
	id := uint(parsed)
	return &id
}

func uintSliceFromPayload(value any) []uint {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	ids := make([]uint, 0, len(raw))
	for _, item := range raw {
		if id := uintFromPayload(item); id != nil {
			ids = append(ids, *id)
		}
	}
	return ids
}
