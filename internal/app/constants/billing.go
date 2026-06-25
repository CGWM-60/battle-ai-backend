package constants

const (
	BillingCurrencyNexusCoin = "NEXUS_COIN"

	BillingPlatformMock    = "mock"
	BillingPlatformIOS     = "ios"
	BillingPlatformAndroid = "android"
	BillingPlatformAll     = "all"

	BillingProductTypeConsumable     = "consumable"
	BillingProductTypeSubscription   = "subscription"
	BillingProductTypeOneTimeUnlock  = "one_time_unlock"
	BillingProductStatusActive      = "active"
	BillingProductStatusInactive    = "inactive"

	BillingTransactionStatusPending  = "pending"
	BillingTransactionStatusVerified = "verified"
	BillingTransactionStatusFailed   = "failed"
	BillingTransactionStatusRefunded = "refunded"

	BillingSubscriptionStatusActive    = "active"
	BillingSubscriptionStatusExpired   = "expired"
	BillingSubscriptionStatusCancelled = "cancelled"
	BillingSubscriptionStatusGrace     = "grace"

	BillingLedgerEntryCredit            = "credit"
	BillingLedgerEntryDebit             = "debit"
	BillingLedgerEntryStarterBonus      = "starter_bonus"
	BillingLedgerEntryPurchase          = "purchase"
	BillingLedgerEntrySubscriptionGrant = "subscription_grant"
	BillingLedgerEntryConsume           = "consume"
	BillingLedgerEntryRefund            = "refund"

	BillingEntitlementStatusActive  = "active"
	BillingEntitlementStatusRevoked = "revoked"
	BillingEntitlementStatusExpired = "expired"

	BillingEntitlementSourcePurchase     = "purchase"
	BillingEntitlementSourceSubscription = "subscription"
	BillingEntitlementSourceStarterBonus = "starter_bonus"
	BillingEntitlementSourceAdmin        = "admin"

	BillingReferenceStoreTransaction = "store_transaction"
	BillingReferenceAIUsage          = "ai_usage"
	BillingReferenceSubscription     = "subscription"
	BillingReferenceManual           = "manual"

	BillingSourceBYOK          = "byok"
	BillingSourcePlatform      = "platform"
	BillingSourcePlatformKey   = "platform_key"
	BillingSourceMock          = "mock"
	BillingSourceClientKey     = "client_key"

	BillingEntitlementPremiumMonthly       = "premium_monthly"
	BillingEntitlementAnimaCompanionPremium = "anima_companion_premium"
)