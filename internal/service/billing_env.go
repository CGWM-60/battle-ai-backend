package service

import (
	"os"
	"strconv"
	"strings"
)

func envString(primary string, legacy string, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(primary)); value != "" {
		return value
	}
	if legacy != "" {
		if value := strings.TrimSpace(os.Getenv(legacy)); value != "" {
			return value
		}
	}
	return defaultValue
}

func envInt64(primary string, legacy string, defaultValue int64) int64 {
	raw := envString(primary, legacy, "")
	if raw == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed < 0 {
		return defaultValue
	}
	return parsed
}

func envBool(key string, defaultValue bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return defaultValue
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func BillingModeEnv() string {
	return strings.ToLower(envString("BILLING_MODE", "", "live"))
}

func AIPlatformMode() string {
	return strings.ToLower(envString("AI_PLATFORM_MODE", "", "live"))
}

func StoreVerifierMode() string {
	return strings.ToLower(envString("STORE_VERIFIER", "", "mock"))
}

func AICreditsEnabled() bool {
	return envBool("AI_CREDITS_ENABLED", true)
}

func AIMockEnabled() bool {
	return envBool("AI_MOCK_ENABLED", false)
}

func StarterBonusCredits() int64 {
	return envInt64("DEFAULT_STARTER_BONUS_CREDITS", "BILLING_STARTER_BONUS_NEXUS_COINS", 500)
}

func DefaultFreeDailyCredits() int64 {
	return envInt64("DEFAULT_FREE_DAILY_CREDITS", "", 100)
}

func NewStoreVerifierFromEnv() StoreVerifier {
	switch StoreVerifierMode() {
	case "mock", "":
		return NewMockStoreVerifier()
	default:
		return NewMockStoreVerifier()
	}
}
