package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type memoryWalletRepo struct {
	mu       sync.Mutex
	nextID   uint
	wallets  map[uint]*models.UserAIWallet
	ledger   map[string]*models.AIWalletLedger
	byUserID map[uint]uint
}

func newMemoryWalletRepo() *memoryWalletRepo {
	return &memoryWalletRepo{
		nextID:   1,
		wallets:  map[uint]*models.UserAIWallet{},
		ledger:   map[string]*models.AIWalletLedger{},
		byUserID: map[uint]uint{},
	}
}

func (r *memoryWalletRepo) GetByUserID(_ context.Context, userID uint) (*models.UserAIWallet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	walletID, ok := r.byUserID[userID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyWallet := *r.wallets[walletID]
	return &copyWallet, nil
}

func (r *memoryWalletRepo) GetOrCreate(ctx context.Context, userID uint) (*models.UserAIWallet, error) {
	wallet, err := r.GetByUserID(ctx, userID)
	if err == nil {
		return wallet, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	wallet = &models.UserAIWallet{
		UserID:         userID,
		BalanceCredits: 0,
		Currency:       constants.BillingCurrencyNexusCoin,
		BillingMode:    models.BillingModePlatform,
		LowBalance:     true,
	}
	if err := r.Create(ctx, wallet); err != nil {
		return nil, err
	}
	return r.GetByUserID(ctx, userID)
}

func (r *memoryWalletRepo) Create(_ context.Context, wallet *models.UserAIWallet) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	wallet.Id = r.nextID
	r.nextID++
	r.wallets[wallet.Id] = wallet
	r.byUserID[wallet.UserID] = wallet.Id
	return nil
}

func (r *memoryWalletRepo) Save(_ context.Context, wallet *models.UserAIWallet) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.wallets[wallet.Id] = wallet
	return nil
}

func (r *memoryWalletRepo) UpdateBillingMode(ctx context.Context, userID uint, mode string) (*models.UserAIWallet, error) {
	wallet, err := r.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}
	wallet.BillingMode = mode
	return wallet, r.Save(ctx, wallet)
}

func (r *memoryWalletRepo) ApplyLedgerMovement(_ context.Context, userID uint, amount int64, entryType string, idempotencyKey string, referenceType string, description string, feature string, markStarterBonus bool) (*models.AIWalletLedger, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.ledger[idempotencyKey]; ok {
		copyEntry := *existing
		return &copyEntry, nil
	}

	walletID, ok := r.byUserID[userID]
	if !ok {
		wallet := &models.UserAIWallet{
			Id:             r.nextID,
			UserID:         userID,
			BalanceCredits: 0,
			Currency:       constants.BillingCurrencyNexusCoin,
			BillingMode:    models.BillingModePlatform,
		}
		r.nextID++
		r.wallets[wallet.Id] = wallet
		r.byUserID[userID] = wallet.Id
		walletID = wallet.Id
	}

	wallet := r.wallets[walletID]
	nextBalance := wallet.BalanceCredits + amount
	if nextBalance < 0 {
		return nil, errors.New("insufficient credits")
	}

	entry := &models.AIWalletLedger{
		Id:             r.nextID,
		WalletID:       walletID,
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
	r.nextID++
	r.ledger[idempotencyKey] = entry
	wallet.BalanceCredits = nextBalance
	wallet.LowBalance = nextBalance < 50
	if markStarterBonus {
		wallet.StarterBonusGranted = true
	}
	return entry, nil
}

func (r *memoryWalletRepo) ListLedger(_ context.Context, userID uint, limit int, offset int) ([]models.AIWalletLedger, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entries := make([]models.AIWalletLedger, 0)
	for _, entry := range r.ledger {
		if entry.UserID == userID {
			entries = append(entries, *entry)
		}
	}
	total := int64(len(entries))
	_ = limit
	_ = offset
	return entries, total, nil
}

type memoryProductRepo struct {
	products map[string]*models.StoreProduct
}

func newMemoryProductRepo() *memoryProductRepo {
	repo := &memoryProductRepo{products: map[string]*models.StoreProduct{}}
	repo.products["starter_ai"] = &models.StoreProduct{
		Id: 1, Slug: "starter_ai", StoreProductID: "mock.starter_ai",
		ProductType: models.StoreProductTypeCreditPack, Status: models.StoreProductStatusActive,
		NexusCoinsGrant: 1200, PriceCents: 199, Currency: "EUR",
		Tier: constants.TierUncommon, EntitlementDays: 7,
	}
	repo.products["nexus_light_monthly"] = &models.StoreProduct{
		Id: 2, Slug: "nexus_light_monthly", StoreProductID: "mock.nexus_light_monthly",
		ProductType: models.StoreProductTypeSubscription, Status: models.StoreProductStatusActive,
		NexusCoinsGrant: 4000, Tier: constants.TierUncommon,
		SubscriptionEntitlementKey: constants.TierUncommon,
		SubscriptionPeriodDays:     30,
	}
	return repo
}

func (r *memoryProductRepo) GetBySlug(_ context.Context, slug string) (*models.StoreProduct, error) {
	product, ok := r.products[slug]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyProduct := *product
	return &copyProduct, nil
}

func (r *memoryProductRepo) ListActive(_ context.Context, productType string) ([]models.StoreProduct, error) {
	items := make([]models.StoreProduct, 0)
	for _, product := range r.products {
		if productType != "" && product.ProductType != productType {
			continue
		}
		items = append(items, *product)
	}
	return items, nil
}

func (r *memoryProductRepo) Count(_ context.Context) (int64, error) {
	return int64(len(r.products)), nil
}

func (r *memoryProductRepo) Create(_ context.Context, product *models.StoreProduct) error {
	r.products[product.Slug] = product
	return nil
}

type memoryTransactionRepo struct {
	mu     sync.Mutex
	nextID uint
	byRef  map[string]*models.StoreTransaction
}

func newMemoryTransactionRepo() *memoryTransactionRepo {
	return &memoryTransactionRepo{nextID: 1, byRef: map[string]*models.StoreTransaction{}}
}

func (r *memoryTransactionRepo) Create(_ context.Context, txModel *models.StoreTransaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	txModel.Id = r.nextID
	r.nextID++
	r.byRef[txModel.Provider+":"+txModel.ProviderRef] = txModel
	return nil
}

func (r *memoryTransactionRepo) GetByProviderRef(_ context.Context, provider string, providerRef string) (*models.StoreTransaction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	txModel, ok := r.byRef[provider+":"+providerRef]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copyTx := *txModel
	return &copyTx, nil
}

func (r *memoryTransactionRepo) ListByUserID(_ context.Context, userID uint, limit int) ([]models.StoreTransaction, error) {
	return nil, nil
}

type memorySubscriptionRepo struct {
	mu     sync.Mutex
	nextID uint
	byRef  map[string]*models.UserSubscription
	byUser map[uint]*models.UserSubscription
}

func newMemorySubscriptionRepo() *memorySubscriptionRepo {
	return &memorySubscriptionRepo{
		nextID: 1,
		byRef:  map[string]*models.UserSubscription{},
		byUser: map[uint]*models.UserSubscription{},
	}
}

func (r *memorySubscriptionRepo) Create(_ context.Context, subscription *models.UserSubscription) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	subscription.Id = r.nextID
	r.nextID++
	ref := subscription.ProviderSubscriptionID
	if ref == "" {
		ref = subscription.StoreSubscriptionID
	}
	provider := subscription.Provider
	if provider == "" {
		provider = subscription.Platform
	}
	r.byRef[provider+":"+ref] = subscription
	r.byUser[subscription.UserID] = subscription
	return nil
}

func (r *memorySubscriptionRepo) GetByProviderRef(_ context.Context, provider string, providerRef string) (*models.UserSubscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sub, ok := r.byRef[provider+":"+providerRef]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copySub := *sub
	return &copySub, nil
}

func (r *memorySubscriptionRepo) GetActiveByUserID(_ context.Context, userID uint) (*models.UserSubscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	sub, ok := r.byUser[userID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	copySub := *sub
	return &copySub, nil
}

type memoryEntitlementRepo struct {
	mu           sync.Mutex
	nextID       uint
	entitlements []models.UserEntitlement
}

func newMemoryEntitlementRepo() *memoryEntitlementRepo {
	return &memoryEntitlementRepo{nextID: 1}
}

func (r *memoryEntitlementRepo) Create(_ context.Context, entitlement *models.UserEntitlement) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	entitlement.Id = r.nextID
	r.nextID++
	r.entitlements = append(r.entitlements, *entitlement)
	return nil
}

func (r *memoryEntitlementRepo) ListByUserID(_ context.Context, userID uint) ([]models.UserEntitlement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items := make([]models.UserEntitlement, 0)
	for _, item := range r.entitlements {
		if item.UserID == userID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (r *memoryEntitlementRepo) HasActive(_ context.Context, userID uint, key string, at time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.entitlements {
		if item.UserID == userID && item.Key == key && item.Active {
			if item.ExpiresAt == nil || item.ExpiresAt.After(at) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (r *memoryEntitlementRepo) GetActiveByKey(_ context.Context, userID uint, key string, at time.Time) (*models.UserEntitlement, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for index := range r.entitlements {
		item := r.entitlements[index]
		if item.UserID == userID && item.Key == key && item.Active {
			if item.ExpiresAt == nil || item.ExpiresAt.After(at) {
				copyItem := item
				return &copyItem, nil
			}
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func newTestBillingStack(t *testing.T) *BillingService {
	t.Helper()
	t.Setenv("DEFAULT_STARTER_BONUS_CREDITS", "500")
	productRepo := newMemoryProductRepo()
	return &BillingService{
		wallets:       NewWalletService(newMemoryWalletRepo(), NewAICreditEstimator()),
		products:      &StoreProductService{products: productRepo},
		entitlements:  &EntitlementService{entitlements: newMemoryEntitlementRepo()},
		transactions:  newMemoryTransactionRepo(),
		subscriptions: newMemorySubscriptionRepo(),
		verifier:      NewMockStoreVerifier(),
		estimator:     NewAICreditEstimator(),
	}
}

func TestWalletStarterBonusIsIdempotent(t *testing.T) {
	t.Setenv("DEFAULT_STARTER_BONUS_CREDITS", "500")
	walletService := NewWalletService(newMemoryWalletRepo(), NewAICreditEstimator())

	first, err := walletService.GetOrCreateWithStarterBonus(context.Background(), 42)
	if err != nil {
		t.Fatalf("first wallet: %v", err)
	}
	if first.BalanceCredits != 500 {
		t.Fatalf("expected starter bonus balance 500, got %d", first.BalanceCredits)
	}

	second, err := walletService.GetOrCreateWithStarterBonus(context.Background(), 42)
	if err != nil {
		t.Fatalf("second wallet: %v", err)
	}
	if second.BalanceCredits != 500 {
		t.Fatalf("expected unchanged balance, got %d", second.BalanceCredits)
	}
}

func TestBillingServiceMockPurchaseCreditsWallet(t *testing.T) {
	billing := newTestBillingStack(t)
	result, err := billing.MockPurchase(context.Background(), MockPurchaseInput{
		UserID: 7, ProductID: "starter_ai", ReceiptID: "receipt-001",
	})
	if err != nil {
		t.Fatalf("mock purchase: %v", err)
	}
	if result.Wallet.BalanceCredits != 1700 {
		t.Fatalf("expected 1700, got %d", result.Wallet.BalanceCredits)
	}
}

func TestBillingServiceMockSubscribeGrantsEntitlement(t *testing.T) {
	billing := newTestBillingStack(t)
	result, err := billing.MockSubscribe(context.Background(), MockSubscribeInput{
		UserID: 9, ProductID: "nexus_light_monthly", ReceiptID: "sub-001",
	})
	if err != nil {
		t.Fatalf("mock subscribe: %v", err)
	}
	if result.Entitlement == nil || result.Entitlement.Key != "tier:uncommon" {
		t.Fatalf("expected tier entitlement, got %+v", result.Entitlement)
	}
	if result.Wallet.BalanceCredits != 4500 {
		t.Fatalf("expected 4500, got %d", result.Wallet.BalanceCredits)
	}
}

func TestWalletConsumeRequiresBalance(t *testing.T) {
	t.Setenv("BILLING_STARTER_BONUS_NEXUS_COINS", "10")
	walletService := NewWalletService(newMemoryWalletRepo(), NewAICreditEstimator())
	ctx := context.Background()
	if _, err := walletService.GetOrCreateWithStarterBonus(ctx, 3); err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	if _, err := walletService.Consume(ctx, 3, 50, "consume:test", "test", "billing"); err == nil {
		t.Fatal("expected insufficient balance error")
	}
	if _, err := walletService.Consume(ctx, 3, 5, "consume:test-ok", "test", "billing"); err != nil {
		t.Fatalf("consume: %v", err)
	}
	wallet, err := walletService.GetOrCreateWithStarterBonus(ctx, 3)
	if err != nil || wallet.BalanceCredits != 5 {
		t.Fatalf("expected balance 5, got %+v err=%v", wallet, err)
	}
}

func TestAICreditEstimatorConvertsTokens(t *testing.T) {
	estimate := NewAICreditEstimator().EstimateFromTokens(1500, 500)
	if estimate.NexusCoins != 2 {
		t.Fatalf("expected 2 coins, got %d", estimate.NexusCoins)
	}
}

func TestEstimateAICreditCostKnownActions(t *testing.T) {
	if got := EstimateAICreditCost("battle_standard_3_rounds", nil); got != 250 {
		t.Fatalf("expected 250, got %d", got)
	}
	if got := EstimateAICreditCost("roleplay_short_action", nil); got != 25 {
		t.Fatalf("expected 25, got %d", got)
	}
}

func TestBillingServiceEstimateUsageActionType(t *testing.T) {
	t.Setenv("DEFAULT_STARTER_BONUS_CREDITS", "500")
	billing := newTestBillingStack(t)
	payload, err := billing.EstimateUsage(
		context.Background(),
		11,
		"battle_standard_3_rounds",
		"battle_ia",
		0,
		"openai",
		"gpt-test",
		map[string]any{"rounds": 3},
	)
	if err != nil {
		t.Fatalf("estimate usage: %v", err)
	}
	estimate, ok := payload["estimate"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if estimate["estimatedCredits"] != int64(250) {
		t.Fatalf("expected 250 credits, got %+v", estimate["estimatedCredits"])
	}
}

func TestAIOrchestratorModes(t *testing.T) {
	t.Setenv("AI_PLATFORM_MODE", "mock")
	t.Setenv("AI_MOCK_ENABLED", "true")
	orchestrator := NewAIOrchestrator(NewWalletService(newMemoryWalletRepo(), NewAICreditEstimator()), NewAICreditEstimator(), nil)
	if orchestrator.ResolveMode("byok", "key") != AIOrchestratorModeBYOK {
		t.Fatal("expected byok")
	}
	if orchestrator.ResolveMode("platform", "") != AIOrchestratorModeMock {
		t.Fatal("expected mock when platform mode is mock and no client key")
	}
	plan := orchestrator.BuildExecutionPlan("platform", "client-key", "openai", "gpt-test", 1200, 800, "usage:test")
	if !plan.RequiresWallet {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func TestMapBillingErrorPaymentRequired(t *testing.T) {
	mapped := MapBillingError(fmt.Errorf("insufficient credits"))
	billingErr, ok := AsBillingError(mapped)
	if !ok || billingErr.Code != BillingErrorPaymentRequired {
		t.Fatalf("unexpected mapped error: %+v", mapped)
	}
}
