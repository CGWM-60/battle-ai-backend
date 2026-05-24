package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/repository"

	"gorm.io/datatypes"
)

type BattleQuestInput struct {
	Slug     string
	Title    string
	Content  string
	Level    string
	Point    int
	Theme    string
	Xp       int
	Coin     int
	Mode     string
	Source   string
	Status   string
	Metadata map[string]any
}

type RolePlayQuestInput struct {
	Slug     string
	Title    string
	Summary  string
	Prompt   string
	Theme    string
	Level    string
	Xp       int
	Coin     int
	Source   string
	Status   string
	Metadata map[string]any
}

type QuestService struct {
	quests *repository.QuestRepository
}

func NewQuestService(quests *repository.QuestRepository) *QuestService {
	return &QuestService{quests: quests}
}

func (s *QuestService) ListBattle(ctx context.Context, status, theme, level string, limit int) ([]models.QuestIaBattle, error) {
	return s.quests.ListBattleQuests(ctx, status, theme, level, limit)
}

func (s *QuestService) GetBattle(ctx context.Context, id uint) (*models.QuestIaBattle, error) {
	return s.quests.GetBattleQuestByID(ctx, id)
}

func (s *QuestService) CreateBattle(ctx context.Context, input BattleQuestInput) (*models.QuestIaBattle, error) {
	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Content) == "" {
		return nil, fmt.Errorf("title and content are required")
	}
	metadata, _ := json.Marshal(input.Metadata)
	quest := &models.QuestIaBattle{
		Slug:     defaultSlug(input.Slug, input.Title),
		Title:    input.Title,
		Content:  input.Content,
		Level:    input.Level,
		Point:    input.Point,
		Theme:    input.Theme,
		Xp:       input.Xp,
		Coin:     input.Coin,
		Mode:     defaultString(input.Mode, constants.ModeBattleIA),
		Source:   defaultString(input.Source, "manual"),
		Status:   defaultString(input.Status, constants.QuestStatusPublished),
		Metadata: datatypes.JSON(metadata),
	}
	if err := s.quests.CreateBattleQuest(ctx, quest); err != nil {
		return nil, err
	}
	return quest, nil
}

func (s *QuestService) UpdateBattle(ctx context.Context, id uint, input BattleQuestInput) error {
	fields := map[string]any{}
	addString(fields, "slug", input.Slug)
	addString(fields, "title", input.Title)
	addString(fields, "content", input.Content)
	addString(fields, "level", input.Level)
	addString(fields, "theme", input.Theme)
	addString(fields, "mode", input.Mode)
	addString(fields, "source", input.Source)
	addString(fields, "status", input.Status)
	fields["point"] = input.Point
	fields["xp"] = input.Xp
	fields["coin"] = input.Coin
	if input.Metadata != nil {
		metadata, _ := json.Marshal(input.Metadata)
		fields["metadata"] = datatypes.JSON(metadata)
	}
	return s.quests.UpdateBattleQuest(ctx, id, fields)
}

func (s *QuestService) PublishBattle(ctx context.Context, id uint) error {
	return s.quests.UpdateBattleQuest(ctx, id, map[string]any{"status": constants.QuestStatusPublished})
}

func (s *QuestService) ArchiveBattle(ctx context.Context, id uint) error {
	return s.quests.UpdateBattleQuest(ctx, id, map[string]any{"status": constants.QuestStatusArchived})
}

func (s *QuestService) DeleteBattle(ctx context.Context, id uint) error {
	return s.quests.DeleteBattleQuest(ctx, id)
}

func (s *QuestService) RandomBattle(ctx context.Context, theme, level string) (*models.QuestIaBattle, error) {
	return s.quests.RandomBattleQuest(ctx, theme, level)
}

func (s *QuestService) ListRolePlay(ctx context.Context, status, theme, level string, limit int) ([]models.RolePlayQuestTemplate, error) {
	return s.quests.ListRolePlayQuests(ctx, status, theme, level, limit)
}

func (s *QuestService) GetRolePlay(ctx context.Context, id uint) (*models.RolePlayQuestTemplate, error) {
	return s.quests.GetRolePlayQuestByID(ctx, id)
}

func (s *QuestService) CreateRolePlay(ctx context.Context, input RolePlayQuestInput) (*models.RolePlayQuestTemplate, error) {
	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Prompt) == "" {
		return nil, fmt.Errorf("title and prompt are required")
	}
	metadata, _ := json.Marshal(input.Metadata)
	quest := &models.RolePlayQuestTemplate{
		Slug:     defaultSlug(input.Slug, input.Title),
		Title:    input.Title,
		Summary:  input.Summary,
		Prompt:   input.Prompt,
		Theme:    input.Theme,
		Level:    input.Level,
		Xp:       input.Xp,
		Coin:     input.Coin,
		Source:   defaultString(input.Source, "manual"),
		Status:   defaultString(input.Status, constants.QuestStatusPublished),
		Metadata: datatypes.JSON(metadata),
	}
	if err := s.quests.CreateRolePlayQuest(ctx, quest); err != nil {
		return nil, err
	}
	return quest, nil
}

func (s *QuestService) UpdateRolePlay(ctx context.Context, id uint, input RolePlayQuestInput) error {
	fields := map[string]any{}
	addString(fields, "slug", input.Slug)
	addString(fields, "title", input.Title)
	addString(fields, "summary", input.Summary)
	addString(fields, "prompt", input.Prompt)
	addString(fields, "theme", input.Theme)
	addString(fields, "level", input.Level)
	addString(fields, "source", input.Source)
	addString(fields, "status", input.Status)
	fields["xp"] = input.Xp
	fields["coin"] = input.Coin
	if input.Metadata != nil {
		metadata, _ := json.Marshal(input.Metadata)
		fields["metadata"] = datatypes.JSON(metadata)
	}
	return s.quests.UpdateRolePlayQuest(ctx, id, fields)
}

func (s *QuestService) DeleteRolePlay(ctx context.Context, id uint) error {
	return s.quests.DeleteRolePlayQuest(ctx, id)
}

func addString(fields map[string]any, key string, value string) {
	if strings.TrimSpace(value) != "" {
		fields[key] = value
	}
}

func defaultSlug(slug, title string) string {
	if strings.TrimSpace(slug) != "" {
		return slug
	}
	base := strings.ToLower(strings.TrimSpace(title))
	base = strings.ReplaceAll(base, " ", "-")
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, base)
	if base == "" {
		base = "quest"
	}
	return fmt.Sprintf("%s-%d", base, time.Now().Unix())
}
