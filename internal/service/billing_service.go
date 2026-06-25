package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type MockPurchaseInput struct {
	UserID    uint
	ProductID string
	ReceiptID string
	Platform  string
}

type MockSubscribeInput struct {
	UserID    uint
	ProductID string
	ReceiptID string
	Platform  string
}

type MockRestoreInput struct {
	UserID    uint
	ReceiptID string
	Platform  string
}

type BillingPurchaseResult struct {
	Success bool                 `json:"success"`
	Message string               `json:"message"`
	Wallet  *AIWalletView        `json:"wallet"`
	Product *models.StoreProduct `json:"product,omitempty"`
}

type BillingSubscribeResult struct {
	Success      bool                     `json:"success"`
	Message      string                   `json:"message"`
	Wallet       *AIWalletView            `json:"wallet"`
	Subscription *models.UserSubscription `json:"subscription,omitempty"`
	Entitlement  *models.UserEntitlement  `json:"entitlement,omitempty"`
	Product      *models.StoreProduct     `json:"product,omitempty"`
}

type BillingRestoreResult struct {
	Success      bool                      `json:"success"`
	Message      string                    `json:"message"`
	Wallet       *AIWalletView             `json:"wallet"`
	Subscription *models.UserSubscription  `json:"subscription,omitempty"`
	Entitlements []models.UserEntitlement  `json:"entitlements,omitempty"`
	Transactions []models.StoreTransaction `json:"transactions,omitempty"`
}

type billingTransactionRepo interface {
	Create(ctx context.Context, txModel *models.StoreTransaction) error
	GetByProviderRef(ctx context.Context, provider string, providerRef string) (*models.StoreTransaction, error)
	ListByUserID(ctx context.Context, userID uint, limit int) ([]models.StoreTransaction, error)
}

type BillingSubscriptionView struct {
	Active            bool       `json:"active"`
	Tier              string     `json:"tier"`
	ProductID         string     `json:"productId"`
	ProductSKU        string     `json:"productSku,omitempty"`
	RenewsAt          *time.Time `json:"renewsAt,omitempty"`
	CancelAtPeriodEnd bool       `json:"cancelAtPeriodEnd"`
	MonthlyCredits    int64      `json:"monthlyCredits"`
	Status            string     `json:"status,omitempty"`
}

type billingSubscriptionRepo interface {
	Create(ctx context.Context, subscription *models.UserSubscription) error
	Save(ctx context.Context, subscription *models.UserSubscription) error
	GetByProviderRef(ctx context.Context, provider string, providerRef string) (*models.UserSubscription, error)
	GetActiveByUserID(ctx context.Context, userID uint) (*models.UserSubscription, error)
}

type BillingService struct {
	wallets       *WalletService
	products      *StoreProductService
	entitlements  *EntitlementService
	transactions  billingTransactionRepo
	subscriptions billingSubscriptionRepo
	verifier      StoreVerifier
	estimator     *AICreditEstimator
}

func NewBillingService(
	wallets *WalletService,
	products *StoreProductService,
	entitlements *EntitlementService,
	transactions billingTransactionRepo,
	subscriptions billingSubscriptionRepo,
	verifier StoreVerifier,
) *BillingService {
	if verifier == nil {
		verifier = NewMockStoreVerifier()
	}
	return &BillingService{
		wallets:       wallets,
		products:      products,
		entitlements:  entitlements,
		transactions:  transactions,
		subscriptions: subscriptions,
		verifier:      verifier,
		estimator:     NewAICreditEstimator(),
	}
}

func (s *BillingService) GetWallet(ctx context.Context, userID uint) (*AIWalletView, error) {
	if err := s.validateBillingDeps(); err != nil {
		return nil, err
	}
	return s.wallets.GetOrCreateWithStarterBonus(ctx, userID)
}

func (s *BillingService) ListProducts(ctx context.Context, productType string) ([]models.StoreProduct, error) {
	if s.products == nil {
		return nil, fmt.Errorf("billing service unavailable")
	}
	return s.products.ListActive(ctx, productType)
}

func (s *BillingService) ListLedger(ctx context.Context, userID uint, limit int, offset int) ([]models.AIWalletLedger, int64, error) {
	if s.wallets == nil {
		return nil, 0, fmt.Errorf("billing service unavailable")
	}
	return s.wallets.ListLedger(ctx, userID, limit, offset)
}

func (s *BillingService) ListEntitlements(ctx context.Context, userID uint) ([]models.UserEntitlement, error) {
	if s.entitlements == nil {
		return nil, fmt.Errorf("billing service unavailable")
	}
	return s.entitlements.ListByUserID(ctx, userID)
}

func (s *BillingService) CheckTierAccess(ctx context.Context, userID uint, tier string) (bool, string, error) {
	if s.entitlements == nil {
		return false, "", fmt.Errorf("billing service unavailable")
	}
	return s.entitlements.HasTierAccess(ctx, userID, tier)
}

func (s *BillingService) EstimateUsage(
	ctx context.Context,
	userID uint,
	actionType string,
	feature string,
	promptChars int,
	providerName string,
	modelName string,
	metadata map[string]any,
) (map[string]any, error) {
	wallet, err := s.GetWallet(ctx, userID)
	if err != nil {
		return nil, err
	}

	resolvedActionType := strings.TrimSpace(actionType)
	if resolvedActionType == "" {
		resolvedActionType = strings.TrimSpace(feature)
	}
	resolvedFeature := strings.TrimSpace(feature)
	if resolvedFeature == "" {
		resolvedFeature = resolvedActionType
	}

	var estimatedCredits int64
	var estimatedTokens int
	if resolvedActionType != "" {
		estimatedCredits = EstimateAICreditCost(resolvedActionType, metadata)
		estimatedTokens = int(estimatedCredits * s.estimator.TokensPerCoin())
	}
	if estimatedCredits <= 0 {
		estimatedTokens = estimateTokensFromChars(promptChars)
		estimate := s.estimator.EstimateFromTokens(estimatedTokens/2, estimatedTokens-estimatedTokens/2)
		estimatedCredits = estimate.NexusCoins
		estimatedTokens = estimate.TotalTokens
	}

	billingMode := normalizeBillingMode(wallet.BillingMode)

	canAfford := true
	balanceAfter := wallet.BalanceCredits
	message := ""

	if billingMode == models.BillingModeOwnKey || !AICreditsEnabled() {
		estimatedCredits = 0
	} else if estimatedCredits > 0 {
		balanceAfter = wallet.BalanceCredits - estimatedCredits
		canAfford = balanceAfter >= 0
		if !canAfford {
			message = "insufficient_credits"
		}
	}

	return map[string]any{
		"actionType":    resolvedActionType,
		"walletBalance": wallet.BalanceCredits,
		"estimate": map[string]any{
			"actionType":       resolvedActionType,
			"feature":          resolvedFeature,
			"estimatedCredits": estimatedCredits,
			"estimatedTokens":  estimatedTokens,
			"canAfford":        canAfford,
			"billingMode":      billingMode,
			"balanceAfter":     balanceAfter,
			"message":          message,
			"providerName":     providerName,
			"modelName":        modelName,
		},
	}, nil
}

func (s *BillingService) MockPurchase(ctx context.Context, input MockPurchaseInput) (*BillingPurchaseResult, error) {
	if err := s.validateBillingDeps(); err != nil {
		return nil, err
	}
	if input.UserID == 0 {
		return nil, fmt.Errorf("user id is required")
	}
	if err := s.products.EnsureDefaultMockProducts(ctx); err != nil {
		return nil, err
	}
	if _, err := s.wallets.GetOrCreateWithStarterBonus(ctx, input.UserID); err != nil {
		return nil, err
	}

	product, err := s.products.GetActiveBySlug(ctx, strings.TrimSpace(input.ProductID))
	if err != nil {
		return nil, fmt.Errorf("store product not found")
	}
	if product.ProductType != constants.BillingProductTypeConsumable &&
		product.ProductType != constants.BillingProductTypeOneTimeUnlock {
		return nil, BillingConflictError("product is not purchasable", nil)
	}

	if product.ProductType == constants.BillingProductTypeOneTimeUnlock {
		featureKey := strings.TrimSpace(product.FeatureEntitlementKey)
		if featureKey != "" && s.entitlements != nil {
			hasActive, err := s.entitlements.HasActive(ctx, input.UserID, featureKey)
			if err != nil {
				return nil, err
			}
			if hasActive {
				wallet, walletErr := s.wallets.GetOrCreateWithStarterBonus(ctx, input.UserID)
				if walletErr != nil {
					return nil, walletErr
				}
				return &BillingPurchaseResult{
					Success: true,
					Message: "feature already unlocked",
					Wallet:  wallet,
					Product: product,
				}, nil
			}
		}
	}

	platform := defaultString(strings.TrimSpace(input.Platform), models.StoreProviderMock)
	receiptID := strings.TrimSpace(input.ReceiptID)
	if receiptID == "" {
		receiptID = randomCode()
	}

	providerRef := purchaseIdempotencyKey(input.UserID, receiptID)
	if _, err := s.transactions.GetByProviderRef(ctx, platform, providerRef); err == nil {
		wallet, walletErr := s.wallets.GetOrCreateWithStarterBonus(ctx, input.UserID)
		if walletErr != nil {
			return nil, walletErr
		}
		return &BillingPurchaseResult{
			Success: true,
			Message: "purchase already processed",
			Wallet:  wallet,
			Product: product,
		}, nil
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	verification, err := s.verifier.VerifyPurchase(ctx, StoreVerificationInput{
		Platform:       platform,
		ReceiptID:      receiptID,
		StoreProductID: product.SKU(),
		UserID:         input.UserID,
	})
	if err != nil || !verification.Valid {
		return nil, BillingForbiddenError("purchase verification failed", nil)
	}

	now := time.Now()
	rawReceipt, _ := json.Marshal(verification)
	transaction := &models.StoreTransaction{
		UserID:          input.UserID,
		StoreProductID:  product.Id,
		ProductSKU:      product.Slug,
		Platform:        platform,
		Provider:        platform,
		ProviderRef:     providerRef,
		StoreReceiptID:  receiptID,
		IdempotencyKey:  providerRef,
		Status:          models.StoreTransactionStatusCompleted,
		NexusCoinsGrant: product.NexusCoinsGrant,
		CreditsGranted:  product.NexusCoinsGrant,
		AmountCents:     product.PriceCents,
		VerifiedAt:      &now,
		CompletedAt:     &now,
		RawReceipt:      datatypes.JSON(rawReceipt),
	}
	if err := s.transactions.Create(ctx, transaction); err != nil {
		return nil, err
	}

	if product.NexusCoinsGrant > 0 {
		_, err = s.wallets.Credit(
			ctx,
			input.UserID,
			product.NexusCoinsGrant,
			models.WalletLedgerTypePurchase,
			providerRef+":credit",
			constants.BillingReferenceStoreTransaction,
			"mock purchase credit",
			"billing",
		)
		if err != nil {
			return nil, err
		}
	}

	if s.entitlements != nil && strings.TrimSpace(product.Tier) != "" && product.EntitlementDays > 0 {
		expiresAt := now.Add(time.Duration(product.EntitlementDays) * 24 * time.Hour)
		if _, err := s.entitlements.Grant(
			ctx,
			input.UserID,
			tierEntitlementKey(product.Tier),
			models.EntitlementSourcePurchase,
			providerRef+":tier",
			&expiresAt,
		); err != nil {
			return nil, err
		}
	}

	if s.entitlements != nil {
		featureKey := strings.TrimSpace(product.FeatureEntitlementKey)
		if featureKey != "" {
			if _, err := s.entitlements.Grant(
				ctx,
				input.UserID,
				featureKey,
				models.EntitlementSourcePurchase,
				providerRef+":feature",
				nil,
			); err != nil {
				return nil, err
			}
		}
	}

	wallet, err := s.wallets.GetOrCreateWithStarterBonus(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	return &BillingPurchaseResult{
		Success: true,
		Message: "purchase completed",
		Wallet:  wallet,
		Product: product,
	}, nil
}

func (s *BillingService) MockSubscribe(ctx context.Context, input MockSubscribeInput) (*BillingSubscribeResult, error) {
	if err := s.validateBillingDeps(); err != nil {
		return nil, err
	}
	if input.UserID == 0 {
		return nil, fmt.Errorf("user id is required")
	}
	if err := s.products.EnsureDefaultMockProducts(ctx); err != nil {
		return nil, err
	}
	if _, err := s.wallets.GetOrCreateWithStarterBonus(ctx, input.UserID); err != nil {
		return nil, err
	}

	product, err := s.products.GetActiveBySlug(ctx, strings.TrimSpace(input.ProductID))
	if err != nil {
		return nil, fmt.Errorf("store product not found")
	}
	if product.ProductType != constants.BillingProductTypeSubscription {
		return nil, BillingConflictError("product is not a subscription", nil)
	}

	platform := defaultString(strings.TrimSpace(input.Platform), models.StoreProviderMock)
	receiptID := strings.TrimSpace(input.ReceiptID)
	if receiptID == "" {
		receiptID = randomCode()
	}

	providerRef := subscriptionIdempotencyKey(input.UserID, receiptID)
	if existing, err := s.subscriptions.GetByProviderRef(ctx, platform, providerRef); err == nil {
		wallet, walletErr := s.wallets.GetOrCreateWithStarterBonus(ctx, input.UserID)
		if walletErr != nil {
			return nil, walletErr
		}
		entitlement, _ := s.entitlements.GetActiveByKey(ctx, input.UserID, tierEntitlementKey(existing.Tier))
		return &BillingSubscribeResult{
			Success:      true,
			Message:      "subscription already active",
			Wallet:       wallet,
			Subscription: existing,
			Entitlement:  entitlement,
			Product:      product,
		}, nil
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if active, err := s.subscriptions.GetActiveByUserID(ctx, input.UserID); err == nil && active != nil {
		return nil, BillingConflictError("user already has an active subscription", map[string]any{"tier": active.Tier})
	} else if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	verification, err := s.verifier.VerifyPurchase(ctx, StoreVerificationInput{
		Platform:       platform,
		ReceiptID:      receiptID,
		StoreProductID: product.SKU(),
		UserID:         input.UserID,
	})
	if err != nil || !verification.Valid {
		return nil, BillingForbiddenError("subscription verification failed", nil)
	}

	now := time.Now()
	periodDays := product.SubscriptionPeriodDays
	if periodDays <= 0 {
		periodDays = 30
	}
	periodEnd := now.Add(time.Duration(periodDays) * 24 * time.Hour)
	rawReceipt, _ := json.Marshal(verification)
	tier := defaultString(product.Tier, constants.TierUncommon)
	entitlementKey := tierEntitlementKey(defaultString(product.SubscriptionEntitlementKey, tier))

	subscription := &models.UserSubscription{
		UserID:                 input.UserID,
		StoreProductID:         product.Id,
		ProductSKU:             product.Slug,
		Platform:               platform,
		Provider:               platform,
		StoreSubscriptionID:    receiptID,
		ProviderSubscriptionID: providerRef,
		Status:                 models.SubscriptionStatusActive,
		AutoRenew:              true,
		Tier:                   tier,
		MonthlyCredits:         product.NexusCoinsGrant,
		CurrentPeriodStart:     &now,
		CurrentPeriodEnd:       &periodEnd,
		RenewsAt:               &periodEnd,
		RawReceipt:             datatypes.JSON(rawReceipt),
	}
	if err := s.subscriptions.Create(ctx, subscription); err != nil {
		return nil, err
	}

	entitlement, err := s.entitlements.Grant(
		ctx,
		input.UserID,
		entitlementKey,
		models.EntitlementSourceSubscription,
		providerRef,
		&periodEnd,
	)
	if err != nil {
		return nil, err
	}

	wallet, err := s.wallets.ApplySubscriptionAllowance(ctx, input.UserID, product.NexusCoinsGrant, tier, providerRef+":allowance")
	if err != nil {
		return nil, err
	}

	return &BillingSubscribeResult{
		Success:      true,
		Message:      "subscription activated",
		Wallet:       wallet,
		Subscription: subscription,
		Entitlement:  entitlement,
		Product:      product,
	}, nil
}

func (s *BillingService) MockRestore(ctx context.Context, input MockRestoreInput) (*BillingRestoreResult, error) {
	if err := s.validateBillingDeps(); err != nil {
		return nil, err
	}
	if input.UserID == 0 {
		return nil, fmt.Errorf("user id is required")
	}

	wallet, err := s.wallets.GetOrCreateWithStarterBonus(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	entitlements, err := s.entitlements.ListByUserID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	transactions, err := s.transactions.ListByUserID(ctx, input.UserID, 20)
	if err != nil {
		return nil, err
	}

	subscription, err := s.subscriptions.GetActiveByUserID(ctx, input.UserID)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	receiptID := strings.TrimSpace(input.ReceiptID)
	if receiptID != "" {
		platform := defaultString(strings.TrimSpace(input.Platform), models.StoreProviderMock)
		if restoredSub, subErr := s.subscriptions.GetByProviderRef(ctx, platform, subscriptionIdempotencyKey(input.UserID, receiptID)); subErr == nil {
			subscription = restoredSub
		}
	}

	return &BillingRestoreResult{
		Success:      true,
		Message:      "restore completed",
		Wallet:       wallet,
		Subscription: subscription,
		Entitlements: entitlements,
		Transactions: transactions,
	}, nil
}

func (s *BillingService) GetSubscription(ctx context.Context, userID uint) (*BillingSubscriptionView, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user id is required")
	}
	if s.subscriptions == nil {
		return &BillingSubscriptionView{Active: false}, nil
	}

	subscription, err := s.subscriptions.GetActiveByUserID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &BillingSubscriptionView{Active: false}, nil
		}
		return nil, err
	}
	return subscriptionViewFromModel(subscription), nil
}

func (s *BillingService) CancelSubscription(ctx context.Context, userID uint, atPeriodEnd bool) (map[string]any, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user id is required")
	}
	if s.subscriptions == nil {
		return map[string]any{
			"success": true,
			"message": "no active subscription",
		}, nil
	}

	subscription, err := s.subscriptions.GetActiveByUserID(ctx, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return map[string]any{
				"success": true,
				"message": "no active subscription",
			}, nil
		}
		return nil, err
	}

	subscription.CancelAtPeriodEnd = atPeriodEnd
	subscription.AutoRenew = false
	if !atPeriodEnd {
		subscription.Status = models.SubscriptionStatusCancelled
		now := time.Now()
		subscription.CurrentPeriodEnd = &now
		subscription.RenewsAt = nil
	}

	if err := s.subscriptions.Save(ctx, subscription); err != nil {
		return nil, err
	}

	message := "subscription cancellation scheduled"
	if !atPeriodEnd {
		message = "subscription cancelled"
	}

	return map[string]any{
		"success":      true,
		"message":      message,
		"subscription": subscriptionViewFromModel(subscription),
	}, nil
}

func subscriptionViewFromModel(subscription *models.UserSubscription) *BillingSubscriptionView {
	if subscription == nil {
		return &BillingSubscriptionView{Active: false}
	}

	active := subscription.Status == models.SubscriptionStatusActive
	renewsAt := subscription.RenewsAt
	if renewsAt == nil {
		renewsAt = subscription.CurrentPeriodEnd
	}

	return &BillingSubscriptionView{
		Active:            active,
		Tier:              subscription.Tier,
		ProductID:         subscription.ProductSKU,
		ProductSKU:        subscription.ProductSKU,
		RenewsAt:          renewsAt,
		CancelAtPeriodEnd: subscription.CancelAtPeriodEnd,
		MonthlyCredits:    subscription.MonthlyCredits,
		Status:            subscription.Status,
	}
}

func (s *BillingService) UpdateBillingMode(ctx context.Context, userID uint, mode string) (*AIWalletView, error) {
	if s.wallets == nil {
		return nil, fmt.Errorf("billing service unavailable")
	}
	return s.wallets.UpdateBillingMode(ctx, userID, mode)
}

func (s *BillingService) validateBillingDeps() error {
	if s == nil || s.wallets == nil || s.products == nil {
		return fmt.Errorf("billing service unavailable")
	}
	return nil
}

func estimateTokensFromChars(promptChars int) int {
	if promptChars <= 0 {
		return 512
	}
	return promptChars / 4
}
