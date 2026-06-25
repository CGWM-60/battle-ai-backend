package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	BillingModeOwnKey        = "own_key"
	BillingModePlatform      = "platform"
	StoreProviderMock        = "mock"
	StoreProductStatusActive = "active"

	StoreProductTypeCreditPack   = "consumable"
	StoreProductTypeSubscription = "subscription"

	StoreTransactionStatusCompleted = "verified"
	StoreTransactionStatusPending   = "pending"
	StoreTransactionStatusFailed    = "failed"

	SubscriptionStatusActive    = "active"
	SubscriptionStatusExpired   = "expired"
	SubscriptionStatusCancelled = "cancelled"

	WalletLedgerTypeBonus        = "starter_bonus"
	WalletLedgerTypePurchase     = "purchase"
	WalletLedgerTypeConsumption  = "consume"
	WalletLedgerTypeSubscription = "subscription_grant"
	WalletLedgerTypeRefund       = "refund"

	EntitlementSourcePurchase     = "purchase"
	EntitlementSourceSubscription = "subscription"
	EntitlementSourceStarterBonus = "starter_bonus"
	EntitlementSourceAdmin        = "admin"
)

// UserAIWallet = solde Nexus Coin / credits IA d'un joueur.
type UserAIWallet struct {
	Id        uint `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID uint  `gorm:"uniqueIndex" json:"userId"`
	User   Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	BalanceCredits      int64  `gorm:"column:balance;index" json:"balanceCredits"`
	Currency            string `gorm:"size:16" json:"currency"`
	StarterBonusGranted bool   `json:"starterBonusGranted"`

	MonthlyAllowance   int64      `json:"monthlyAllowance"`
	UsedThisMonth      int64      `json:"usedThisMonth"`
	BillingMode        string     `gorm:"size:32;index" json:"billingMode"`
	SubscriptionTier   string     `gorm:"size:64;index" json:"subscriptionTier"`
	SubscriptionActive bool       `gorm:"index" json:"subscriptionActive"`
	LowBalance         bool       `json:"lowBalance"`
	AllowanceResetAt   *time.Time `gorm:"index" json:"allowanceResetAt,omitempty"`
}

func (UserAIWallet) TableName() string { return "user_ai_wallets" }

// AIWalletLedger = journal immuable des mouvements wallet.
type AIWalletLedger struct {
	Id        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"createdAt"`

	WalletID uint         `gorm:"index" json:"walletId"`
	Wallet   UserAIWallet `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	UserID   uint         `gorm:"index" json:"userId"`
	User     Users        `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	Amount         int64          `json:"amount"`
	BalanceAfter   int64          `json:"balanceAfter"`
	EntryType      string         `gorm:"size:32;index" json:"type"`
	Currency       string         `gorm:"size:16" json:"currency"`
	ReferenceID    string         `gorm:"size:190;uniqueIndex" json:"referenceId"`
	Description    string         `gorm:"size:255" json:"description"`
	Feature        string         `gorm:"size:80;index" json:"feature"`
	IdempotencyKey string         `gorm:"size:190;index" json:"idempotencyKey"`
	ReferenceType  string         `gorm:"size:64;index" json:"referenceType"`
	Metadata       datatypes.JSON `gorm:"type:json" json:"metadata,omitempty"`
}

func (AIWalletLedger) TableName() string { return "ai_wallet_ledgers" }

// StoreProduct = catalogue boutique (packs consommables ou abonnements).
type StoreProduct struct {
	Id        uint `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Slug           string `gorm:"size:80;uniqueIndex" json:"id"`
	StoreProductID string `gorm:"size:120;index" json:"storeProductId"`
	Platform       string `gorm:"size:32;index" json:"platform"`
	ProductType    string `gorm:"size:32;index" json:"type"`
	Name           string `gorm:"size:120" json:"name"`
	Description    string `gorm:"type:text" json:"description"`
	Status         string `gorm:"size:32;index" json:"status"`

	NexusCoinsGrant int64  `gorm:"column:nexus_coins_grant" json:"credits"`
	PriceCents      int64  `json:"priceCents"`
	Currency        string `gorm:"size:8" json:"currency"`
	Badge           string `gorm:"size:64" json:"badge"`
	Popular         bool   `json:"popular"`
	Interval        string `gorm:"size:32" json:"interval,omitempty"`
	Tier            string `gorm:"size:64;index" json:"tier,omitempty"`
	EntitlementDays int    `json:"entitlementDays,omitempty"`
	Position        int    `gorm:"index" json:"position"`

	SubscriptionEntitlementKey string         `gorm:"size:80" json:"subscriptionEntitlementKey,omitempty"`
	SubscriptionPeriodDays     int            `json:"subscriptionPeriodDays,omitempty"`
	FeatureEntitlementKey      string         `gorm:"size:80" json:"featureEntitlementKey,omitempty"`
	Metadata                   datatypes.JSON `gorm:"type:json" json:"metadata,omitempty"`
}

func (StoreProduct) TableName() string { return "store_products" }

func (p *StoreProduct) SKU() string {
	if p == nil {
		return ""
	}
	if p.Slug != "" {
		return p.Slug
	}
	return p.StoreProductID
}

func (p *StoreProduct) Credits() int64 {
	if p == nil {
		return 0
	}
	return p.NexusCoinsGrant
}

// StoreTransaction = achat verifie (mock, mobile stores, etc.).
type StoreTransaction struct {
	Id        uint `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time

	UserID uint  `gorm:"index" json:"userId"`
	User   Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	StoreProductID uint         `gorm:"index" json:"storeProductId"`
	StoreProduct   StoreProduct `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"product,omitempty"`

	ProductSKU      string         `gorm:"size:80;index" json:"productSku"`
	Platform        string         `gorm:"size:32;index" json:"platform"`
	Provider        string         `gorm:"size:32;index" json:"provider"`
	ProviderRef     string         `gorm:"size:190;uniqueIndex" json:"providerRef"`
	StoreReceiptID  string         `gorm:"size:190;index" json:"storeReceiptId"`
	IdempotencyKey  string         `gorm:"size:190;index" json:"idempotencyKey"`
	Status          string         `gorm:"size:32;index" json:"status"`
	NexusCoinsGrant int64          `json:"nexusCoinsGrant"`
	VerifiedAt      *time.Time     `gorm:"index" json:"verifiedAt,omitempty"`
	RawReceipt      datatypes.JSON `gorm:"type:json" json:"rawReceipt,omitempty"`

	AmountCents    int64          `json:"amountCents"`
	CreditsGranted int64          `json:"creditsGranted"`
	CheckoutURL    string         `gorm:"size:512" json:"checkoutUrl,omitempty"`
	CompletedAt    *time.Time     `gorm:"index" json:"completedAt,omitempty"`
	Metadata       datatypes.JSON `gorm:"type:json" json:"metadata,omitempty"`
}

func (StoreTransaction) TableName() string { return "store_transactions" }

// UserSubscription = abonnement actif d'un joueur.
type UserSubscription struct {
	Id        uint `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID uint  `gorm:"index" json:"userId"`
	User   Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	StoreProductID uint         `gorm:"index" json:"storeProductId"`
	StoreProduct   StoreProduct `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"product,omitempty"`

	ProductSKU             string `gorm:"size:80;index" json:"productSku"`
	Platform               string `gorm:"size:32;index" json:"platform"`
	Provider               string `gorm:"size:32;index" json:"provider"`
	StoreSubscriptionID    string `gorm:"size:190;index" json:"storeSubscriptionId"`
	ProviderSubscriptionID string `gorm:"size:190;index" json:"providerSubscriptionId"`
	Status                 string `gorm:"size:32;index" json:"status"`
	AutoRenew              bool   `json:"autoRenew"`

	Tier              string `gorm:"size:64;index" json:"tier"`
	MonthlyCredits    int64  `json:"monthlyCredits"`
	CancelAtPeriodEnd bool   `json:"cancelAtPeriodEnd"`

	CurrentPeriodStart *time.Time     `gorm:"index" json:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd   *time.Time     `gorm:"index" json:"currentPeriodEnd,omitempty"`
	RenewsAt           *time.Time     `gorm:"index" json:"renewsAt,omitempty"`
	RawReceipt         datatypes.JSON `gorm:"type:json" json:"rawReceipt,omitempty"`
	Metadata           datatypes.JSON `gorm:"type:json" json:"metadata,omitempty"`
}

func (UserSubscription) TableName() string { return "user_subscriptions" }

// UserEntitlement = droit fonctionnel accorde a un joueur.
type UserEntitlement struct {
	Id        uint `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID uint  `gorm:"index" json:"userId"`
	User   Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	Key        string         `gorm:"size:80;index" json:"key"`
	SourceType string         `gorm:"size:32;index" json:"sourceType"`
	Source     string         `gorm:"size:32;index" json:"source"`
	SourceID   *uint          `json:"sourceId,omitempty"`
	SourceRef  string         `gorm:"size:190;index" json:"sourceRef"`
	Status     string         `gorm:"size:32;index" json:"status"`
	StartsAt   time.Time      `gorm:"index" json:"startsAt"`
	ExpiresAt  *time.Time     `gorm:"index" json:"expiresAt,omitempty"`
	Value      string         `gorm:"size:255" json:"value"`
	Active     bool           `gorm:"index" json:"active"`
	Metadata   datatypes.JSON `gorm:"type:json" json:"metadata,omitempty"`
}

func (UserEntitlement) TableName() string { return "user_entitlements" }
