package repository

import (
	"context"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type QuestRepository struct {
	db *gorm.DB
}

func NewQuestRepository(db *gorm.DB) *QuestRepository {
	return &QuestRepository{db: db}
}

func (r *QuestRepository) GetPublishedBattleQuestByID(ctx context.Context, id uint) (*models.QuestIaBattle, error) {
	var quest models.QuestIaBattle
	err := r.db.WithContext(ctx).
		Where("id = ? AND status = ?", id, constants.QuestStatusPublished).
		First(&quest).Error
	if err != nil {
		return nil, err
	}

	return &quest, nil
}

func (r *QuestRepository) ListBattleQuests(ctx context.Context, status string, theme string, level string, limit int) ([]models.QuestIaBattle, error) {
	var quests []models.QuestIaBattle
	query := r.db.WithContext(ctx).Model(&models.QuestIaBattle{})
	query = applyQuestQuery(query, status, theme, level)
	err := query.Order("created_at DESC").Limit(limit).Find(&quests).Error
	return quests, err
}

func (r *QuestRepository) GetBattleQuestByID(ctx context.Context, id uint) (*models.QuestIaBattle, error) {
	var quest models.QuestIaBattle
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&quest).Error
	if err != nil {
		return nil, err
	}
	return &quest, nil
}

func (r *QuestRepository) CreateBattleQuest(ctx context.Context, quest *models.QuestIaBattle) error {
	return r.db.WithContext(ctx).Create(quest).Error
}

func (r *QuestRepository) UpdateBattleQuest(ctx context.Context, id uint, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&models.QuestIaBattle{}).Where("id = ?", id).Updates(fields).Error
}

func (r *QuestRepository) DeleteBattleQuest(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.QuestIaBattle{}, id).Error
}

func (r *QuestRepository) RandomBattleQuest(ctx context.Context, theme string, level string) (*models.QuestIaBattle, error) {
	var quest models.QuestIaBattle
	query := r.db.WithContext(ctx).Where("status = ?", constants.QuestStatusPublished)
	if theme != "" {
		query = query.Where("theme = ?", theme)
	}
	if level != "" {
		query = query.Where("level = ?", level)
	}
	err := query.Order("RAND()").First(&quest).Error
	if err != nil {
		return nil, err
	}
	return &quest, nil
}

func (r *QuestRepository) ListRolePlayQuests(ctx context.Context, status string, theme string, level string, limit int) ([]models.RolePlayQuestTemplate, error) {
	var quests []models.RolePlayQuestTemplate
	query := r.db.WithContext(ctx).Model(&models.RolePlayQuestTemplate{})
	query = applyQuestQuery(query, status, theme, level)
	err := query.Order("created_at DESC").Limit(limit).Find(&quests).Error
	return quests, err
}

func (r *QuestRepository) GetRolePlayQuestByID(ctx context.Context, id uint) (*models.RolePlayQuestTemplate, error) {
	var quest models.RolePlayQuestTemplate
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&quest).Error
	if err != nil {
		return nil, err
	}
	return &quest, nil
}

func (r *QuestRepository) CreateRolePlayQuest(ctx context.Context, quest *models.RolePlayQuestTemplate) error {
	return r.db.WithContext(ctx).Create(quest).Error
}

func (r *QuestRepository) UpdateRolePlayQuest(ctx context.Context, id uint, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&models.RolePlayQuestTemplate{}).Where("id = ?", id).Updates(fields).Error
}

func (r *QuestRepository) DeleteRolePlayQuest(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.RolePlayQuestTemplate{}, id).Error
}

func applyQuestQuery(query *gorm.DB, status string, theme string, level string) *gorm.DB {
	if status == "" {
		status = constants.QuestStatusPublished
	}
	if status != "all" {
		query = query.Where("status = ?", status)
	}
	if theme != "" {
		query = query.Where("theme = ?", theme)
	}
	if level != "" {
		query = query.Where("level = ?", level)
	}
	return query
}
