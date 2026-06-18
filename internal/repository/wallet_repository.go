package repository

import (
	"context"
	"fmt"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WalletRepository struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) *WalletRepository {
	return &WalletRepository{db: db}
}

func (r *WalletRepository) DB() *gorm.DB {
	return r.db
}

func (r *WalletRepository) GetByUserID(ctx context.Context, userID uint) (*models.UserAIWallet, error) {
	var wallet models.UserAIWallet
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&wallet).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *WalletRepository) Create(ctx context.Context, wallet *models.UserAIWallet) error {
	return r.db.WithContext(ctx).Create(wallet).Error
}

func (r *WalletRepository) Save(ctx context.Context, wallet *models.UserAIWallet) error {
	return r.db.WithContext(ctx).Save(wallet).Error
}

func (r *WalletRepository) UpdateBillingMode(ctx context.Context, userID uint, mode string) (*models.UserAIWallet, error) {
	wallet, err := r.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}
	wallet.BillingMode = mode
	if err := r.db.WithContext(ctx).Save(wallet).Error; err != nil {
		return nil, err
	}
	return wallet, nil
}

func (r *WalletRepository) GetOrCreate(ctx context.Context, userID uint) (*models.UserAIWallet, error) {
	wallet, err := r.GetByUserID(ctx, userID)
	if err == nil {
		return wallet, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	newWallet := &models.UserAIWallet{
		UserID:              userID,
		BalanceCredits:      0,
		Currency:            constants.BillingCurrencyNexusCoin,
		BillingMode:         models.BillingModePlatform,
		LowBalance:          true,
		StarterBonusGranted: false,
	}
	if err := r.Create(ctx, newWallet); err != nil {
		return nil, err
	}
	return r.GetByUserID(ctx, userID)
}

func (r *WalletRepository) ListLedger(ctx context.Context, userID uint, limit int, offset int) ([]models.AIWalletLedger, int64, error) {
	if limit <= 0 {
		limit = 30
	}
	if offset < 0 {
		offset = 0
	}

	var total int64
	if err := r.db.WithContext(ctx).
		Model(&models.AIWalletLedger{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var entries []models.AIWalletLedger
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&entries).Error
	return entries, total, err
}

func (r *WalletRepository) ApplyLedgerMovement(
	ctx context.Context,
	userID uint,
	amount int64,
	entryType string,
	idempotencyKey string,
	referenceType string,
	description string,
	feature string,
	markStarterBonus bool,
) (*models.AIWalletLedger, error) {
	if amount == 0 {
		return nil, fmt.Errorf("ledger amount cannot be zero")
	}
	if idempotencyKey == "" {
		return nil, fmt.Errorf("idempotency key is required")
	}

	var createdEntry models.AIWalletLedger
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := r.getLedgerByReferenceTx(ctx, tx, idempotencyKey)
		if err == nil {
			createdEntry = *existing
			return nil
		}
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		wallet, err := r.getOrCreateWalletForUpdateTx(ctx, tx, userID)
		if err != nil {
			return err
		}

		nextBalance := wallet.BalanceCredits + amount
		if nextBalance < 0 {
			return fmt.Errorf("insufficient credits")
		}

		entry := &models.AIWalletLedger{
			WalletID:       wallet.Id,
			UserID:         userID,
			Amount:         amount,
			BalanceAfter:   nextBalance,
			EntryType:      entryType,
			Currency:       wallet.Currency,
			ReferenceID:    idempotencyKey,
			IdempotencyKey: idempotencyKey,
			ReferenceType:  referenceType,
			Description:    description,
			Feature:        feature,
		}
		if err := tx.WithContext(ctx).Create(entry).Error; err != nil {
			return err
		}

		wallet.BalanceCredits = nextBalance
		wallet.LowBalance = nextBalance < 50
		if markStarterBonus {
			wallet.StarterBonusGranted = true
		}
		wallet.UpdatedAt = time.Now()
		if err := tx.WithContext(ctx).Save(wallet).Error; err != nil {
			return err
		}

		createdEntry = *entry
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &createdEntry, nil
}

func (r *WalletRepository) getLedgerByReferenceTx(ctx context.Context, tx *gorm.DB, referenceID string) (*models.AIWalletLedger, error) {
	var entry models.AIWalletLedger
	err := tx.WithContext(ctx).Where("reference_id = ?", referenceID).First(&entry).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *WalletRepository) getOrCreateWalletForUpdateTx(ctx context.Context, tx *gorm.DB, userID uint) (*models.UserAIWallet, error) {
	var wallet models.UserAIWallet
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&wallet).Error
	if err == nil {
		return &wallet, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	newWallet := &models.UserAIWallet{
		UserID:              userID,
		BalanceCredits:      0,
		Currency:            constants.BillingCurrencyNexusCoin,
		BillingMode:         models.BillingModePlatform,
		LowBalance:          true,
		StarterBonusGranted: false,
	}
	if err := tx.WithContext(ctx).Create(newWallet).Error; err != nil {
		return nil, err
	}

	err = tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&wallet).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}