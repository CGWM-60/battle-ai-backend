package service

import (
	"context"
	"fmt"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/repository"
)

type storeProductStore interface {
	GetBySlug(ctx context.Context, slug string) (*models.StoreProduct, error)
	ListActive(ctx context.Context, productType string) ([]models.StoreProduct, error)
	Count(ctx context.Context) (int64, error)
	Create(ctx context.Context, product *models.StoreProduct) error
}

type StoreProductService struct {
	products storeProductStore
}

func NewStoreProductService(products *repository.StoreProductRepository) *StoreProductService {
	return &StoreProductService{products: products}
}

func (s *StoreProductService) EnsureDefaultMockProducts(ctx context.Context) error {
	if s == nil || s.products == nil {
		return fmt.Errorf("store product service unavailable")
	}
	count, err := s.products.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaults := fallbackMockStoreProducts()
	for index := range defaults {
		if err := s.products.Create(ctx, &defaults[index]); err != nil {
			return err
		}
	}
	return nil
}

func fallbackMockStoreProducts() []models.StoreProduct {
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
			Currency:        "EUR",
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
			Currency:        "EUR",
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
			Currency:        "EUR",
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
			Currency:        "EUR",
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
			Currency:        "EUR",
			Tier:            constants.TierLegendary,
			EntitlementDays: 45,
			Position:        50,
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
			Currency:              "EUR",
			Badge:                 "PREMIUM",
			Popular:               true,
			Tier:                  constants.TierLegendary,
			Position:              60,
			FeatureEntitlementKey: constants.BillingEntitlementAnimaCompanionPremium,
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
			Currency:                   "EUR",
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
			Currency:                   "EUR",
			Badge:                      "RECOMMANDE",
			Popular:                    true,
			Interval:                   "month",
			Tier:                       constants.TierRare,
			Position:                   120,
			SubscriptionEntitlementKey: constants.TierRare,
			SubscriptionPeriodDays:     30,
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
			Currency:                   "EUR",
			Badge:                      "PRO",
			Interval:                   "month",
			Tier:                       constants.TierEpic,
			Position:                   130,
			SubscriptionEntitlementKey: constants.TierEpic,
			SubscriptionPeriodDays:     30,
		},
	}
}

func (s *StoreProductService) GetActiveBySKU(ctx context.Context, sku string) (*models.StoreProduct, error) {
	return s.GetActiveBySlug(ctx, sku)
}

func (s *StoreProductService) GetActiveBySlug(ctx context.Context, slug string) (*models.StoreProduct, error) {
	product, err := s.products.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if product.Status != constants.BillingProductStatusActive {
		return nil, fmt.Errorf("store product inactive")
	}
	return product, nil
}

func (s *StoreProductService) ListActive(ctx context.Context, productType string) ([]models.StoreProduct, error) {
	return s.products.ListActive(ctx, productType)
}
