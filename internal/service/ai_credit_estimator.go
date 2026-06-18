package service

import (
	"math"
	"os"
	"strconv"
	"strings"
)

const defaultNexusCoinsPer1000Tokens = int64(1)

// AICreditEstimate = estimation de credits Nexus Coin pour un appel IA.
type AICreditEstimate struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	NexusCoins       int64
	TokensPerCoin    int64
}

// AICreditEstimator convertit des tokens IA en credits wallet.
type AICreditEstimator struct {
	tokensPerCoin int64
}

func NewAICreditEstimator() *AICreditEstimator {
	return &AICreditEstimator{
		tokensPerCoin: aiCreditTokensPerCoin(),
	}
}

func (e *AICreditEstimator) TokensPerCoin() int64 {
	if e == nil || e.tokensPerCoin <= 0 {
		return 1000
	}
	return e.tokensPerCoin
}

func (e *AICreditEstimator) EstimateFromTokens(promptTokens int, completionTokens int) AICreditEstimate {
	total := promptTokens + completionTokens
	if total < 0 {
		total = 0
	}
	return AICreditEstimate{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      total,
		NexusCoins:       e.EstimateCreditsFromTokens(total),
		TokensPerCoin:    e.TokensPerCoin(),
	}
}

func (e *AICreditEstimator) EstimateCreditsFromTokens(totalTokens int) int64 {
	if totalTokens <= 0 {
		return 0
	}
	perCoin := e.TokensPerCoin()
	return int64(math.Ceil(float64(totalTokens) / float64(perCoin)))
}

func (e *AICreditEstimator) EstimateTokensFromCredits(credits int64) int64 {
	if credits <= 0 {
		return 0
	}
	return credits * e.TokensPerCoin()
}

// EstimateAICreditCost returns the fixed credit cost for a known action type.
func EstimateAICreditCost(actionType string, metadata map[string]any) int64 {
	normalized := strings.ToLower(strings.TrimSpace(actionType))
	switch normalized {
	case "battle_express_1_round":
		return 80
	case "battle_standard_3_rounds":
		return 250
	case "battle_advanced_5_rounds":
		return 450
	case "battle_judge":
		return 70
	case "battle_resume":
		return 40
	case "battle_public_arena_extra":
		return 100
	case "roleplay_short_action":
		return 25
	case "roleplay_full_scene":
		return 90
	case "roleplay_coop_scene":
		return 120
	case "roleplay_chapter":
		return 280
	case "roleplay_arc":
		return 750
	case "roleplay_character_generate":
		return 50
	case "anima_simple_chat":
		return 10
	case "anima_emotional_analysis":
		return 40
	case "anima_long_memory":
		return 30
	case "anima_personality_evolution":
		return 120
	case "anima_neural_map":
		return 250
	default:
		if rounds := metadataInt(metadata, "rounds"); rounds > 0 {
			switch {
			case rounds <= 1:
				return 80
			case rounds <= 3:
				return 250
			case rounds <= 5:
				return 450
			default:
				return 450 + int64((rounds-5)*80)
			}
		}
		return 0
	}
}

func metadataInt(metadata map[string]any, key string) int {
	if metadata == nil {
		return 0
	}
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return 0
	}
	switch value := raw.(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float32:
		return int(value)
	case float64:
		return int(value)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func aiCreditTokensPerCoin() int64 {
	raw := strings.TrimSpace(os.Getenv("BILLING_TOKENS_PER_NEXUS_COIN"))
	if raw == "" {
		return 1000
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed <= 0 {
		return 1000
	}
	return parsed
}

func starterBonusNexusCoins() int64 {
	return StarterBonusCredits()
}

func starterBonusIdempotencyKey(userID uint) string {
	return "starter_bonus:user:" + strconv.FormatUint(uint64(userID), 10)
}
