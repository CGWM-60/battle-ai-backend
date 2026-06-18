package constants

import "strings"

const (
	TierFree      = "free"
	TierCommon    = "common"
	TierUncommon  = "uncommon"
	TierRare      = "rare"
	TierEpic      = "epic"
	TierLegendary = "legendary"
	TierNexus     = "nexus"

	// Legacy subscription slugs kept for backward compatibility.
	TierNexusLight = "nexus_light"
	TierNexusPlus  = "nexus_plus"
	TierNexusPro   = "nexus_pro"
)

var tierRank = map[string]int{
	TierFree:      0,
	TierCommon:    1,
	TierUncommon:  2,
	TierRare:      3,
	TierEpic:      4,
	TierLegendary: 5,
	TierNexus:     6,
}

// NormalizeTier returns a canonical tier slug.
func NormalizeTier(tier string) string {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "", "free":
		return TierFree
	case "common":
		return TierCommon
	case "uncommon", "light", "nexus_light", "nexus-light", "starter":
		return TierUncommon
	case "rare", "plus", "nexus_plus", "nexus-plus":
		return TierRare
	case "epic", "pro", "nexus_pro", "nexus-pro", "elite":
		return TierEpic
	case "legendary":
		return TierLegendary
	case "nexus":
		return TierNexus
	default:
		return strings.ToLower(strings.TrimSpace(tier))
	}
}

// TierRank returns the numeric rank of a tier (higher = better).
func TierRank(tier string) int {
	normalized := NormalizeTier(tier)
	if rank, ok := tierRank[normalized]; ok {
		return rank
	}
	return 0
}

// CompareTiers compares two tiers.
// Returns -1 if a < b, 0 if equal, 1 if a > b.
func CompareTiers(a, b string) int {
	rankA := TierRank(a)
	rankB := TierRank(b)
	switch {
	case rankA < rankB:
		return -1
	case rankA > rankB:
		return 1
	default:
		return 0
	}
}

// IsTierAtLeast reports whether userTier meets or exceeds requiredTier.
func IsTierAtLeast(userTier, requiredTier string) bool {
	return TierRank(userTier) >= TierRank(requiredTier)
}

// MaxTier returns the higher-ranked tier between a and b.
func MaxTier(a, b string) string {
	if CompareTiers(a, b) >= 0 {
		return NormalizeTier(a)
	}
	return NormalizeTier(b)
}

// SubscriptionTierFromProductSKU maps a subscription SKU to its tier slug.
func SubscriptionTierFromProductSKU(sku string) string {
	switch strings.ToLower(strings.TrimSpace(sku)) {
	case "nexus_light_monthly":
		return TierUncommon
	case "nexus_plus_monthly":
		return TierRare
	case "nexus_pro_monthly":
		return TierEpic
	default:
		return NormalizeTier(sku)
	}
}

// AllSubscriptionTiers returns subscription tiers ordered from lowest to highest.
func AllSubscriptionTiers() []string {
	return []string{TierUncommon, TierRare, TierEpic}
}
