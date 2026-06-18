package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/repository"
)

type entitlementStore interface {
	Create(ctx context.Context, entitlement *models.UserEntitlement) error
	ListByUserID(ctx context.Context, userID uint) ([]models.UserEntitlement, error)
	HasActive(ctx context.Context, userID uint, key string, at time.Time) (bool, error)
	GetActiveByKey(ctx context.Context, userID uint, key string, at time.Time) (*models.UserEntitlement, error)
}

type EntitlementService struct {
	entitlements entitlementStore
}

func NewEntitlementService(entitlements *repository.EntitlementRepository) *EntitlementService {
	return &EntitlementService{entitlements: entitlements}
}

func (s *EntitlementService) Grant(ctx context.Context, userID uint, key string, source string, sourceRef string, expiresAt *time.Time) (*models.UserEntitlement, error) {
	if s == nil || s.entitlements == nil {
		return nil, fmt.Errorf("entitlement service unavailable")
	}
	if userID == 0 || key == "" {
		return nil, fmt.Errorf("user id and entitlement key are required")
	}

	entitlement := &models.UserEntitlement{
		UserID:     userID,
		Key:        key,
		Value:      key,
		Source:     source,
		SourceType: source,
		SourceRef:  sourceRef,
		Status:     constants.BillingEntitlementStatusActive,
		StartsAt:   time.Now(),
		ExpiresAt:  expiresAt,
		Active:     true,
	}
	if err := s.entitlements.Create(ctx, entitlement); err != nil {
		return nil, err
	}
	return entitlement, nil
}

func (s *EntitlementService) HasActive(ctx context.Context, userID uint, key string) (bool, error) {
	return s.entitlements.HasActive(ctx, userID, key, time.Now())
}

func (s *EntitlementService) ListByUserID(ctx context.Context, userID uint) ([]models.UserEntitlement, error) {
	return s.entitlements.ListByUserID(ctx, userID)
}

func (s *EntitlementService) GetActiveByKey(ctx context.Context, userID uint, key string) (*models.UserEntitlement, error) {
	return s.entitlements.GetActiveByKey(ctx, userID, key, time.Now())
}

func (s *EntitlementService) HasTierAccess(ctx context.Context, userID uint, requiredTier string) (bool, string, error) {
	required := constants.NormalizeTier(requiredTier)
	requiredRank := constants.TierRank(required)
	if requiredRank <= 0 {
		return true, constants.TierFree, nil
	}

	entitlements, err := s.entitlements.ListByUserID(ctx, userID)
	if err != nil {
		return false, "", err
	}

	currentTier := constants.TierFree
	currentRank := 0
	now := time.Now()
	for _, item := range entitlements {
		if !item.Active || item.Status != constants.BillingEntitlementStatusActive {
			continue
		}
		if item.ExpiresAt != nil && !item.ExpiresAt.After(now) {
			continue
		}
		keyTier := entitlementKeyToTier(item.Key)
		rank := constants.TierRank(keyTier)
		if rank > currentRank {
			currentRank = rank
			currentTier = keyTier
		}
	}

	if currentRank >= requiredRank {
		return true, currentTier, nil
	}
	return false, currentTier, nil
}

func entitlementKeyToTier(key string) string {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if strings.HasPrefix(normalized, "tier:") {
		return constants.NormalizeTier(strings.TrimPrefix(normalized, "tier:"))
	}
	return constants.NormalizeTier(normalized)
}

func tierEntitlementKey(tier string) string {
	return "tier:" + constants.NormalizeTier(tier)
}
