package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"
	"cgwm/battle/internal/repository"
	"cgwm/battle/internal/scenarios"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BattleRequest struct {
	Provider1            string
	Provider2            string
	IAKey1               string
	IAKey2               string
	IAModels             string
	IAModels2            string
	IA1ProfileID         uint
	IA2ProfileID         uint
	IA1Name              string
	IA1Personality       string
	IA1Mindset           string
	IA1Style             string
	IA1Goal              string
	IA1Weakness          string
	IA2Name              string
	IA2Personality       string
	IA2Mindset           string
	IA2Style             string
	IA2Goal              string
	IA2Weakness          string
	Question             string
	QuestID              uint
	Title                string
	Visibility           string
	TotalRounds          int
	RoundDurationSeconds int
	PublicVote           bool
}

type BattleRun struct {
	Battle *models.BattleSave
	IAs    []models.BattleIAConfig
}

type AIProviderInfo struct {
	Name                      string   `json:"name"`
	DisplayName               string   `json:"displayName"`
	Aliases                   []string `json:"aliases"`
	ChatCompletionsCompatible bool     `json:"chatCompletionsCompatible"`
}

type BattleService struct {
	battles    *repository.BattleRepository
	quests     *repository.QuestRepository
	iaProfiles *repository.IAProfileRepository
	live       *LiveService
}

func NewBattleService(
	battles *repository.BattleRepository,
	quests *repository.QuestRepository,
	iaProfiles *repository.IAProfileRepository,
	live *LiveService,
) *BattleService {
	return &BattleService{
		battles:    battles,
		quests:     quests,
		iaProfiles: iaProfiles,
		live:       live,
	}
}

func (s *BattleService) Create(ctx context.Context, ownerID uint, req BattleRequest) (*BattleRun, error) {
	ias, err := s.buildIAConfigs(ctx, ownerID, &req)
	if err != nil {
		return nil, err
	}

	if req.QuestID != 0 && req.Question == "" {
		quest, err := s.quests.GetPublishedBattleQuestByID(ctx, req.QuestID)
		if err != nil {
			return nil, fmt.Errorf("battle quest not found")
		}
		req.Question = quest.Content
		if req.Title == "" {
			req.Title = quest.Title
		}
	}

	if req.Question == "" {
		return nil, fmt.Errorf("question is required")
	}

	req.TotalRounds = sanitizeTotalRounds(req.TotalRounds)
	req.RoundDurationSeconds = sanitizeRoundDurationSeconds(req.RoundDurationSeconds)

	now := time.Now()
	iaSnapshot, _ := json.Marshal(stripRuntimeProviders(ias))
	title := req.Title
	if title == "" {
		title = req.Question
	}
	if len(title) > 160 {
		title = title[:160]
	}

	visibility := req.Visibility
	if visibility == "" {
		visibility = constants.VisibilityPrivate
	}

	var questID *uint
	if req.QuestID != 0 {
		questID = &req.QuestID
	}

	battle := &models.BattleSave{
		OwnerID:            ownerID,
		QuestID:            questID,
		Title:              title,
		Question:           req.Question,
		Status:             constants.BattleStatusLive,
		Visibility:         visibility,
		CurrentRound:       1,
		TotalRounds:        req.TotalRounds,
		DebateRoundSeconds: req.RoundDurationSeconds,
		PublicVote:         req.PublicVote,
		IASnapshot:         datatypes.JSON(iaSnapshot),
		Context: models.BattleMessageContext{
			Question:             req.Question,
			TotalRounds:          req.TotalRounds,
			RoundDurationSeconds: req.RoundDurationSeconds,
		},
		StartedAt:      &now,
		LastActivityAt: &now,
	}
	if err := s.battles.Create(ctx, battle); err != nil {
		return nil, fmt.Errorf("cannot create battle")
	}

	return &BattleRun{Battle: battle, IAs: ias}, nil
}

func (s *BattleService) Resume(ctx context.Context, ownerID uint, battleID uint, req BattleRequest) (*BattleRun, []models.BattleSaveTurn, error) {
	battle, err := s.battles.GetOwnedByID(ctx, battleID, ownerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, fmt.Errorf("battle not found")
		}
		return nil, nil, fmt.Errorf("cannot get battle")
	}

	if battle.Status == constants.BattleStatusFinished {
		return nil, nil, fmt.Errorf("battle already finished")
	}
	if battle.Status == constants.BattleStatusAbandoned {
		return nil, nil, fmt.Errorf("battle is abandoned")
	}

	updates := map[string]any{}
	if req.TotalRounds > 0 && battle.TotalRounds == 0 {
		battle.TotalRounds = sanitizeTotalRounds(req.TotalRounds)
		updates["total_rounds"] = battle.TotalRounds
	}
	if req.RoundDurationSeconds > 0 && battle.DebateRoundSeconds == 0 {
		battle.DebateRoundSeconds = sanitizeRoundDurationSeconds(req.RoundDurationSeconds)
		updates["debate_round_seconds"] = battle.DebateRoundSeconds
	}
	if req.PublicVote && !battle.PublicVote {
		battle.PublicVote = true
		updates["public_vote"] = true
	}
	if len(updates) > 0 {
		_ = s.battles.UpdateFields(ctx, battle.Id, updates)
	}

	turns, err := s.battles.ListTurns(ctx, battle.Id)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot list battle turns")
	}

	ias, err := s.buildResumeIAConfigs(ctx, ownerID, battle, &req)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	_ = s.battles.UpdateFields(ctx, battle.Id, map[string]any{
		"status":           constants.BattleStatusLive,
		"last_activity_at": &now,
	})

	return &BattleRun{Battle: battle, IAs: ias}, turns, nil
}

func (s *BattleService) Run(ctx context.Context, run *BattleRun, resumeTurns []models.BattleSaveTurn, onEvent func(scenarios.BattleStreamEvent)) error {
	sequence, err := s.battles.NextTurnSequence(ctx, run.Battle.Id)
	if err != nil {
		return fmt.Errorf("cannot prepare battle sequence")
	}

	chunks := make(map[string]string)
	persistEvent := func(event scenarios.BattleStreamEvent) {
		if event.Content != "" {
			key := battleEventKey(event)
			chunks[key] += event.Content
		}

		if onEvent != nil {
			onEvent(event)
		}
		if s.live != nil && ctx.Err() == nil {
			eventType := constants.LiveEventTypeChunk
			if event.Done {
				eventType = constants.LiveEventTypeMessage
			}
			if event.Type == "error" {
				eventType = constants.LiveEventTypeError
			}
			authorType := constants.AuthorTypeIA
			authorName := event.IA
			if event.Type == "judge_result" {
				authorType = constants.AuthorTypeSystem
				authorName = defaultString(event.JudgeName, "Juge")
			}
			s.live.AppendBattleEvent(ctx, run.Battle.Id, eventType, authorType, authorName, event)
		}

		if event.Done && event.Type != "error" && ctx.Err() == nil {
			key := battleEventKey(event)
			content := chunks[key]
			delete(chunks, key)

			authorType := constants.AuthorTypeIA
			authorName := event.IA
			if event.Type == "judge_result" {
				authorType = constants.AuthorTypeSystem
				authorName = defaultString(event.JudgeName, "Juge")
			}

			payloadBytes, _ := json.Marshal(map[string]any{
				"turnIndex":       event.TurnIndex,
				"judgeName":       event.JudgeName,
				"judgeSlot":       event.JudgeSlot,
				"judgeWinnerSlot": event.JudgeWinnerSlot,
				"judgeScoreOne":   event.JudgeScoreOne,
				"judgeScoreTwo":   event.JudgeScoreTwo,
			})

			turn := &models.BattleSaveTurn{
				BattleSaveID: run.Battle.Id,
				Round:        event.Round,
				Phase:        event.Type,
				AuthorType:   authorType,
				AuthorName:   authorName,
				Content:      content,
				Payload:      datatypes.JSON(payloadBytes),
				Sequence:     sequence,
			}
			sequence++

			activityAt := time.Now()
			_ = s.battles.AppendTurn(ctx, turn)
			_ = s.battles.UpdateFields(ctx, run.Battle.Id, map[string]any{
				"current_round":    event.Round,
				"last_activity_at": activityAt,
			})
		}
	}

	var runErr error
	if len(resumeTurns) > 0 {
		runErr = scenarios.ResumeBattleScenarioWithDurationStreamContext(
			ctx,
			run.Battle.Question,
			run.IAs,
			turnsToHistory(resumeTurns),
			2*time.Minute,
			persistEvent,
		)
	} else if run.Battle.TotalRounds > 0 {
		runErr = scenarios.RunBattleScenarioWithRoundsStreamContext(
			ctx,
			run.Battle.Question,
			run.IAs,
			run.Battle.TotalRounds,
			persistEvent,
		)
	} else {
		runErr = scenarios.RunBattleScenarioWithDurationStreamContext(
			ctx,
			run.Battle.Question,
			run.IAs,
			2*time.Minute,
			persistEvent,
		)
	}

	finishedAt := time.Now()
	status := constants.BattleStatusFinished
	if runErr != nil {
		status = constants.BattleStatusPaused
		if onEvent != nil {
			onEvent(scenarios.BattleStreamEvent{
				Type:  "error",
				Error: runErr.Error(),
				Done:  true,
			})
		}
	}

	writeCtx := ctx
	if ctx.Err() != nil {
		writeCtx = context.Background()
	}

	if runErr == nil && status == constants.BattleStatusFinished {
		if _, judgeErr := s.finalizeJudgeResult(writeCtx, run, &sequence, onEvent); judgeErr != nil {
			runErr = judgeErr
			status = constants.BattleStatusPaused
			if onEvent != nil {
				onEvent(scenarios.BattleStreamEvent{
					Type:  "error",
					Error: judgeErr.Error(),
					Done:  true,
				})
			}
		}
	}

	updates := map[string]any{
		"status":           status,
		"finished_at":      finishedAtIfFinished(status, &finishedAt),
		"last_activity_at": &finishedAt,
	}
	if run.Battle.Context.JudgeName != "" {
		updates["winner_name"] = winnerNameFromJudgeContext(run)
		updates["context"] = run.Battle.Context
	}
	_ = s.battles.UpdateFields(writeCtx, run.Battle.Id, updates)
	if status == constants.BattleStatusFinished && s.live != nil {
		_ = s.live.EndSessionsByBattle(writeCtx, run.Battle.Id)
	}

	return runErr
}

func (s *BattleService) RunNextRound(ctx context.Context, run *BattleRun, turns []models.BattleSaveTurn, onEvent func(scenarios.BattleStreamEvent)) error {
	sequence, err := s.battles.NextTurnSequence(ctx, run.Battle.Id)
	if err != nil {
		return fmt.Errorf("cannot prepare battle sequence")
	}

	chunks := make(map[string]string)
	persistEvent := func(event scenarios.BattleStreamEvent) {
		if event.Content != "" {
			key := battleEventKey(event)
			chunks[key] += event.Content
		}

		if onEvent != nil {
			onEvent(event)
		}
		if s.live != nil && ctx.Err() == nil {
			eventType := constants.LiveEventTypeChunk
			if event.Done {
				eventType = constants.LiveEventTypeMessage
			}
			if event.Type == "error" {
				eventType = constants.LiveEventTypeError
			}
			authorType := constants.AuthorTypeIA
			authorName := event.IA
			if event.Type == "judge_result" {
				authorType = constants.AuthorTypeSystem
				authorName = defaultString(event.JudgeName, "Juge")
			}
			s.live.AppendBattleEvent(ctx, run.Battle.Id, eventType, authorType, authorName, event)
		}

		if event.Done && event.Type != "error" && ctx.Err() == nil {
			key := battleEventKey(event)
			content := chunks[key]
			delete(chunks, key)

			authorType := constants.AuthorTypeIA
			authorName := event.IA
			if event.Type == "judge_result" {
				authorType = constants.AuthorTypeSystem
				authorName = defaultString(event.JudgeName, "Juge")
			}

			payloadBytes, _ := json.Marshal(map[string]any{
				"turnIndex":       event.TurnIndex,
				"judgeName":       event.JudgeName,
				"judgeSlot":       event.JudgeSlot,
				"judgeWinnerSlot": event.JudgeWinnerSlot,
				"judgeScoreOne":   event.JudgeScoreOne,
				"judgeScoreTwo":   event.JudgeScoreTwo,
			})

			turn := &models.BattleSaveTurn{
				BattleSaveID: run.Battle.Id,
				Round:        event.Round,
				Phase:        event.Type,
				AuthorType:   authorType,
				AuthorName:   authorName,
				Content:      content,
				Payload:      datatypes.JSON(payloadBytes),
				Sequence:     sequence,
			}
			sequence++

			activityAt := time.Now()
			_ = s.battles.AppendTurn(ctx, turn)
			_ = s.battles.UpdateFields(ctx, run.Battle.Id, map[string]any{
				"current_round":    event.Round,
				"last_activity_at": activityAt,
			})
		}
	}

	totalRounds := run.Battle.TotalRounds
	if totalRounds < 1 {
		totalRounds = 3
	}
	roundDuration := time.Duration(run.Battle.DebateRoundSeconds) * time.Second
	if run.Battle.DebateRoundSeconds <= 0 {
		roundDuration = defaultBattleDebateRoundDuration()
	}

	history := turnsToHistory(turns)
	round := scenarios.NextBattleRound(history)
	if round > totalRounds {
		round = totalRounds
	}

	runErr := scenarios.RunBattleScenarioSingleRoundStreamWithDurationContext(
		ctx,
		run.Battle.Question,
		run.IAs,
		history,
		round,
		totalRounds,
		roundDuration,
		persistEvent,
	)

	now := time.Now()
	status := constants.BattleStatusPaused
	var finishedAt *time.Time
	var winnerName string
	if runErr != nil {
		status = constants.BattleStatusPaused
		if onEvent != nil {
			onEvent(scenarios.BattleStreamEvent{
				Type:  "error",
				Error: runErr.Error(),
				Done:  true,
			})
		}
	} else if round >= totalRounds {
		status = constants.BattleStatusFinished
		finishedAt = &now
	}

	writeCtx := ctx
	if ctx.Err() != nil {
		writeCtx = context.Background()
	}

	if runErr == nil && round >= totalRounds {
		judge, judgeErr := s.finalizeJudgeResult(writeCtx, run, &sequence, onEvent)
		if judgeErr != nil {
			runErr = judgeErr
			status = constants.BattleStatusPaused
			finishedAt = nil
			if onEvent != nil {
				onEvent(scenarios.BattleStreamEvent{
					Type:  "error",
					Error: judgeErr.Error(),
					Done:  true,
				})
			}
		} else {
			winnerName = judge.WinnerName
		}
	}

	updates := map[string]any{
		"status":           status,
		"finished_at":      finishedAt,
		"last_activity_at": &now,
	}
	if winnerName != "" {
		updates["winner_name"] = winnerName
	}
	if run.Battle.Context.JudgeName != "" {
		updates["context"] = run.Battle.Context
	}
	_ = s.battles.UpdateFields(writeCtx, run.Battle.Id, updates)
	if status == constants.BattleStatusFinished && s.live != nil {
		_ = s.live.EndSessionsByBattle(writeCtx, run.Battle.Id)
	}

	return runErr
}

func (s *BattleService) finalizeJudgeResult(
	ctx context.Context,
	run *BattleRun,
	sequence *int,
	onEvent func(scenarios.BattleStreamEvent),
) (judgeDecision, error) {
	finalTurns, listErr := s.battles.ListTurns(ctx, run.Battle.Id)
	if listErr != nil {
		return judgeDecision{}, fmt.Errorf("judge mandatory: cannot load battle turns")
	}

	history := turnsToHistory(finalTurns)
	if len(history) == 0 {
		return judgeDecision{}, fmt.Errorf("judge mandatory: no IA messages to judge")
	}

	judge := decideJudgeResult(run.IAs, history)
	finalRound := maxRoundFromHistory(history)
	if finalRound < 1 {
		finalRound = run.Battle.TotalRounds
	}

	judgeEvent := scenarios.BattleStreamEvent{
		Round:           finalRound,
		Type:            "judge_result",
		Content:         judge.Reason,
		Done:            true,
		JudgeName:       judge.JudgeName,
		JudgeSlot:       judge.JudgeSlot,
		JudgeWinnerSlot: judge.WinnerSlot,
		JudgeScoreOne:   judge.ScoreOne,
		JudgeScoreTwo:   judge.ScoreTwo,
	}
	if onEvent != nil {
		onEvent(judgeEvent)
	}
	if s.live != nil {
		s.live.AppendBattleEvent(
			ctx,
			run.Battle.Id,
			constants.LiveEventTypeScore,
			constants.AuthorTypeSystem,
			judge.JudgeName,
			judgeEvent,
		)
	}

	payloadBytes, _ := json.Marshal(map[string]any{
		"judgeName":       judge.JudgeName,
		"judgeSlot":       judge.JudgeSlot,
		"judgeWinnerSlot": judge.WinnerSlot,
		"judgeScoreOne":   judge.ScoreOne,
		"judgeScoreTwo":   judge.ScoreTwo,
	})
	judgeTurn := &models.BattleSaveTurn{
		BattleSaveID: run.Battle.Id,
		Round:        finalRound,
		Phase:        "judge_result",
		AuthorType:   constants.AuthorTypeSystem,
		AuthorName:   judge.JudgeName,
		Content:      judge.Reason,
		Payload:      datatypes.JSON(payloadBytes),
		Sequence:     *sequence,
	}
	*sequence = *sequence + 1
	if err := s.battles.AppendTurn(ctx, judgeTurn); err != nil {
		return judgeDecision{}, fmt.Errorf("judge mandatory: cannot persist judge result")
	}

	run.Battle.Context.JudgeName = judge.JudgeName
	run.Battle.Context.JudgeSlot = judge.JudgeSlot
	run.Battle.Context.JudgeWinnerSlot = judge.WinnerSlot
	run.Battle.Context.JudgeScoreOne = judge.ScoreOne
	run.Battle.Context.JudgeScoreTwo = judge.ScoreTwo
	run.Battle.Context.JudgeReason = judge.Reason

	return judge, nil
}

func maxRoundFromHistory(history []models.BattleRoundMessage) int {
	maxRound := 0
	for _, msg := range history {
		if msg.Round > maxRound {
			maxRound = msg.Round
		}
	}

	return maxRound
}

func winnerNameFromJudgeContext(run *BattleRun) string {
	if run == nil {
		return ""
	}
	slot := run.Battle.Context.JudgeWinnerSlot
	if slot >= 1 && slot <= len(run.IAs) {
		return defaultString(run.IAs[slot-1].Name, fmt.Sprintf("IA %d", slot))
	}

	return ""
}

func (s *BattleService) buildIAConfigs(ctx context.Context, ownerID uint, req *BattleRequest) ([]models.BattleIAConfig, error) {
	if err := s.hydrateProfiles(ctx, ownerID, req); err != nil {
		if !hasInlineIAConfigs(req) {
			return nil, err
		}

		// The client already supplied enough battle data to continue even if the
		// referenced IA profile IDs are stale or have been deleted server-side.
		req.IA1ProfileID = 0
		req.IA2ProfileID = 0
	}

	provider1URL, err := ProviderURL(req.Provider1)
	if err != nil {
		return nil, fmt.Errorf("provider1 invalide")
	}
	provider2URL, err := ProviderURL(req.Provider2)
	if err != nil {
		return nil, fmt.Errorf("provider2 invalide")
	}

	ias := []models.BattleIAConfig{
		{
			Name:         req.IA1Name,
			Provider:     provider.NewsProvider(req.IAKey1, provider1URL, req.IAModels),
			ProviderName: normalizeProviderName(req.Provider1),
			ModelName:    req.IAModels,
			Personality:  req.IA1Personality,
			Mindset:      req.IA1Mindset,
			Style:        req.IA1Style,
			Goal:         req.IA1Goal,
			Weakness:     req.IA1Weakness,
		},
		{
			Name:         req.IA2Name,
			Provider:     provider.NewsProvider(req.IAKey2, provider2URL, req.IAModels2),
			ProviderName: normalizeProviderName(req.Provider2),
			ModelName:    req.IAModels2,
			Personality:  req.IA2Personality,
			Mindset:      req.IA2Mindset,
			Style:        req.IA2Style,
			Goal:         req.IA2Goal,
			Weakness:     req.IA2Weakness,
		},
	}

	for _, ia := range ias {
		if ia.Name == "" {
			return nil, fmt.Errorf("ia names are required")
		}
	}

	return ias, nil
}

func (s *BattleService) buildResumeIAConfigs(ctx context.Context, ownerID uint, battle *models.BattleSave, req *BattleRequest) ([]models.BattleIAConfig, error) {
	ias, err := s.buildIAConfigs(ctx, ownerID, req)
	if err == nil {
		return ias, nil
	}

	var snapshots []models.BattleIAConfig
	if len(battle.IASnapshot) == 0 || json.Unmarshal(battle.IASnapshot, &snapshots) != nil || len(snapshots) < 2 {
		return nil, err
	}

	req.IA1Name = defaultString(req.IA1Name, snapshots[0].Name)
	req.Provider1 = defaultString(req.Provider1, snapshots[0].ProviderName)
	req.IAModels = defaultString(req.IAModels, snapshots[0].ModelName)
	req.IA1Personality = defaultString(req.IA1Personality, snapshots[0].Personality)
	req.IA1Mindset = defaultString(req.IA1Mindset, snapshots[0].Mindset)
	req.IA1Style = defaultString(req.IA1Style, snapshots[0].Style)
	req.IA1Goal = defaultString(req.IA1Goal, snapshots[0].Goal)
	req.IA1Weakness = defaultString(req.IA1Weakness, snapshots[0].Weakness)

	req.IA2Name = defaultString(req.IA2Name, snapshots[1].Name)
	req.Provider2 = defaultString(req.Provider2, snapshots[1].ProviderName)
	req.IAModels2 = defaultString(req.IAModels2, snapshots[1].ModelName)
	req.IA2Personality = defaultString(req.IA2Personality, snapshots[1].Personality)
	req.IA2Mindset = defaultString(req.IA2Mindset, snapshots[1].Mindset)
	req.IA2Style = defaultString(req.IA2Style, snapshots[1].Style)
	req.IA2Goal = defaultString(req.IA2Goal, snapshots[1].Goal)
	req.IA2Weakness = defaultString(req.IA2Weakness, snapshots[1].Weakness)

	// Clear profile IDs so hydrateProfiles does not try to reload them again
	// (they already failed once, and the snapshot has supplied the missing fields).
	req.IA1ProfileID = 0
	req.IA2ProfileID = 0

	return s.buildIAConfigs(ctx, ownerID, req)
}

func (s *BattleService) hydrateProfiles(ctx context.Context, ownerID uint, req *BattleRequest) error {
	if req.IA1ProfileID != 0 {
		profile, err := s.iaProfiles.GetOwnedByID(ctx, req.IA1ProfileID, ownerID)
		if err != nil {
			return fmt.Errorf("ia1 profile not found")
		}
		applyProfile(profile, req, 1)
	}

	if req.IA2ProfileID != 0 {
		profile, err := s.iaProfiles.GetOwnedByID(ctx, req.IA2ProfileID, ownerID)
		if err != nil {
			return fmt.Errorf("ia2 profile not found")
		}
		applyProfile(profile, req, 2)
	}

	return nil
}

func ProviderURL(name string) (string, error) {
	switch normalizeProviderName(name) {
	case "mistral":
		return "https://api.mistral.ai/v1/chat/completions", nil
	case "openai", "openapi", "open_api":
		return "https://api.openai.com/v1/chat/completions", nil
	case "openrouter", "open_router":
		return "https://openrouter.ai/api/v1/chat/completions", nil
	case "xia", "xai", "x-ai":
		return "https://api.x.ai/v1/chat/completions", nil
	default:
		return "", fmt.Errorf("unknown provider")
	}
}

func SupportedAIProviders() []AIProviderInfo {
	return []AIProviderInfo{
		{
			Name:                      "mistral",
			DisplayName:               "Mistral",
			Aliases:                   []string{"mistral"},
			ChatCompletionsCompatible: true,
		},
		{
			Name:                      "openai",
			DisplayName:               "OpenAI",
			Aliases:                   []string{"openai", "openapi", "open_api"},
			ChatCompletionsCompatible: true,
		},
		{
			Name:                      "openrouter",
			DisplayName:               "OpenRouter",
			Aliases:                   []string{"openrouter", "open_router"},
			ChatCompletionsCompatible: true,
		},
		{
			Name:                      "xia",
			DisplayName:               "xAI",
			Aliases:                   []string{"xia", "xai", "x-ai"},
			ChatCompletionsCompatible: true,
		},
	}
}

func applyProfile(profile *models.IAProfile, req *BattleRequest, slot int) {
	if slot == 1 {
		req.Provider1 = defaultString(req.Provider1, profile.ProviderName)
		req.IAModels = defaultString(req.IAModels, profile.ModelName)
		req.IA1Name = defaultString(req.IA1Name, profile.Name)
		req.IA1Personality = defaultString(req.IA1Personality, profile.Personality)
		req.IA1Mindset = defaultString(req.IA1Mindset, profile.Mindset)
		req.IA1Style = defaultString(req.IA1Style, profile.Style)
		req.IA1Goal = defaultString(req.IA1Goal, profile.Goal)
		req.IA1Weakness = defaultString(req.IA1Weakness, profile.Weakness)
		return
	}

	req.Provider2 = defaultString(req.Provider2, profile.ProviderName)
	req.IAModels2 = defaultString(req.IAModels2, profile.ModelName)
	req.IA2Name = defaultString(req.IA2Name, profile.Name)
	req.IA2Personality = defaultString(req.IA2Personality, profile.Personality)
	req.IA2Mindset = defaultString(req.IA2Mindset, profile.Mindset)
	req.IA2Style = defaultString(req.IA2Style, profile.Style)
	req.IA2Goal = defaultString(req.IA2Goal, profile.Goal)
	req.IA2Weakness = defaultString(req.IA2Weakness, profile.Weakness)
}

func turnsToHistory(turns []models.BattleSaveTurn) []models.BattleRoundMessage {
	history := make([]models.BattleRoundMessage, 0, len(turns))
	for _, turn := range turns {
		if normalizeProviderName(turn.AuthorType) != constants.AuthorTypeIA {
			continue
		}

		history = append(history, models.BattleRoundMessage{
			IA:      turn.AuthorName,
			Round:   turn.Round,
			Content: turn.Content,
		})
	}

	return history
}

func stripRuntimeProviders(ias []models.BattleIAConfig) []models.BattleIAConfig {
	snapshots := make([]models.BattleIAConfig, len(ias))
	for i, ia := range ias {
		ia.Provider = nil
		snapshots[i] = ia
	}

	return snapshots
}

func battleEventKey(event scenarios.BattleStreamEvent) string {
	return fmt.Sprintf("%d:%s:%s:%d", event.Round, event.IA, event.Type, event.TurnIndex)
}

func finishedAtIfFinished(status string, finishedAt *time.Time) *time.Time {
	if status != constants.BattleStatusFinished {
		return nil
	}

	return finishedAt
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}

func hasInlineIAConfigs(req *BattleRequest) bool {
	return strings.TrimSpace(req.Provider1) != "" &&
		strings.TrimSpace(req.Provider2) != "" &&
		strings.TrimSpace(req.IAModels) != "" &&
		strings.TrimSpace(req.IAModels2) != "" &&
		strings.TrimSpace(req.IA1Name) != "" &&
		strings.TrimSpace(req.IA2Name) != ""
}

func normalizeProviderName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

type judgeDecision struct {
	JudgeName  string
	JudgeSlot  int
	WinnerSlot int
	WinnerName string
	ScoreOne   int
	ScoreTwo   int
	Reason     string
}

func decideJudgeResult(ias []models.BattleIAConfig, history []models.BattleRoundMessage) judgeDecision {
	if len(ias) == 0 {
		return judgeDecision{
			JudgeName:  "Juge IA",
			JudgeSlot:  1,
			WinnerSlot: 1,
			WinnerName: "IA 1",
			ScoreOne:   50,
			ScoreTwo:   50,
			Reason:     "Aucune donnee suffisante pour juger ce round final.",
		}
	}

	judgeIndex := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(ias))
	judgeName := defaultString(ias[judgeIndex].Name, fmt.Sprintf("IA %d", judgeIndex+1))

	charStats := make([]int, len(ias))
	messageStats := make([]int, len(ias))
	for _, message := range history {
		for index, ia := range ias {
			if strings.EqualFold(strings.TrimSpace(message.IA), strings.TrimSpace(ia.Name)) {
				charStats[index] += len([]rune(strings.TrimSpace(message.Content)))
				messageStats[index]++
				break
			}
		}
	}

	totalChars := 0
	for _, count := range charStats {
		totalChars += count
	}
	if totalChars <= 0 {
		totalChars = 1
	}

	ratioOne := float64(charStats[0]) / float64(totalChars)
	scoreOne := clampJudgeScore(int(40 + ratioOne*60))
	scoreTwo := clampJudgeScore(100 - scoreOne)

	winnerSlot := 1
	if len(ias) > 1 && (scoreTwo > scoreOne || (scoreTwo == scoreOne && rand.Intn(2) == 1)) {
		winnerSlot = 2
	}
	winnerName := defaultString(ias[winnerSlot-1].Name, fmt.Sprintf("IA %d", winnerSlot))

	reason := fmt.Sprintf(
		"Juge %s: analyse finale sur le volume d'arguments (%d/%d caracteres, %d/%d prises de parole). Verdict %s avec un score juge %d-%d.",
		judgeName,
		charStats[0],
		func() int {
			if len(charStats) > 1 {
				return charStats[1]
			}
			return 0
		}(),
		messageStats[0],
		func() int {
			if len(messageStats) > 1 {
				return messageStats[1]
			}
			return 0
		}(),
		winnerName,
		scoreOne,
		scoreTwo,
	)

	return judgeDecision{
		JudgeName:  judgeName,
		JudgeSlot:  judgeIndex + 1,
		WinnerSlot: winnerSlot,
		WinnerName: winnerName,
		ScoreOne:   scoreOne,
		ScoreTwo:   scoreTwo,
		Reason:     reason,
	}
}

func clampJudgeScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}

	return score
}

func sanitizeTotalRounds(totalRounds int) int {
	if totalRounds < 1 {
		return 3
	}
	if totalRounds > 8 {
		return 8
	}

	return totalRounds
}

func sanitizeRoundDurationSeconds(seconds int) int {
	if seconds <= 0 {
		return int(defaultBattleDebateRoundDuration().Seconds())
	}
	if seconds < 15 {
		return 15
	}
	if seconds > 120 {
		return 120
	}

	return seconds
}

func defaultBattleDebateRoundDuration() time.Duration {
	return 30 * time.Second
}
