package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"
	"cgwm/battle/internal/repository"
)

const (
	billingSourceClientKey   = "client_key"
	billingSourcePlatformKey = "platform_key"
)

type usageSessionRef struct {
	OwnerID           uint
	SessionMode       string
	BattleSaveID      *uint
	RolePlaySessionID *uint
	BillingSource     string
	ProviderName      string
	ModelName         string
	Feature           string
	SettlementPlan    *AIExecutionPlan
	Orchestrator      *AIOrchestrator
}

func attachUsageRecorder(
	usage *repository.AIUsageRepository,
	ref usageSessionRef,
	ai *provider.Provider,
) *provider.Provider {
	if ai == nil || usage == nil {
		return ai
	}
	return ai.WithUsageRecorder(func(record provider.UsageRecord) {
		commitAIUsageRecord(usage, ref, record)
	})
}

func attachCapturedUsageRecorder(ai *provider.Provider, capture *provider.UsageRecord) *provider.Provider {
	if ai == nil || capture == nil {
		return ai
	}
	return ai.WithUsageRecorder(func(record provider.UsageRecord) {
		*capture = record
	})
}

func commitAIUsageRecord(usage *repository.AIUsageRepository, ref usageSessionRef, record provider.UsageRecord) {
	if usage == nil {
		return
	}
	ctx := context.Background()
	price := usagePricing(ref.ProviderName)
	costMicros := estimateUsageCostMicros(record.PromptTokens, record.CompletionTokens, price)
	usageRecord := &models.AIUsageRecord{
		OwnerID:              ref.OwnerID,
		SessionMode:          defaultString(ref.SessionMode, constants.ModeBattleIA),
		BattleSaveID:         ref.BattleSaveID,
		RolePlaySessionID:    ref.RolePlaySessionID,
		BillingSource:        defaultString(ref.BillingSource, billingSourceClientKey),
		ProviderName:         normalizeProviderName(ref.ProviderName),
		ProviderHost:         record.ProviderHost,
		ModelName:            defaultString(record.Model, ref.ModelName),
		Operation:            record.Operation,
		Phase:                record.Phase,
		Round:                record.Round,
		ActorName:            record.ActorName,
		PromptTokens:         record.PromptTokens,
		CompletionTokens:     record.CompletionTokens,
		TotalTokens:          record.TotalTokens,
		InputChars:           record.InputChars,
		OutputChars:          record.OutputChars,
		Stream:               record.Stream,
		Fallback:             record.Fallback,
		Estimated:            record.Estimated,
		Currency:             "USD",
		InputUSDPer1MToken:   price.InputUSDPer1M,
		OutputUSDPer1MToken:  price.OutputUSDPer1M,
		EstimatedCostMicros:  costMicros,
		PricingConfiguration: price.Source,
	}
	if err := usage.Create(ctx, usageRecord); err != nil {
		log.Printf("[ai-usage] create failed mode=%s provider=%s model=%s err=%v", ref.SessionMode, ref.ProviderName, ref.ModelName, err)
		return
	}
	if ref.BattleSaveID != nil {
		if err := usage.IncrementBattleUsage(ctx, *ref.BattleSaveID, record.PromptTokens, record.CompletionTokens, record.TotalTokens, costMicros); err != nil {
			log.Printf("[ai-usage] increment battle failed battle_id=%d err=%v", *ref.BattleSaveID, err)
		}
	}
	if ref.RolePlaySessionID != nil {
		if err := usage.IncrementRolePlayUsage(ctx, *ref.RolePlaySessionID, record.PromptTokens, record.CompletionTokens, record.TotalTokens, costMicros); err != nil {
			log.Printf("[ai-usage] increment roleplay failed session_id=%d err=%v", *ref.RolePlaySessionID, err)
		}
	}
	if ref.Orchestrator != nil && ref.SettlementPlan != nil {
		referenceID := ""
		if usageRecord.Id != 0 {
			referenceID = fmt.Sprintf("ai_usage_record:%d", usageRecord.Id)
		}
		if _, err := ref.Orchestrator.SettleUsage(
			ctx,
			ref.OwnerID,
			*ref.SettlementPlan,
			record.PromptTokens,
			record.CompletionTokens,
			defaultString(ref.Feature, ref.SessionMode),
			referenceID,
		); err != nil {
			log.Printf("[ai-usage] billing settlement failed mode=%s user_id=%d err=%v", ref.SessionMode, ref.OwnerID, err)
		}
	}
}

type usagePrice struct {
	InputUSDPer1M  float64
	OutputUSDPer1M float64
	Source         string
}

func usagePricing(providerName string) usagePrice {
	normalized := strings.ToUpper(strings.ReplaceAll(normalizeProviderName(providerName), "-", "_"))
	if normalized == "" {
		normalized = "DEFAULT"
	}
	inputKey := fmt.Sprintf("AI_PRICE_%s_INPUT_USD_PER_1M", normalized)
	outputKey := fmt.Sprintf("AI_PRICE_%s_OUTPUT_USD_PER_1M", normalized)
	input, inputOK := envFloat(inputKey)
	output, outputOK := envFloat(outputKey)
	if inputOK || outputOK {
		return usagePrice{InputUSDPer1M: input, OutputUSDPer1M: output, Source: inputKey + "," + outputKey}
	}
	input, inputOK = envFloat("AI_PRICE_DEFAULT_INPUT_USD_PER_1M")
	output, outputOK = envFloat("AI_PRICE_DEFAULT_OUTPUT_USD_PER_1M")
	if inputOK || outputOK {
		return usagePrice{InputUSDPer1M: input, OutputUSDPer1M: output, Source: "AI_PRICE_DEFAULT_INPUT_USD_PER_1M,AI_PRICE_DEFAULT_OUTPUT_USD_PER_1M"}
	}
	return usagePrice{Source: "unset"}
}

func estimateUsageCostMicros(promptTokens int, completionTokens int, price usagePrice) int64 {
	if price.InputUSDPer1M <= 0 && price.OutputUSDPer1M <= 0 {
		return 0
	}
	cost := float64(promptTokens)*price.InputUSDPer1M + float64(completionTokens)*price.OutputUSDPer1M
	return int64(math.Round(cost))
}

func envFloat(key string) (float64, bool) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 {
		return 0, false
	}
	return parsed, true
}
