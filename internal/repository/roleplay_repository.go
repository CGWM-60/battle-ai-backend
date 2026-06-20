package repository

import (
	"context"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type RolePlayRepository struct {
	db *gorm.DB
}

func NewRolePlayRepository(db *gorm.DB) *RolePlayRepository {
	return &RolePlayRepository{db: db}
}

func (r *RolePlayRepository) ListSessionsByOwner(ctx context.Context, ownerID uint, limit int) ([]models.RolePlaySession, error) {
	var sessions []models.RolePlaySession
	err := PreloadRolePlayQuestVisuals(r.db.WithContext(ctx), "").
		Preload("ActiveCharacter").
		Where("owner_id = ?", ownerID).
		Order("updated_at DESC").
		Limit(limit).
		Find(&sessions).Error
	if err == nil {
		for index := range sessions {
			ApplyRolePlaySessionImageCompatibility(&sessions[index])
		}
	}
	return sessions, err
}

func (r *RolePlayRepository) GetSessionOwnedByID(ctx context.Context, id uint, ownerID uint) (*models.RolePlaySession, error) {
	var session models.RolePlaySession
	err := PreloadRolePlayQuestVisuals(r.db.WithContext(ctx), "").
		Preload("ActiveCharacter").
		Where("id = ? AND owner_id = ?", id, ownerID).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	ApplyRolePlaySessionImageCompatibility(&session)

	return &session, nil
}

func (r *RolePlayRepository) CreateSession(ctx context.Context, session *models.RolePlaySession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *RolePlayRepository) UpdateSessionFields(ctx context.Context, id uint, ownerID uint, fields map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&models.RolePlaySession{}).
		Where("id = ? AND owner_id = ?", id, ownerID).
		Updates(fields).Error
}

func (r *RolePlayRepository) ListTurns(ctx context.Context, sessionID uint) ([]models.RolePlaySessionTurn, error) {
	var turns []models.RolePlaySessionTurn
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("sequence ASC").
		Find(&turns).Error
	return turns, err
}

func (r *RolePlayRepository) AppendTurn(ctx context.Context, turn *models.RolePlaySessionTurn) error {
	return r.db.WithContext(ctx).Create(turn).Error
}

func (r *RolePlayRepository) NextTurnSequence(ctx context.Context, sessionID uint) (int, error) {
	var last models.RolePlaySessionTurn
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("sequence DESC").
		First(&last).Error
	if err == gorm.ErrRecordNotFound {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	return last.Sequence + 1, nil
}

func (r *RolePlayRepository) CreateQuestRun(ctx context.Context, run *models.RolePlayQuestRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

func (r *RolePlayRepository) GetQuestRunBySession(ctx context.Context, sessionID uint) (*models.RolePlayQuestRun, error) {
	var run models.RolePlayQuestRun
	err := r.db.WithContext(ctx).
		Preload("Template").
		Preload("Character").
		Where("session_id = ?", sessionID).
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *RolePlayRepository) UpdateQuestRunBySession(ctx context.Context, sessionID uint, fields map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&models.RolePlayQuestRun{}).
		Where("session_id = ?", sessionID).
		Updates(fields).Error
}

func (r *RolePlayRepository) CompleteQuestRunAndSession(ctx context.Context, sessionID uint, ownerID uint, runFields map[string]any, sessionFields map[string]any, xpReward int, coinReward int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.RolePlayQuestRun{}).
			Where("session_id = ? AND status <> ?", sessionID, "finished").
			Updates(runFields)
		if result.Error != nil {
			return result.Error
		}
		if err := tx.Model(&models.RolePlaySession{}).
			Where("id = ? AND owner_id = ?", sessionID, ownerID).
			Updates(sessionFields).Error; err != nil {
			return err
		}
		if result.RowsAffected > 0 && (xpReward > 0 || coinReward > 0) {
			return tx.Model(&models.Users{}).
				Where("id = ?", ownerID).
				Updates(map[string]any{
					"xp":   gorm.Expr("COALESCE(xp, 0) + ?", xpReward),
					"coin": gorm.Expr("COALESCE(coin, 0) + ?", coinReward),
				}).Error
		}
		return nil
	})
}
