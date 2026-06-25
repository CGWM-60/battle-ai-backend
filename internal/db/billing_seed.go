package db

import (
	"errors"
	"strings"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

func seedMockStoreProducts(db *gorm.DB) error {
	mode := strings.ToLower(strings.TrimSpace(getEnv("BILLING_MODE", "mock")))
	if mode != "mock" {
		return nil
	}

	currency := strings.ToUpper(strings.TrimSpace(getEnv("BILLING_CURRENCY", "EUR")))
	if currency == "" {
		currency = "EUR"
	}

	seeds := defaultMockStoreProducts(currency)

	for _, seed := range seeds {
		if err := upsertStoreProduct(db, seed); err != nil {
			return err
		}
	}
	return nil
}

func defaultMockStoreProducts(currency string) []models.StoreProduct {
	return []models.StoreProduct{
		{
			Slug:            "starter_ai",
			StoreProductID:  "mock.starter_ai",
			Platform:        constants.BillingPlatformMock,
			ProductType:     constants.BillingProductTypeConsumable,
			Name:            "Pack Starter IA",
			Description:     "Ideal pour decouvrir les experiences IA de Battle IA.",
			Status:          constants.BillingProductStatusActive,
			NexusCoinsGrant: 1200,
			PriceCents:      199,
			Currency:        currency,
			Tier:            constants.TierUncommon,
			EntitlementDays: 7,
			Position:        10,
		},
		{
			Slug:            "player_pack",
			StoreProductID:  "mock.player_pack",
			Platform:        constants.BillingPlatformMock,
			ProductType:     constants.BillingProductTypeConsumable,
			Name:            "Pack Joueur",
			Description:     "Pour les sessions RP et battles regulieres.",
			Status:          constants.BillingProductStatusActive,
			NexusCoinsGrant: 3500,
			PriceCents:      499,
			Currency:        currency,
			Tier:            constants.TierUncommon,
			EntitlementDays: 15,
			Position:        20,
		},
		{
			Slug:            "adventurer_pack",
			StoreProductID:  "mock.adventurer_pack",
			Platform:        constants.BillingPlatformMock,
			ProductType:     constants.BillingProductTypeConsumable,
			Name:            "Pack Aventurier",
			Description:     "Volume confortable pour explorer le monde IA et le roleplay.",
			Status:          constants.BillingProductStatusActive,
			NexusCoinsGrant: 8000,
			PriceCents:      999,
			Currency:        currency,
			Badge:           "RECOMMANDE",
			Popular:         true,
			Tier:            constants.TierRare,
			EntitlementDays: 30,
			Position:        30,
		},
		{
			Slug:            "strategist_pack",
			StoreProductID:  "mock.strategist_pack",
			Platform:        constants.BillingPlatformMock,
			ProductType:     constants.BillingProductTypeConsumable,
			Name:            "Pack Stratege",
			Description:     "Credits eleves pour les joueurs qui pilotent plusieurs modes IA.",
			Status:          constants.BillingProductStatusActive,
			NexusCoinsGrant: 18000,
			PriceCents:      1999,
			Currency:        currency,
			Tier:            constants.TierEpic,
			EntitlementDays: 30,
			Position:        40,
		},
		{
			Slug:            "creator_pack",
			StoreProductID:  "mock.creator_pack",
			Platform:        constants.BillingPlatformMock,
			ProductType:     constants.BillingProductTypeConsumable,
			Name:            "Pack Createur",
			Description:     "Volume maximal pour createurs de contenu et power users.",
			Status:          constants.BillingProductStatusActive,
			NexusCoinsGrant: 55000,
			PriceCents:      4999,
			Currency:        currency,
			Tier:            constants.TierLegendary,
			EntitlementDays: 45,
			Position:        50,
		},
		{
			Slug:                       "nexus_light_monthly",
			StoreProductID:             "mock.nexus_light_monthly",
			Platform:                   constants.BillingPlatformMock,
			ProductType:                constants.BillingProductTypeSubscription,
			Name:                       "Nexus Light",
			Description:                "Credits mensuels et acces prioritaire aux nouveautes IA.",
			Status:                     constants.BillingProductStatusActive,
			NexusCoinsGrant:            4000,
			PriceCents:                 499,
			Currency:                   currency,
			Interval:                   "month",
			Tier:                       constants.TierUncommon,
			Position:                   110,
			SubscriptionEntitlementKey: constants.TierUncommon,
			SubscriptionPeriodDays:     30,
		},
		{
			Slug:                       "nexus_plus_monthly",
			StoreProductID:             "mock.nexus_plus_monthly",
			Platform:                   constants.BillingPlatformMock,
			ProductType:                constants.BillingProductTypeSubscription,
			Name:                       "Nexus Plus",
			Description:                "Volume IA confortable avec bonus mensuel et tier Plus.",
			Status:                     constants.BillingProductStatusActive,
			NexusCoinsGrant:            10000,
			PriceCents:                 999,
			Currency:                   currency,
			Badge:                      "RECOMMANDE",
			Popular:                    true,
			Interval:                   "month",
			Tier:                       constants.TierRare,
			Position:                   120,
			SubscriptionEntitlementKey: constants.TierRare,
			SubscriptionPeriodDays:     30,
		},
		{
			Slug:                  "anima_companion_premium",
			StoreProductID:        "mock.anima_companion_premium",
			Platform:              constants.BillingPlatformMock,
			ProductType:           constants.BillingProductTypeOneTimeUnlock,
			Name:                  "Amina Compagnon Global",
			Description:           "Debloque Amina dans toute l'application avec des interactions contextuelles.",
			Status:                constants.BillingProductStatusActive,
			NexusCoinsGrant:       0,
			PriceCents:            499,
			Currency:              currency,
			Badge:                 "PREMIUM",
			Popular:               true,
			Tier:                  constants.TierLegendary,
			Position:              60,
			FeatureEntitlementKey: constants.BillingEntitlementAnimaCompanionPremium,
		},
		{
			Slug:                       "nexus_pro_monthly",
			StoreProductID:             "mock.nexus_pro_monthly",
			Platform:                   constants.BillingPlatformMock,
			ProductType:                constants.BillingProductTypeSubscription,
			Name:                       "Nexus Pro",
			Description:                "Volume IA maximal, tier Pro et meilleur rapport mensuel.",
			Status:                     constants.BillingProductStatusActive,
			NexusCoinsGrant:            26000,
			PriceCents:                 1999,
			Currency:                   currency,
			Badge:                      "PRO",
			Interval:                   "month",
			Tier:                       constants.TierEpic,
			Position:                   130,
			SubscriptionEntitlementKey: constants.TierEpic,
			SubscriptionPeriodDays:     30,
		},
	}
}

func upsertStoreProduct(db *gorm.DB, seed models.StoreProduct) error {
	var existing models.StoreProduct
	err := db.Unscoped().Where("slug = ?", seed.Slug).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&seed).Error
	}
	if err != nil {
		return err
	}

	updates := map[string]any{
		"store_product_id":             seed.StoreProductID,
		"platform":                     seed.Platform,
		"product_type":                 seed.ProductType,
		"name":                         seed.Name,
		"description":                  seed.Description,
		"status":                       seed.Status,
		"nexus_coins_grant":            seed.NexusCoinsGrant,
		"price_cents":                  seed.PriceCents,
		"currency":                     seed.Currency,
		"badge":                        seed.Badge,
		"popular":                      seed.Popular,
		"interval":                     seed.Interval,
		"tier":                         seed.Tier,
		"entitlement_days":             seed.EntitlementDays,
		"position":                     seed.Position,
		"subscription_entitlement_key": seed.SubscriptionEntitlementKey,
		"subscription_period_days":     seed.SubscriptionPeriodDays,
		"feature_entitlement_key":      seed.FeatureEntitlementKey,
		"deleted_at":                   nil,
	}
	return db.Unscoped().Model(&existing).Updates(updates).Error
}
