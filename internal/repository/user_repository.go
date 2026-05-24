package repository

import (
	"context"
	"fmt"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.Users, error) {
	var user models.Users
	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uint) (*models.Users, error) {
	var user models.Users
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) Create(ctx context.Context, user *models.Users) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) UpdateFields(ctx context.Context, id uint, fields map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&models.Users{}).
		Where("id = ?", id).
		Updates(fields).Error
}

func (r *UserRepository) UpdateProgression(ctx context.Context, id uint, xp *int, coin *int, xpDelta *int, coinDelta *int) (*models.Users, error) {
	var updatedUser models.Users

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", id).
			First(&updatedUser).Error; err != nil {
			return err
		}

		nextXP := updatedUser.Xp
		nextCoin := updatedUser.Coin

		if xp != nil {
			nextXP = *xp
		}
		if coin != nil {
			nextCoin = *coin
		}
		if xpDelta != nil {
			nextXP += *xpDelta
		}
		if coinDelta != nil {
			nextCoin += *coinDelta
		}

		if nextXP < 0 || nextCoin < 0 {
			return fmt.Errorf("xp and coin cannot be negative")
		}

		if err := tx.Model(&updatedUser).
			Updates(map[string]any{
				"xp":   nextXP,
				"coin": nextCoin,
			}).Error; err != nil {
			return err
		}

		updatedUser.Xp = nextXP
		updatedUser.Coin = nextCoin
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &updatedUser, nil
}
