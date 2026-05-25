package repository

import (
	"context"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type AIUsageRepository struct {
	db *gorm.DB
}

func NewAIUsageRepository(db *gorm.DB) *AIUsageRepository {
	return &AIUsageRepository{db: db}
}

func (r *AIUsageRepository) Create(ctx context.Context, usage *models.AIUsageRecord) error {
	return r.db.WithContext(ctx).Create(usage).Error
}

func (r *AIUsageRepository) IncrementBattleUsage(ctx context.Context, battleID uint, promptTokens int, completionTokens int, totalTokens int, costMicros int64) error {
	return r.db.WithContext(ctx).
		Model(&models.BattleSave{}).
		Where("id = ?", battleID).
		Updates(map[string]any{
			"prompt_tokens":         gorm.Expr("COALESCE(prompt_tokens, 0) + ?", promptTokens),
			"completion_tokens":     gorm.Expr("COALESCE(completion_tokens, 0) + ?", completionTokens),
			"total_tokens":          gorm.Expr("COALESCE(total_tokens, 0) + ?", totalTokens),
			"estimated_cost_micros": gorm.Expr("COALESCE(estimated_cost_micros, 0) + ?", costMicros),
		}).Error
}

func (r *AIUsageRepository) IncrementRolePlayUsage(ctx context.Context, sessionID uint, promptTokens int, completionTokens int, totalTokens int, costMicros int64) error {
	return r.db.WithContext(ctx).
		Model(&models.RolePlaySession{}).
		Where("id = ?", sessionID).
		Updates(map[string]any{
			"prompt_tokens":         gorm.Expr("COALESCE(prompt_tokens, 0) + ?", promptTokens),
			"completion_tokens":     gorm.Expr("COALESCE(completion_tokens, 0) + ?", completionTokens),
			"total_tokens":          gorm.Expr("COALESCE(total_tokens, 0) + ?", totalTokens),
			"estimated_cost_micros": gorm.Expr("COALESCE(estimated_cost_micros, 0) + ?", costMicros),
		}).Error
}
