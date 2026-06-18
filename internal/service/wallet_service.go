package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
)

type AIWalletView struct {
	BalanceCredits     int64     `json:"balanceCredits"`
	MonthlyAllowance   int64     `json:"monthlyAllowance"`
	UsedThisMonth      int64     `json:"usedThisMonth"`
	BillingMode        string    `json:"billingMode"`
	SubscriptionTier   string    `json:"subscriptionTier"`
	SubscriptionActive bool      `json:"subscriptionActive"`
	LowBalance         bool      `json:"lowBalance"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type walletRepository interface {
	GetByUserID(ctx context.Context, userID uint) (*models.UserAIWallet, error)
	GetOrCreate(ctx context.Context, userID uint) (*models.UserAIWallet, error)
	Save(ctx context.Context, wallet *models.UserAIWallet) error
	UpdateBillingMode(ctx context.Context, userID uint, mode string) (*models.UserAIWallet, error)
	ApplyLedgerMovement(ctx context.Context, userID uint, amount int64, entryType string, idempotencyKey string, referenceType string, description string, feature string, markStarterBonus bool) (*models.AIWalletLedger, error)
	ListLedger(ctx context.Context, userID uint, limit int, offset int) ([]models.AIWalletLedger, int64, error)
}

type WalletService struct {
	wallets   walletRepository
	estimator *AICreditEstimator
}

func NewWalletService(wallets walletRepository, estimator *AICreditEstimator) *WalletService {
	if estimator == nil {
		estimator = NewAICreditEstimator()
	}
	return &WalletService{wallets: wallets, estimator: estimator}
}

func (s *WalletService) GetOrCreateWithStarterBonus(ctx context.Context, userID uint) (*AIWalletView, error) {
	if s == nil || s.wallets == nil {
		return nil, fmt.Errorf("wallet service unavailable")
	}
	if userID == 0 {
		return nil, fmt.Errorf("user id is required")
	}

	wallet, err := s.wallets.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !wallet.StarterBonusGranted {
		bonus := starterBonusNexusCoins()
		if bonus > 0 {
			_, err = s.wallets.ApplyLedgerMovement(
				ctx,
				userID,
				bonus,
				models.WalletLedgerTypeBonus,
				starterBonusIdempotencyKey(userID),
				constants.BillingReferenceManual,
				"starter bonus",
				"billing",
				true,
			)
			if err != nil {
				return nil, MapBillingError(err)
			}
		} else {
			wallet.StarterBonusGranted = true
			if err := s.wallets.Save(ctx, wallet); err != nil {
				return nil, err
			}
		}
		wallet, err = s.wallets.GetByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
	}

	return walletViewFromModel(wallet), nil
}

func (s *WalletService) UpdateBillingMode(ctx context.Context, userID uint, mode string) (*AIWalletView, error) {
	wallet, err := s.wallets.UpdateBillingMode(ctx, userID, normalizeBillingMode(mode))
	if err != nil {
		return nil, err
	}
	return walletViewFromModel(wallet), nil
}

func (s *WalletService) Credit(ctx context.Context, userID uint, amount int64, entryType string, idempotencyKey string, referenceType string, description string, feature string) (*models.AIWalletLedger, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("credit amount must be positive")
	}
	entry, err := s.wallets.ApplyLedgerMovement(ctx, userID, amount, entryType, idempotencyKey, referenceType, description, feature, false)
	return entry, MapBillingError(err)
}

func (s *WalletService) Consume(ctx context.Context, userID uint, amount int64, idempotencyKey string, description string, feature string) (*models.AIWalletLedger, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("consume amount must be positive")
	}
	entry, err := s.wallets.ApplyLedgerMovement(ctx, userID, -amount, models.WalletLedgerTypeConsumption, idempotencyKey, constants.BillingReferenceAIUsage, description, feature, false)
	return entry, MapBillingError(err)
}

func (s *WalletService) ConsumeForTokens(ctx context.Context, userID uint, promptTokens int, completionTokens int, idempotencyKey string, feature string, description string) (*models.AIWalletLedger, AICreditEstimate, error) {
	estimate := s.estimator.EstimateFromTokens(promptTokens, completionTokens)
	if estimate.NexusCoins <= 0 {
		return nil, estimate, nil
	}
	entry, err := s.Consume(ctx, userID, estimate.NexusCoins, idempotencyKey, description, feature)
	return entry, estimate, err
}

func (s *WalletService) ListLedger(ctx context.Context, userID uint, limit int, offset int) ([]models.AIWalletLedger, int64, error) {
	return s.wallets.ListLedger(ctx, userID, limit, offset)
}

func (s *WalletService) ApplySubscriptionAllowance(ctx context.Context, userID uint, monthlyCredits int64, tier string, referenceID string) (*AIWalletView, error) {
	if monthlyCredits > 0 {
		_, err := s.Credit(ctx, userID, monthlyCredits, models.WalletLedgerTypeSubscription, referenceID, constants.BillingReferenceSubscription, "subscription monthly grant", "billing")
		if err != nil {
			return nil, err
		}
	}

	wallet, err := s.wallets.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}
	wallet.SubscriptionTier = tier
	wallet.SubscriptionActive = true
	wallet.MonthlyAllowance = monthlyCredits
	wallet.UsedThisMonth = 0
	now := time.Now()
	wallet.AllowanceResetAt = &now
	if err := s.wallets.Save(ctx, wallet); err != nil {
		return nil, err
	}
	return walletViewFromModel(wallet), nil
}

func walletViewFromModel(wallet *models.UserAIWallet) *AIWalletView {
	if wallet == nil {
		return nil
	}
	return &AIWalletView{
		BalanceCredits:     wallet.BalanceCredits,
		MonthlyAllowance:   wallet.MonthlyAllowance,
		UsedThisMonth:      wallet.UsedThisMonth,
		BillingMode:        normalizeBillingMode(wallet.BillingMode),
		SubscriptionTier:   wallet.SubscriptionTier,
		SubscriptionActive: wallet.SubscriptionActive,
		LowBalance:         wallet.LowBalance,
		UpdatedAt:          wallet.UpdatedAt,
	}
}

func normalizeBillingMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case models.BillingModeOwnKey, "ownkey", "api_key", "provider", "byok", "client_key":
		return models.BillingModeOwnKey
	default:
		return models.BillingModePlatform
	}
}

func purchaseIdempotencyKey(userID uint, receiptID string) string {
	return "purchase:" + strconv.FormatUint(uint64(userID), 10) + ":" + receiptID
}

func subscriptionIdempotencyKey(userID uint, receiptID string) string {
	return "subscription:" + strconv.FormatUint(uint64(userID), 10) + ":" + receiptID
}