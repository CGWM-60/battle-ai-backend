package router

import (
	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	nexuscache "cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/provider"
	"cgwm/battle/internal/repository"
	"cgwm/battle/internal/scenarios"
	"cgwm/battle/internal/service"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	responseCacheOnce sync.Once
	responseCache     *nexuscache.ResponseCache
)

type battleRequest struct {
	Provider1            string `form:"provider1" json:"provider1"`
	Provider2            string `form:"provider2" json:"provider2"`
	IAKey1               string `form:"iaKey1" json:"iaKey1"`
	IAKey2               string `form:"iaKey2" json:"iaKey2"`
	IAModels             string `form:"iaModels" json:"iaModels"`
	IAModels2            string `form:"iaModels2" json:"iaModels2"`
	IA1ProfileID         uint   `form:"ia1ProfileId" json:"ia1ProfileId"`
	IA2ProfileID         uint   `form:"ia2ProfileId" json:"ia2ProfileId"`
	IA1Name              string `form:"ia1Name" json:"ia1Name"`
	IA1Personality       string `form:"ia1Personality" json:"ia1Personality"`
	IA1Mindset           string `form:"ia1Mindset" json:"ia1Mindset"`
	IA1Style             string `form:"ia1Style" json:"ia1Style"`
	IA1Goal              string `form:"ia1Goal" json:"ia1Goal"`
	IA1Weakness          string `form:"ia1Weakness" json:"ia1Weakness"`
	IA2Name              string `form:"ia2Name" json:"ia2Name"`
	IA2Personality       string `form:"ia2Personality" json:"ia2Personality"`
	IA2Mindset           string `form:"ia2Mindset" json:"ia2Mindset"`
	IA2Style             string `form:"ia2Style" json:"ia2Style"`
	IA2Goal              string `form:"ia2Goal" json:"ia2Goal"`
	IA2Weakness          string `form:"ia2Weakness" json:"ia2Weakness"`
	Question             string `form:"question" json:"question"`
	QuestID              uint   `form:"questId" json:"questId"`
	Title                string `form:"title" json:"title"`
	Visibility           string `form:"visibility" json:"visibility"`
	TotalRounds          int    `form:"totalRounds" json:"totalRounds"`
	RoundDurationSeconds int    `form:"roundDurationSeconds" json:"roundDurationSeconds"`
	PublicVote           bool   `form:"publicVote" json:"publicVote"`
	BillingMode          string `form:"billingMode" json:"billingMode"`
}

type iaProfileRequest struct {
	Name         string `form:"name" json:"name"`
	ProviderName string `form:"providerName" json:"providerName"`
	ModelName    string `form:"modelName" json:"modelName"`
	Personality  string `form:"personality" json:"personality"`
	Mindset      string `form:"mindset" json:"mindset"`
	Style        string `form:"style" json:"style"`
	Goal         string `form:"goal" json:"goal"`
	Weakness     string `form:"weakness" json:"weakness"`
}

type aiProviderTestRequest struct {
	ProviderName string `form:"providerName" json:"providerName"`
	Provider     string `form:"provider" json:"provider"`
	ModelName    string `form:"modelName" json:"modelName"`
	Model        string `form:"model" json:"model"`
	APIKey       string `form:"apiKey" json:"apiKey"`
	Prompt       string `form:"prompt" json:"prompt"`
}

type aiProviderGenerateRequest struct {
	ProviderName string `form:"providerName" json:"providerName"`
	Provider     string `form:"provider" json:"provider"`
	ModelName    string `form:"modelName" json:"modelName"`
	Model        string `form:"model" json:"model"`
	APIKey       string `form:"apiKey" json:"apiKey"`
	SystemPrompt string `form:"systemPrompt" json:"systemPrompt"`
	Prompt       string `form:"prompt" json:"prompt"`
	MaxChars     int    `form:"maxChars" json:"maxChars"`
}

type battleQuestRequest struct {
	Slug     string         `form:"slug" json:"slug"`
	Title    string         `form:"title" json:"title"`
	Content  string         `form:"content" json:"content"`
	Level    string         `form:"level" json:"level"`
	Point    int            `form:"point" json:"point"`
	Theme    string         `form:"theme" json:"theme"`
	Xp       int            `form:"xp" json:"xp"`
	Coin     int            `form:"coin" json:"coin"`
	Mode     string         `form:"mode" json:"mode"`
	Source   string         `form:"source" json:"source"`
	Status   string         `form:"status" json:"status"`
	Metadata map[string]any `form:"-" json:"metadata"`
}

type rolePlayQuestRequest struct {
	Slug     string                    `form:"slug" json:"slug"`
	Title    string                    `form:"title" json:"title"`
	Summary  string                    `form:"summary" json:"summary"`
	Prompt   string                    `form:"prompt" json:"prompt"`
	Theme    string                    `form:"theme" json:"theme"`
	Level    string                    `form:"level" json:"level"`
	Xp       int                       `form:"xp" json:"xp"`
	Coin     int                       `form:"coin" json:"coin"`
	Source   string                    `form:"source" json:"source"`
	Status   string                    `form:"status" json:"status"`
	Metadata map[string]any            `form:"-" json:"metadata"`
	Arcs     []rolePlayQuestArcRequest `form:"-" json:"arcs"`
}

type rolePlayQuestArcRequest struct {
	Position  int                           `form:"position" json:"position"`
	Slug      string                        `form:"slug" json:"slug"`
	Title     string                        `form:"title" json:"title"`
	Summary   string                        `form:"summary" json:"summary"`
	Objective string                        `form:"objective" json:"objective"`
	Prompt    string                        `form:"prompt" json:"prompt"`
	Metadata  map[string]any                `form:"-" json:"metadata"`
	Chapters  []rolePlayQuestChapterRequest `form:"-" json:"chapters"`
}

type rolePlayQuestChapterRequest struct {
	Position      int            `form:"position" json:"position"`
	Slug          string         `form:"slug" json:"slug"`
	Title         string         `form:"title" json:"title"`
	Summary       string         `form:"summary" json:"summary"`
	Objective     string         `form:"objective" json:"objective"`
	IntroPrompt   string         `form:"introPrompt" json:"introPrompt"`
	SuccessPrompt string         `form:"successPrompt" json:"successPrompt"`
	FailurePrompt string         `form:"failurePrompt" json:"failurePrompt"`
	IsBoss        bool           `form:"isBoss" json:"isBoss"`
	Xp            int            `form:"xp" json:"xp"`
	Coin          int            `form:"coin" json:"coin"`
	Metadata      map[string]any `form:"-" json:"metadata"`
}

type liveSessionRequest struct {
	ChannelKey        string `form:"channelKey" json:"channelKey"`
	Mode              string `form:"mode" json:"mode"`
	Status            string `form:"status" json:"status"`
	BattleSaveID      *uint  `form:"battleSaveId" json:"battleSaveId"`
	RolePlaySessionID *uint  `form:"rolePlaySessionId" json:"rolePlaySessionId"`
	ArenaID           *uint  `form:"arenaId" json:"arenaId"`
	CoopPartyID       *uint  `form:"coopPartyId" json:"coopPartyId"`
	AllowReplay       bool   `form:"allowReplay" json:"allowReplay"`
}

type arenaRequest struct {
	Name            string `form:"name" json:"name"`
	BattleSaveID    uint   `form:"battleSaveId" json:"battleSaveId"`
	MaxPlayers      int    `form:"maxPlayers" json:"maxPlayers"`
	AllowSpectators bool   `form:"allowSpectators" json:"allowSpectators"`
	Spectator       bool   `form:"spectator" json:"spectator"`
}

type rolePlaySessionRequest struct {
	TemplateID     uint           `form:"templateId" json:"templateId"`
	Title          string         `form:"title" json:"title"`
	Mode           string         `form:"mode" json:"mode"`
	ScenarioPrompt string         `form:"scenarioPrompt" json:"scenarioPrompt"`
	Snapshot       map[string]any `form:"-" json:"snapshot"`
	ProviderName   string         `form:"providerName" json:"providerName"`
	Provider       string         `form:"provider" json:"provider"`
	ModelName      string         `form:"modelName" json:"modelName"`
	Model          string         `form:"model" json:"model"`
	APIKey         string         `form:"apiKey" json:"apiKey"`
	CharacterID    uint           `form:"characterId" json:"characterId"`
	BillingMode    string         `form:"billingMode" json:"billingMode"`
}

type rolePlayActionRequest struct {
	AuthorName string         `form:"authorName" json:"authorName"`
	Content    string         `form:"content" json:"content"`
	Payload    map[string]any `form:"-" json:"payload"`
}

type coopPartyRequest struct {
	Mode              string         `form:"mode" json:"mode"`
	BattleSaveID      *uint          `form:"battleSaveId" json:"battleSaveId"`
	RolePlaySessionID *uint          `form:"rolePlaySessionId" json:"rolePlaySessionId"`
	MaxMembers        int            `form:"maxMembers" json:"maxMembers"`
	SharedState       map[string]any `form:"-" json:"sharedState"`
	CharacterID       uint           `form:"characterId" json:"characterId"`
}

type rolePlayCharacterRequest struct {
	Name            string         `form:"name" json:"name"`
	Class           string         `form:"class" json:"class"`
	Archetype       string         `form:"archetype" json:"archetype"`
	Origin          string         `form:"origin" json:"origin"`
	Race            string         `form:"race" json:"race"`
	Species         string         `form:"species" json:"species"`
	Alignment       string         `form:"alignment" json:"alignment"`
	Personality     string         `form:"personality" json:"personality"`
	Background      string         `form:"background" json:"background"`
	PersonalGoal    string         `form:"personalGoal" json:"personalGoal"`
	PersonalGoalAlt string         `form:"personal_goal" json:"personal_goal"`
	Goal            string         `form:"goal" json:"goal"`
	Level           int            `form:"level" json:"level"`
	HeroImageID     *uint          `form:"heroImageId" json:"heroImageId"`
	ImageURL        string         `form:"imageUrl" json:"imageUrl"`
	Attributes      map[string]int `form:"-" json:"attributes"`
	Skills          map[string]int `form:"-" json:"skills"`
	Traits          []string       `form:"-" json:"traits"`
	Inventory       []string       `form:"-" json:"inventory"`
	Health          int            `form:"health" json:"health"`
	MaxHealth       int            `form:"maxHealth" json:"maxHealth"`
	MaxHealthAlt    int            `form:"max_health" json:"max_health"`
	Stress          int            `form:"stress" json:"stress"`
	Fatigue         int            `form:"fatigue" json:"fatigue"`
	Morale          int            `form:"morale" json:"morale"`
}

type rolePlayCharacterGenerateRequest struct {
	ProviderName string `form:"providerName" json:"providerName"`
	Provider     string `form:"provider" json:"provider"`
	ModelName    string `form:"modelName" json:"modelName"`
	Model        string `form:"model" json:"model"`
	APIKey       string `form:"apiKey" json:"apiKey"`
	Prompt       string `form:"prompt" json:"prompt"`
	QuestContext string `form:"questContext" json:"questContext"`
	Mode         string `form:"mode" json:"mode"`
	QuestID      uint   `form:"questId" json:"questId"`
	CoopPartyID  uint   `form:"coopPartyId" json:"coopPartyId"`
}

type updateUserRequest struct {
	Email           *string `form:"email" json:"email"`
	Pseudo          *string `form:"pseudo" json:"pseudo"`
	BirthdayDate    *string `form:"birthdayDate" json:"birthdayDate"`
	Avatar          *string `form:"avatar" json:"avatar"`
	CurrentPassword *string `form:"currentPassword" json:"currentPassword"`
	NewPassword     *string `form:"newPassword" json:"newPassword"`
}

type progressionRequest struct {
	XP        *int   `form:"xp" json:"xp"`
	Coin      *int   `form:"coin" json:"coin"`
	XPDelta   *int   `form:"xpDelta" json:"xpDelta"`
	CoinDelta *int   `form:"coinDelta" json:"coinDelta"`
	Reason    string `form:"reason" json:"reason"`
}

func listAIProviders() gin.HandlerFunc {
	return func(c *gin.Context) {
		providers := service.SupportedAIProviders()
		names := make([]string, 0, len(providers))
		for _, providerInfo := range providers {
			names = append(names, providerInfo.Name)
		}

		c.JSON(http.StatusOK, gin.H{
			"providers": providers,
			"names":     names,
		})
	}
}

func testAIProvider() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req aiProviderTestRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ai provider test payload"})
			return
		}

		providerName := defaultString(req.ProviderName, req.Provider)
		modelName := defaultString(req.ModelName, req.Model)
		apiKey := strings.TrimSpace(req.APIKey)

		if strings.TrimSpace(providerName) == "" || strings.TrimSpace(modelName) == "" || apiKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "providerName, modelName and apiKey are required"})
			return
		}

		url, err := providerURL(providerName)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "providerName invalide"})
			return
		}

		prompt := strings.TrimSpace(req.Prompt)
		if prompt == "" {
			prompt = "Reponds uniquement par OK si tu recois ce message."
		}

		startedAt := time.Now()
		ctx, cancel := context.WithTimeout(c.Request.Context(), aiProviderTestTimeout())
		defer cancel()

		client := provider.NewsProvider(apiKey, url, modelName)
		response, err := client.Chat(ctx, []provider.ProviderMessage{
			{Role: "system", Content: "Tu es un endpoint de verification technique. Reponds tres court."},
			{Role: "user", Content: prompt},
		})
		latency := time.Since(startedAt)

		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"ok":           false,
				"providerName": providerName,
				"modelName":    modelName,
				"latencyMs":    latency.Milliseconds(),
				"error":        providerTestErrorMessage(providerName, err),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":           true,
			"providerName": providerName,
			"modelName":    modelName,
			"latencyMs":    latency.Milliseconds(),
			"response":     truncateString(strings.TrimSpace(response), 500),
		})
	}
}

func generateAIProviderText() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req aiProviderGenerateRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ai provider generate payload"})
			return
		}

		providerName := defaultString(req.ProviderName, req.Provider)
		modelName := defaultString(req.ModelName, req.Model)
		apiKey := strings.TrimSpace(req.APIKey)
		prompt := strings.TrimSpace(req.Prompt)

		if strings.TrimSpace(providerName) == "" || strings.TrimSpace(modelName) == "" || apiKey == "" || prompt == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "providerName, modelName, apiKey and prompt are required"})
			return
		}

		url, err := providerURL(providerName)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "providerName invalide"})
			return
		}

		messages := make([]provider.ProviderMessage, 0, 2)
		if systemPrompt := strings.TrimSpace(req.SystemPrompt); systemPrompt != "" {
			messages = append(messages, provider.ProviderMessage{
				Role:    "system",
				Content: systemPrompt,
			})
		}
		messages = append(messages, provider.ProviderMessage{
			Role:    "user",
			Content: prompt,
		})

		startedAt := time.Now()
		generationTimeout := aiProviderGenerationTimeout()
		ctx, cancel := context.WithTimeout(c.Request.Context(), generationTimeout)
		defer cancel()

		client := provider.NewsProvider(apiKey, url, modelName)
		response, err := client.Chat(ctx, messages)
		latency := time.Since(startedAt)

		if err != nil {
			statusCode := http.StatusBadGateway
			retryable := false
			errorMessage := err.Error()
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
				statusCode = http.StatusGatewayTimeout
				retryable = true
				errorMessage = fmt.Sprintf("AI provider generation timed out after %s", generationTimeout)
			}
			log.Printf(
				"[aiProviderGenerate] provider call failed provider=%s model=%s timeout_ms=%d latency_ms=%d status=%d retryable=%t err=%v",
				providerName,
				modelName,
				generationTimeout.Milliseconds(),
				latency.Milliseconds(),
				statusCode,
				retryable,
				err,
			)
			c.JSON(statusCode, gin.H{
				"ok":           false,
				"providerName": providerName,
				"modelName":    modelName,
				"latencyMs":    latency.Milliseconds(),
				"timeoutMs":    generationTimeout.Milliseconds(),
				"retryable":    retryable,
				"error":        errorMessage,
			})
			return
		}

		generated := strings.TrimSpace(response)
		if req.MaxChars > 0 {
			generated = truncateString(generated, req.MaxChars)
		}
		log.Printf(
			"[aiProviderGenerate] provider call succeeded provider=%s model=%s timeout_ms=%d latency_ms=%d response_chars=%d",
			providerName,
			modelName,
			generationTimeout.Milliseconds(),
			latency.Milliseconds(),
			len(generated),
		)

		c.JSON(http.StatusOK, gin.H{
			"ok":           true,
			"providerName": providerName,
			"modelName":    modelName,
			"latencyMs":    latency.Milliseconds(),
			"response":     generated,
		})
	}
}

func startBattle(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req battleRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid battle payload"})
			return
		}

		battleService := newBattleService(database)
		run, err := battleService.Create(c.Request.Context(), currentUserID(c), toServiceBattleRequest(req))
		if err != nil {
			log.Printf("[startBattle] Create error: %v", err)
			writeBillingError(c, err)
			return
		}

		c.Writer.Header().Set("Content-Type", "application/x-ndjson")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming non supporte"})
			return
		}

		writeNDJSON(c, flusher, gin.H{
			"type":      "battle_created",
			"battle_id": run.Battle.Id,
			"done":      true,
		})

		defer func() {
			if r := recover(); r != nil {
				log.Printf("[startBattle] PANIC recovered: %v", r)
				writeNDJSON(c, flusher, scenarios.BattleStreamEvent{
					Type:  "error",
					Error: fmt.Sprintf("internal server error: %v", r),
					Done:  true,
				})
			}
		}()

		if err := battleService.RunNextRound(c.Request.Context(), run, nil, func(event scenarios.BattleStreamEvent) {
			writeNDJSON(c, flusher, event)
		}); err != nil {
			log.Printf("[startBattle] RunNextRound error: %v", err)
		}
	}
}

func nextBattleRound(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req battleRequest
		if c.Request.ContentLength != 0 {
			if err := bindPayload(c, &req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid battle round payload"})
				return
			}
		}

		battleID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid battle id"})
			return
		}

		battleService := newBattleService(database)
		run, turns, err := battleService.Resume(c.Request.Context(), currentUserID(c), battleID, toServiceBattleRequest(req))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.Writer.Header().Set("Content-Type", "application/x-ndjson")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming non supporte"})
			return
		}

		writeNDJSON(c, flusher, gin.H{
			"type":           "round_started",
			"battle_id":      run.Battle.Id,
			"existing_turns": len(turns),
			"done":           true,
		})

		if err := battleService.RunNextRound(c.Request.Context(), run, turns, func(event scenarios.BattleStreamEvent) {
			writeNDJSON(c, flusher, event)
		}); err != nil {
			log.Printf("[nextBattleRound] RunNextRound error: %v", err)
		}
	}
}

func judgeBattle(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req battleRequest
		if c.Request.ContentLength != 0 {
			if err := bindPayload(c, &req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid judge payload"})
				return
			}
		}

		battleID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid battle id"})
			return
		}

		battleService := newBattleService(database)

		c.Writer.Header().Set("Content-Type", "application/x-ndjson")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming non supporte"})
			return
		}

		writeNDJSON(c, flusher, gin.H{
			"type":      "judge_started",
			"battle_id": battleID,
			"done":      true,
		})

		if err := battleService.Judge(
			c.Request.Context(),
			currentUserID(c),
			battleID,
			toServiceBattleRequest(req),
			func(event scenarios.BattleStreamEvent) {
				writeNDJSON(c, flusher, event)
			},
		); err != nil {
			writeNDJSON(c, flusher, scenarios.BattleStreamEvent{
				Type:  "error",
				Error: err.Error(),
				Done:  true,
			})
		}
	}
}

func resumeBattle(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req battleRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid battle resume payload"})
			return
		}

		battleID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid battle id"})
			return
		}

		battleService := newBattleService(database)
		run, turns, err := battleService.Resume(c.Request.Context(), currentUserID(c), battleID, toServiceBattleRequest(req))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.Writer.Header().Set("Content-Type", "application/x-ndjson")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming non supporte"})
			return
		}

		writeNDJSON(c, flusher, gin.H{
			"type":           "battle_resumed",
			"battle_id":      run.Battle.Id,
			"existing_turns": len(turns),
			"done":           true,
		})

		_ = battleService.Run(c.Request.Context(), run, turns, func(event scenarios.BattleStreamEvent) {
			writeNDJSON(c, flusher, event)
		})
	}
}

func listBattles(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var battles []models.BattleSave
		err := database.WithContext(c.Request.Context()).
			Where("owner_id = ?", currentUserID(c)).
			Order("updated_at DESC").
			Limit(limitFromQuery(c)).
			Find(&battles).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list battles"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"battles": battles})
	}
}

func getBattle(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		battle, ok := findOwnedBattle(c, database)
		if !ok {
			return
		}

		c.JSON(http.StatusOK, gin.H{"battle": battle})
	}
}

func getBattleTurns(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		battle, ok := findOwnedBattle(c, database)
		if !ok {
			return
		}

		var turns []models.BattleSaveTurn
		err := database.WithContext(c.Request.Context()).
			Where("battle_save_id = ?", battle.Id).
			Order("sequence ASC").
			Find(&turns).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list battle turns"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"turns": turns})
	}
}

func getBattleUsage(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		battle, ok := findOwnedBattle(c, database)
		if !ok {
			return
		}

		var records []models.AIUsageRecord
		if err := database.WithContext(c.Request.Context()).
			Where("battle_save_id = ? AND owner_id = ?", battle.Id, currentUserID(c)).
			Order("created_at ASC").
			Find(&records).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list battle usage"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"battleId": battle.Id,
			"summary": gin.H{
				"promptTokens":        battle.PromptTokens,
				"completionTokens":    battle.CompletionTokens,
				"totalTokens":         battle.TotalTokens,
				"estimatedCostMicros": battle.EstimatedCostMicros,
				"currency":            "USD",
			},
			"records": records,
		})
	}
}

func listPublicBattles(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var battles []models.BattleSave
		err := publicBattleScope(database.WithContext(c.Request.Context()).Model(&models.BattleSave{})).
			Order("updated_at DESC").
			Limit(limitFromQuery(c)).
			Find(&battles).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list public battles"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"battles": battles})
	}
}

func getPublicBattle(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		battle, ok := findPublicBattle(c, database)
		if !ok {
			return
		}

		c.JSON(http.StatusOK, gin.H{"battle": battle})
	}
}

func getPublicBattleTurns(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		battle, ok := findPublicBattle(c, database)
		if !ok {
			return
		}

		var turns []models.BattleSaveTurn
		err := database.WithContext(c.Request.Context()).
			Where("battle_save_id = ?", battle.Id).
			Order("sequence ASC").
			Find(&turns).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list public battle turns"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"turns": turns})
	}
}

func cancelBattle(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		battle, ok := findOwnedBattle(c, database)
		if !ok {
			return
		}

		now := time.Now()
		err := database.WithContext(c.Request.Context()).
			Model(&battle).
			Updates(map[string]any{
				"status":           constants.BattleStatusAbandoned,
				"finished_at":      &now,
				"last_activity_at": &now,
			}).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot cancel battle"})
			return
		}
		_ = newLiveService(database).EndSessionsByBattle(c.Request.Context(), battle.Id)

		c.JSON(http.StatusOK, gin.H{"battle": battle})
	}
}

func listBattleQuests(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := limitFromQuery(c)
		offset := offsetFromQuery(c)
		quests, total, err := newQuestService(database).ListBattlePage(c.Request.Context(), c.Query("status"), c.Query("theme"), c.Query("level"), limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list battle quests"})
			return
		}

		c.JSON(http.StatusOK, paginatedQuestResponse("quests", quests, total, limit, offset))
	}
}

func getBattleQuest(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, parseErr := parseUintParam(c, "id")
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest id"})
			return
		}
		quest, err := newQuestService(database).GetBattle(c.Request.Context(), id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "battle quest not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get battle quest"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"quest": quest})
	}
}

func createBattleQuest(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req battleQuestRequest
		if err := bindPayload(c, &req); err != nil || req.Title == "" || req.Content == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title and content are required"})
			return
		}

		quest, err := newQuestService(database).CreateBattle(c.Request.Context(), toBattleQuestInput(req))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create battle quest"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"quest": quest})
	}
}

func updateBattleQuest(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest id"})
			return
		}
		var req battleQuestRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid battle quest payload"})
			return
		}
		if err := newQuestService(database).UpdateBattle(c.Request.Context(), id, toBattleQuestInput(req)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update battle quest"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"updated": true})
	}
}

func deleteBattleQuest(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest id"})
			return
		}
		if err := newQuestService(database).DeleteBattle(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot delete battle quest"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

func publishBattleQuest(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest id"})
			return
		}
		if err := newQuestService(database).PublishBattle(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot publish battle quest"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": constants.QuestStatusPublished})
	}
}

func archiveBattleQuest(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest id"})
			return
		}
		if err := newQuestService(database).ArchiveBattle(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot archive battle quest"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": constants.QuestStatusArchived})
	}
}

func randomBattleQuest(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		quest, err := newQuestService(database).RandomBattle(c.Request.Context(), c.Query("theme"), c.Query("level"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "battle quest not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"quest": quest})
	}
}

func listIAProfiles(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var profiles []models.IAProfile
		err := database.WithContext(c.Request.Context()).
			Where("owner_id = ?", currentUserID(c)).
			Order("updated_at DESC").
			Limit(limitFromQuery(c)).
			Find(&profiles).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list ia profiles"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"profiles": profiles})
	}
}

func createIAProfile(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req iaProfileRequest
		if err := bindPayload(c, &req); err != nil || req.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}

		if _, err := providerURL(req.ProviderName); req.ProviderName != "" && err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "providerName invalide"})
			return
		}

		profile := models.IAProfile{
			OwnerID:      currentUserID(c),
			Name:         req.Name,
			ProviderName: req.ProviderName,
			ModelName:    req.ModelName,
			Personality:  req.Personality,
			Mindset:      req.Mindset,
			Style:        req.Style,
			Goal:         req.Goal,
			Weakness:     req.Weakness,
		}
		if err := database.WithContext(c.Request.Context()).Create(&profile).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create ia profile"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"profile": profile})
	}
}

func getIAProfile(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		profile, ok := findOwnedIAProfile(c, database)
		if !ok {
			return
		}

		c.JSON(http.StatusOK, gin.H{"profile": profile})
	}
}

func updateIAProfile(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		profile, ok := findOwnedIAProfile(c, database)
		if !ok {
			return
		}

		var req iaProfileRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ia profile payload"})
			return
		}
		if req.ProviderName != "" {
			if _, err := providerURL(req.ProviderName); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "providerName invalide"})
				return
			}
		}

		updates := map[string]any{
			"name":          defaultString(req.Name, profile.Name),
			"provider_name": defaultString(req.ProviderName, profile.ProviderName),
			"model_name":    defaultString(req.ModelName, profile.ModelName),
			"personality":   defaultString(req.Personality, profile.Personality),
			"mindset":       defaultString(req.Mindset, profile.Mindset),
			"style":         defaultString(req.Style, profile.Style),
			"goal":          defaultString(req.Goal, profile.Goal),
			"weakness":      defaultString(req.Weakness, profile.Weakness),
		}
		if err := database.WithContext(c.Request.Context()).Model(&profile).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update ia profile"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"profile": profile})
	}
}

func deleteIAProfile(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		profile, ok := findOwnedIAProfile(c, database)
		if !ok {
			return
		}

		if err := database.WithContext(c.Request.Context()).Delete(&profile).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot delete ia profile"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

func listRolePlayCharacters(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		characters, err := newRolePlayCharacterService(database).List(c.Request.Context(), currentUserID(c), limitFromQuery(c))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list roleplay characters"})
			return
		}
		response := make([]models.RolePlayCharacter, 0, len(characters))
		for i := range characters {
			response = append(response, rolePlayCharacterWithServedImage(database, c.Request.Context(), &characters[i]))
		}
		c.JSON(http.StatusOK, gin.H{"characters": response})
	}
}

func createRolePlayCharacter(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req rolePlayCharacterRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roleplay character payload"})
			return
		}
		character, err := newRolePlayCharacterService(database).Create(c.Request.Context(), currentUserID(c), toRolePlayCharacterInput(req))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		response := rolePlayCharacterWithServedImage(database, c.Request.Context(), character)
		c.JSON(http.StatusCreated, gin.H{"character": response})
	}
}

func getRolePlayCharacter(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil || id == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roleplay character id"})
			return
		}
		character, err := newRolePlayCharacterService(database).Get(c.Request.Context(), currentUserID(c), uint(id))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "roleplay character not found"})
			return
		}
		response := rolePlayCharacterWithServedImage(database, c.Request.Context(), character)
		c.JSON(http.StatusOK, gin.H{"character": response})
	}
}

func updateRolePlayCharacter(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil || id == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roleplay character id"})
			return
		}
		var req rolePlayCharacterRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roleplay character payload"})
			return
		}
		character, err := newRolePlayCharacterService(database).Update(c.Request.Context(), currentUserID(c), uint(id), toRolePlayCharacterInput(req))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		response := rolePlayCharacterWithServedImage(database, c.Request.Context(), character)
		c.JSON(http.StatusOK, gin.H{"character": response})
	}
}

func rolePlayCharacterWithServedImage(database *gorm.DB, ctx context.Context, character *models.RolePlayCharacter) models.RolePlayCharacter {
	if character == nil {
		return models.RolePlayCharacter{}
	}
	response := *character
	if response.HeroImageID == nil || *response.HeroImageID == 0 {
		return response
	}
	var image models.RolePlayHeroImage
	if err := database.WithContext(ctx).
		Where("id = ? AND is_active = ?", *response.HeroImageID, true).
		First(&image).Error; err == nil {
		response.ImageURL = rolePlayHeroImagePublicURL(image)
	}
	return response
}

func deleteRolePlayCharacter(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil || id == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roleplay character id"})
			return
		}
		if err := newRolePlayCharacterService(database).Delete(c.Request.Context(), currentUserID(c), uint(id)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot delete roleplay character"})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func listRolePlayHeroImages(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var images []models.RolePlayHeroImage
		query := database.WithContext(c.Request.Context()).Where("is_active = ?", true)
		sex := strings.ToLower(strings.TrimSpace(c.Query("sex")))
		if sex == "h" || sex == "f" {
			query = query.Where("sex = ?", sex)
		}
		if err := query.Order("sex ASC, name ASC, id DESC").Find(&images).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list roleplay hero images"})
			return
		}
		response := make([]gin.H, 0, len(images))
		for _, image := range images {
			if len(image.ImageData) == 0 {
				if path, err := rolePlayHeroImageDiskPath(image); err == nil {
					backfillRolePlayHeroImageData(database, c.Request.Context(), &image, path)
				}
			}
			response = append(response, rolePlayHeroImagePayload(image))
		}
		c.JSON(http.StatusOK, gin.H{"images": response})
	}
}

func getPublicRolePlayHeroImage(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil || id == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid hero image id"})
			return
		}

		var image models.RolePlayHeroImage
		if err := database.WithContext(c.Request.Context()).
			Where("id = ? AND is_active = ?", uint(id), true).
			First(&image).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "hero image not found"})
			return
		}

		path, err := rolePlayHeroImageDiskPath(image)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "hero image file not found"})
			return
		}
		backfillRolePlayHeroImageData(database, c.Request.Context(), &image, path)

		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.File(path)
	}
}

func backfillRolePlayHeroImageData(database *gorm.DB, ctx context.Context, image *models.RolePlayHeroImage, path string) {
	if image == nil || image.Id == 0 || len(image.ImageData) > 0 {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 || len(data) > 10*1024*1024 {
		return
	}
	updates := map[string]any{
		"image_data": data,
	}
	if image.ImageSize <= 0 {
		updates["image_size"] = int64(len(data))
	}
	if err := database.WithContext(ctx).
		Model(&models.RolePlayHeroImage{}).
		Where("id = ? AND image_data IS NULL", image.Id).
		Updates(updates).Error; err == nil {
		image.ImageData = data
		if image.ImageSize <= 0 {
			image.ImageSize = int64(len(data))
		}
	}
}

func rolePlayHeroImagePayload(image models.RolePlayHeroImage) gin.H {
	return gin.H{
		"id":        image.Id,
		"createdAt": image.CreatedAt,
		"updatedAt": image.UpdatedAt,
		"name":      image.Name,
		"sex":       image.Sex,
		"imageUrl":  rolePlayHeroImagePublicURL(image),
		"imageHash": image.ImageHash,
		"imageSize": image.ImageSize,
		"version":   image.Version,
		"isActive":  image.IsActive,
	}
}

func rolePlayHeroImagePublicURL(image models.RolePlayHeroImage) string {
	return fmt.Sprintf("/api/v1/public/roleplay/hero-images/%d/image?v=%d", image.Id, image.Version)
}

func rolePlayHeroImageDiskPath(image models.RolePlayHeroImage) (string, error) {
	rel, err := rolePlayHeroImageRelativePath(image.ImageURL)
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(service.HeroImagePublicDir())
	if err != nil {
		return "", err
	}
	candidate, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return "", err
	}
	if candidate != root && !strings.HasPrefix(candidate, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid hero image path")
	}
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		if len(image.ImageData) == 0 {
			return "", fmt.Errorf("hero image file not found")
		}
		if err := os.MkdirAll(filepath.Dir(candidate), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(candidate, image.ImageData, 0o644); err != nil {
			return "", err
		}
	}
	return candidate, nil
}

func rolePlayHeroImageRelativePath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("empty hero image URL")
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Path != "" {
		trimmed = parsed.Path
	}

	publicBase := service.HeroImagePublicBaseURL()
	if parsed, err := url.Parse(publicBase); err == nil && parsed.Path != "" {
		publicBase = parsed.Path
	}
	publicBase = "/" + strings.Trim(strings.TrimSpace(publicBase), "/")
	prefixes := []string{
		publicBase + "/",
		"/assets/heroes/",
		"/nexus_game/assets/heroes/",
		"assets/heroes/",
		"nexus_game/assets/heroes/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(trimmed, prefix) {
			rel := strings.TrimPrefix(trimmed, prefix)
			rel = strings.TrimPrefix(rel, "/")
			if rel != "" {
				return filepath.ToSlash(rel), nil
			}
		}
	}
	return "", fmt.Errorf("unsupported hero image URL")
}

func validateRolePlayCharacter(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req rolePlayCharacterRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roleplay character payload"})
			return
		}
		character, err := newRolePlayCharacterService(database).ValidateDraft(currentUserID(c), toRolePlayCharacterInput(req))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"character": character, "valid": true})
	}
}

func rolePlayCharacterLocalPrompt(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req rolePlayCharacterGenerateRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roleplay character generation payload"})
			return
		}
		prompt, schema := newRolePlayCharacterService(database).PrepareGenerationPrompt(c.Request.Context(), service.CharacterGenerationInput{
			PlayerPrompt: req.Prompt,
			QuestContext: req.QuestContext,
			Mode:         req.Mode,
			QuestID:      req.QuestID,
			CoopPartyID:  req.CoopPartyID,
			LocalLLM:     true,
		})
		c.JSON(http.StatusOK, gin.H{"prompt": prompt, "schema": schema})
	}
}

func generateRolePlayCharacter(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req rolePlayCharacterGenerateRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roleplay character generation payload"})
			return
		}
		character, raw, err := newRolePlayCharacterService(database).Generate(c.Request.Context(), currentUserID(c), service.CharacterGenerationInput{
			ProviderName: defaultString(req.ProviderName, req.Provider),
			ModelName:    defaultString(req.ModelName, req.Model),
			APIKey:       req.APIKey,
			PlayerPrompt: req.Prompt,
			QuestContext: req.QuestContext,
			Mode:         req.Mode,
			QuestID:      req.QuestID,
			CoopPartyID:  req.CoopPartyID,
		})
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "raw": truncateString(raw, 1000)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"character": character, "saved": false})
	}
}

func listRolePlayQuests(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := limitFromQuery(c)
		offset := offsetFromQuery(c)
		quests, total, err := newQuestService(database).ListRolePlayPage(c.Request.Context(), c.Query("status"), c.Query("theme"), c.Query("level"), limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list roleplay quests"})
			return
		}

		c.JSON(http.StatusOK, paginatedQuestResponse("quests", quests, total, limit, offset))
	}
}

func getRolePlayQuest(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, parseErr := parseUintParam(c, "id")
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest id"})
			return
		}
		quest, err := newQuestService(database).GetRolePlay(c.Request.Context(), id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "roleplay quest not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get roleplay quest"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"quest": quest})
	}
}

func createRolePlayQuest(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req rolePlayQuestRequest
		if err := bindPayload(c, &req); err != nil || req.Title == "" || req.Prompt == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title and prompt are required"})
			return
		}

		quest, err := newQuestService(database).CreateRolePlay(c.Request.Context(), toRolePlayQuestInput(req))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create roleplay quest"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"quest": quest})
	}
}

func updateRolePlayQuest(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest id"})
			return
		}
		var req rolePlayQuestRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roleplay quest payload"})
			return
		}
		if err := newQuestService(database).UpdateRolePlay(c.Request.Context(), id, toRolePlayQuestInput(req)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update roleplay quest"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"updated": true})
	}
}

func deleteRolePlayQuest(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest id"})
			return
		}
		if err := newQuestService(database).DeleteRolePlay(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot delete roleplay quest"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"deleted": true})
	}
}

func startRolePlayQuest(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req rolePlaySessionRequest
		if c.Request.ContentLength != 0 {
			if err := bindPayload(c, &req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roleplay start payload"})
				return
			}
		}

		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quest id"})
			return
		}
		req.Snapshot = prepareInitialRolePlaySnapshot(req.Snapshot, req.Title, req.ScenarioPrompt)
		snapshotQuestID := service.UintFromSnapshot(req.Snapshot, "questId")
		if snapshotQuestID != 0 && snapshotQuestID != id {
			log.Printf(
				"[ROLEPLAY_START][REJECT] pathQuest=%d snapshotQuest=%d",
				id,
				snapshotQuestID,
			)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":           "snapshot questId mismatch",
				"expectedQuestId": id,
				"snapshotQuestId": snapshotQuestID,
			})
			return
		}
		if snapshotQuestID == 0 {
			if req.Snapshot == nil {
				req.Snapshot = map[string]any{}
			}
			req.Snapshot["questId"] = id
			snapshotQuestID = id
		}
		resolvedProvider := defaultString(req.ProviderName, req.Provider)
		resolvedModel := defaultString(req.ModelName, req.Model)
		if !strings.EqualFold(strings.TrimSpace(req.Mode), "localDevice") {
			resolvedProvider, resolvedModel = service.ResolveRolePlayProviderDefaults(
				req.BillingMode,
				resolvedProvider,
				resolvedModel,
			)
		} else {
			resolvedProvider = ""
			resolvedModel = ""
		}
		log.Printf(
			"[ROLEPLAY_START] quest=%d snapshotQuestId=%d user=%d mode=%s provider=%s model=%s billing=%s character=%d snapshotKeys=%v",
			id,
			snapshotQuestID,
			currentUserID(c),
			req.Mode,
			resolvedProvider,
			resolvedModel,
			req.BillingMode,
			req.CharacterID,
			mapKeys(req.Snapshot),
		)
		session, err := newRolePlayService(database).CreateSession(c.Request.Context(), currentUserID(c), service.RolePlaySessionInput{
			TemplateID:     id,
			Title:          req.Title,
			Mode:           req.Mode,
			ScenarioPrompt: req.ScenarioPrompt,
			Snapshot:       req.Snapshot,
			ProviderName:   resolvedProvider,
			ModelName:      resolvedModel,
			APIKey:         req.APIKey,
			CharacterID:    req.CharacterID,
			BillingMode:    req.BillingMode,
		})
		if err != nil {
			writeBillingError(c, err)
			return
		}
		turns, _ := newRolePlayService(database).Turns(c.Request.Context(), session.Id, currentUserID(c))
		c.JSON(http.StatusCreated, gin.H{"session": session, "turns": turns})
	}
}

func createRolePlaySession(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req rolePlaySessionRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roleplay session payload"})
			return
		}
		req.Snapshot = prepareInitialRolePlaySnapshot(req.Snapshot, req.Title, req.ScenarioPrompt)
		session, err := newRolePlayService(database).CreateSession(c.Request.Context(), currentUserID(c), service.RolePlaySessionInput{
			TemplateID:     req.TemplateID,
			Title:          req.Title,
			Mode:           req.Mode,
			ScenarioPrompt: req.ScenarioPrompt,
			Snapshot:       req.Snapshot,
			ProviderName:   defaultString(req.ProviderName, req.Provider),
			ModelName:      defaultString(req.ModelName, req.Model),
			APIKey:         req.APIKey,
			CharacterID:    req.CharacterID,
			BillingMode:    req.BillingMode,
		})
		if err != nil {
			writeBillingError(c, err)
			return
		}
		turns, _ := newRolePlayService(database).Turns(c.Request.Context(), session.Id, currentUserID(c))
		c.JSON(http.StatusCreated, gin.H{"session": session, "turns": turns})
	}
}

func listRolePlaySessions(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessions, err := newRolePlayService(database).ListSessions(c.Request.Context(), currentUserID(c), limitFromQuery(c))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list roleplay sessions"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"sessions": sessions})
	}
}

func getRolePlaySession(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, parseErr := parseUintParam(c, "id")
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		session, err := newRolePlayService(database).GetSession(c.Request.Context(), id, currentUserID(c))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "roleplay session not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get roleplay session"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"session": session})
	}
}

func getRolePlaySessionTurns(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		turns, err := newRolePlayService(database).Turns(c.Request.Context(), id, currentUserID(c))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "roleplay session not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"turns": turns})
	}
}

func getRolePlayUsage(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		session, err := newRolePlayService(database).GetSession(c.Request.Context(), id, currentUserID(c))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "roleplay session not found"})
			return
		}

		var records []models.AIUsageRecord
		if err := database.WithContext(c.Request.Context()).
			Where("role_play_session_id = ? AND owner_id = ?", session.Id, currentUserID(c)).
			Order("created_at ASC").
			Find(&records).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list roleplay usage"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"rolePlaySessionId": session.Id,
			"summary": gin.H{
				"promptTokens":        session.PromptTokens,
				"completionTokens":    session.CompletionTokens,
				"totalTokens":         session.TotalTokens,
				"estimatedCostMicros": session.EstimatedCostMicros,
				"currency":            "USD",
			},
			"records": records,
		})
	}
}

func resumeRolePlaySession(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		session, turns, err := newRolePlayService(database).Resume(c.Request.Context(), id, currentUserID(c))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "roleplay session not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"session": session, "turns": turns})
	}
}

func appendRolePlayAction(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		var req rolePlayActionRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action payload"})
			return
		}
		turn, err := newRolePlayService(database).AppendAction(c.Request.Context(), id, currentUserID(c), service.RolePlayActionInput{
			AuthorName: req.AuthorName,
			Content:    req.Content,
			Payload:    req.Payload,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		session, _ := newRolePlayService(database).GetSession(c.Request.Context(), id, currentUserID(c))
		questCompleted := session != nil && session.Status == constants.RolePlayStatusFinished
		nextOptions := req.Payload["nextOptions"]
		if nextOptions == nil {
			nextOptions = req.Payload["options"]
		}
		c.JSON(http.StatusCreated, gin.H{
			"turn":           turn,
			"session":        session,
			"questCompleted": questCompleted,
			"options":        nextOptions,
			"choices":        nextOptions,
			"currentScene":   req.Payload["scene"],
		})
	}
}

func endRolePlaySession(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		if err := newRolePlayService(database).End(c.Request.Context(), id, currentUserID(c)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot end roleplay session"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ended": true})
	}
}

func listArenas(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		arenas, err := newArenaService(database).List(c.Request.Context(), limitFromQuery(c))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list arenas"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"arenas": arenas})
	}
}

func listPublicArenas(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var arenas []models.BattleArena
		err := publicArenaScope(database.WithContext(c.Request.Context()).Model(&models.BattleArena{})).
			Order("updated_at DESC").
			Limit(limitFromQuery(c)).
			Find(&arenas).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list public arenas"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"arenas": arenas})
	}
}

func createArena(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req arenaRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid arena payload"})
			return
		}
		arena, err := newArenaService(database).Create(c.Request.Context(), currentUserID(c), service.ArenaInput{
			Name:            req.Name,
			BattleSaveID:    req.BattleSaveID,
			MaxPlayers:      req.MaxPlayers,
			AllowSpectators: req.AllowSpectators,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"arena": arena})
	}
}

func getArena(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		arena, err := newArenaService(database).Get(c.Request.Context(), c.Param("code"))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "arena not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get arena"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"arena": arena})
	}
}

func getPublicArena(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		arena, ok := findPublicArena(c, database)
		if !ok {
			return
		}

		c.JSON(http.StatusOK, gin.H{"arena": arena})
	}
}

func joinArena(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req arenaRequest
		_ = bindPayload(c, &req)
		arena, err := newArenaService(database).Join(c.Request.Context(), c.Param("code"), currentUserID(c), req.Spectator)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"arena": arena, "joined": true})
	}
}

func leaveArena(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := newArenaService(database).Leave(c.Request.Context(), c.Param("code"), currentUserID(c)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"left": true})
	}
}

func listArenaMembers(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		members, err := newArenaService(database).Members(c.Request.Context(), c.Param("code"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "arena not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"members": members})
	}
}

func listCoopParties(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		parties, err := newCoopService(database).ListByHost(c.Request.Context(), currentUserID(c), limitFromQuery(c))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list coop parties"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"parties": parties})
	}
}

func createCoopParty(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req coopPartyRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coop payload"})
			return
		}
		party, err := newCoopService(database).Create(c.Request.Context(), currentUserID(c), service.CoopPartyInput{
			Mode:              req.Mode,
			BattleSaveID:      req.BattleSaveID,
			RolePlaySessionID: req.RolePlaySessionID,
			MaxMembers:        req.MaxMembers,
			SharedState:       req.SharedState,
			CharacterID:       req.CharacterID,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		_ = createCoopLiveSession(c.Request.Context(), database, currentUserID(c), party)
		c.JSON(http.StatusCreated, gin.H{"party": party})
	}
}

func getCoopParty(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		coop := newCoopService(database)
		var party *models.CoopParty
		var err error
		if c.Query("compact") == "1" {
			party, err = coop.GetCompact(c.Request.Context(), c.Param("code"))
		} else {
			party, err = coop.Get(c.Request.Context(), c.Param("code"))
		}
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "coop party not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"party": party})
	}
}

func joinCoopParty(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req coopPartyRequest
		_ = bindPayload(c, &req)
		party, err := newCoopService(database).Join(c.Request.Context(), c.Param("code"), currentUserID(c), req.CharacterID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"party": party, "joined": true})
	}
}

func leaveCoopParty(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := currentUserID(c)
		party, err := newCoopService(database).Get(c.Request.Context(), c.Param("code"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "coop party not found"})
			return
		}
		shouldEndLive := party.HostUserID == userID && coopPartyCarriesRolePlay(party)
		if err := newCoopService(database).Leave(c.Request.Context(), c.Param("code"), userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		endedLives := 0
		if shouldEndLive {
			var err error
			endedLives, err = endLiveSessionsByCoopParty(c.Request.Context(), database, party.Id, "host left roleplay coop party")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot end coop live sessions"})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"left": true, "liveEnded": endedLives > 0, "endedLiveSessions": endedLives})
	}
}

func readyCoopParty(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := newCoopService(database).Ready(c.Request.Context(), c.Param("code"), currentUserID(c)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ready": true})
	}
}

func listCoopMembers(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		members, err := newCoopService(database).Members(c.Request.Context(), c.Param("code"), currentUserID(c))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "coop party not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"members": members})
	}
}

func updateCoopState(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req coopPartyRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid coop state payload"})
			return
		}
		party, err := newCoopService(database).UpdateState(c.Request.Context(), c.Param("code"), currentUserID(c), req.SharedState)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"updated": true, "party": party})
	}
}

func listLiveSessions(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := limitFromQuery(c)
		cacheKey := fmt.Sprintf("sessions:owner:%d:limit:%d", currentUserID(c), limit)
		var sessions []models.LiveSession
		if appResponseCache().GetJSON(c.Request.Context(), "live", cacheKey, &sessions) {
			c.JSON(http.StatusOK, gin.H{"sessions": sessions})
			return
		}
		err := repository.PreloadLiveRolePlayQuestVisuals(database.WithContext(c.Request.Context())).
			Where("owner_id = ?", currentUserID(c)).
			Order("updated_at DESC").
			Limit(limit).
			Find(&sessions).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list live sessions"})
			return
		}
		for index := range sessions {
			repository.ApplyLiveSessionImageCompatibility(&sessions[index])
		}

		appResponseCache().SetJSON(c.Request.Context(), "live", cacheKey, sessions, 2*time.Second)
		c.JSON(http.StatusOK, gin.H{"sessions": sessions})
	}
}

func listPublicLiveSessions(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := limitFromQuery(c)
		cacheKey := fmt.Sprintf("sessions:public:limit:%d", limit)
		var sessions []models.LiveSession
		if appResponseCache().GetJSON(c.Request.Context(), "live", cacheKey, &sessions) {
			c.JSON(http.StatusOK, gin.H{"sessions": sessions})
			return
		}
		err := repository.PreloadLiveRolePlayQuestVisuals(publicLiveSessionScope(database.WithContext(c.Request.Context()).Model(&models.LiveSession{}))).
			Order("updated_at DESC").
			Limit(limit).
			Find(&sessions).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list public live sessions"})
			return
		}
		for index := range sessions {
			repository.ApplyLiveSessionImageCompatibility(&sessions[index])
		}

		appResponseCache().SetJSON(c.Request.Context(), "live", cacheKey, sessions, 2*time.Second)
		c.JSON(http.StatusOK, gin.H{"sessions": sessions})
	}
}

func createLiveSession(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req liveSessionRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid live session payload"})
			return
		}

		userID := currentUserID(c)
		if err := validateLiveAttachments(c, database, userID, req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		channelKey := req.ChannelKey
		if channelKey == "" {
			channelKey = "live-" + randomHex(12)
		}

		now := time.Now()
		live := models.LiveSession{
			OwnerID:           userID,
			ChannelKey:        channelKey,
			Mode:              defaultString(req.Mode, "battle_ia"),
			Status:            defaultString(req.Status, "streaming"),
			BattleSaveID:      req.BattleSaveID,
			RolePlaySessionID: req.RolePlaySessionID,
			ArenaID:           req.ArenaID,
			CoopPartyID:       req.CoopPartyID,
			ViewerCount:       0,
			LastEventAt:       &now,
			StartedAt:         &now,
			AllowReplay:       req.AllowReplay,
		}

		err := database.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&live).Error; err != nil {
				return err
			}

			payload, _ := json.Marshal(gin.H{
				"status":  live.Status,
				"channel": live.ChannelKey,
				"message": "live session created",
			})
			event := models.LiveEvent{
				LiveSessionID: live.Id,
				Sequence:      1,
				EventType:     "status",
				AuthorType:    "system",
				AuthorName:    "system",
				Payload:       datatypes.JSON(payload),
			}
			return tx.Create(&event).Error
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create live session"})
			return
		}

		appResponseCache().InvalidateNamespace(c.Request.Context(), "live")
		if hydrated, hydrateErr := repository.NewLiveRepository(database).GetSessionOwnedByID(c.Request.Context(), live.Id, userID); hydrateErr == nil {
			live = *hydrated
		}
		c.JSON(http.StatusCreated, gin.H{"session": live})
	}
}

func createCoopLiveSession(ctx context.Context, database *gorm.DB, ownerID uint, party *models.CoopParty) error {
	if party == nil {
		return nil
	}

	coopPartyID := party.Id
	now := time.Now()
	live := models.LiveSession{
		OwnerID:     ownerID,
		ChannelKey:  "live-" + randomHex(12),
		Mode:        constants.ModeCoop,
		Status:      constants.LiveStatusStreaming,
		CoopPartyID: &coopPartyID,
		ViewerCount: 0,
		LastEventAt: &now,
		StartedAt:   &now,
		AllowReplay: true,
	}

	err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&live).Error; err != nil {
			return err
		}

		payload, _ := json.Marshal(gin.H{
			"status":   live.Status,
			"channel":  live.ChannelKey,
			"message":  "coop live session created",
			"coopCode": party.Code,
		})
		event := models.LiveEvent{
			LiveSessionID: live.Id,
			Sequence:      1,
			EventType:     constants.LiveEventTypeStatus,
			AuthorType:    constants.AuthorTypeSystem,
			AuthorName:    constants.AuthorTypeSystem,
			Payload:       datatypes.JSON(payload),
		}
		return tx.Create(&event).Error
	})
	if err != nil {
		return err
	}
	appResponseCache().InvalidateNamespace(ctx, "live")
	return nil
}

func coopPartyCarriesRolePlay(party *models.CoopParty) bool {
	if party == nil {
		return false
	}
	return party.RolePlaySessionID != nil || party.Mode == constants.ModeRolePlayIA
}

func endLiveSessionsByCoopParty(ctx context.Context, database *gorm.DB, coopPartyID uint, message string) (int, error) {
	var sessions []models.LiveSession
	if err := database.WithContext(ctx).
		Where("coop_party_id = ? AND status <> ?", coopPartyID, constants.LiveStatusEnded).
		Find(&sessions).Error; err != nil {
		return 0, err
	}

	ended := 0
	now := time.Now()
	for _, session := range sessions {
		err := database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var last models.LiveEvent
			nextSequence := 1
			if err := tx.Where("live_session_id = ?", session.Id).Order("sequence DESC").First(&last).Error; err == nil {
				nextSequence = last.Sequence + 1
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			payload, _ := json.Marshal(gin.H{
				"status":      constants.LiveStatusEnded,
				"channel":     session.ChannelKey,
				"message":     message,
				"coopPartyId": coopPartyID,
			})
			if err := tx.Create(&models.LiveEvent{
				LiveSessionID: session.Id,
				Sequence:      nextSequence,
				EventType:     constants.LiveEventTypeStatus,
				AuthorType:    constants.AuthorTypeSystem,
				AuthorName:    constants.AuthorTypeSystem,
				Payload:       datatypes.JSON(payload),
			}).Error; err != nil {
				return err
			}

			return tx.Model(&models.LiveSession{}).
				Where("id = ?", session.Id).
				Updates(map[string]any{
					"status":        constants.LiveStatusEnded,
					"ended_at":      &now,
					"last_event_at": &now,
				}).Error
		})
		if err != nil {
			return ended, err
		}
		ended++
	}

	appResponseCache().InvalidateNamespace(ctx, "live")
	return ended, nil
}

func getLiveSession(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, ok := findOwnedLiveSessionByID(c, database)
		if !ok {
			return
		}

		c.JSON(http.StatusOK, gin.H{"session": session})
	}
}

func getPublicLiveSession(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, ok := findPublicLiveSessionByID(c, database)
		if !ok {
			return
		}

		c.JSON(http.StatusOK, gin.H{"session": session})
	}
}

func getLiveSessionEvents(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, ok := findOwnedLiveSessionByID(c, database)
		if !ok {
			return
		}

		after, _ := strconv.Atoi(c.DefaultQuery("after", "0"))
		events, err := liveEventsAfter(c, database, session.Id, after, limitFromQuery(c))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list live events"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"events": events})
	}
}

func getPublicLiveSessionEvents(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, ok := findPublicLiveSessionByID(c, database)
		if !ok {
			return
		}

		after, _ := strconv.Atoi(c.DefaultQuery("after", "0"))
		events, err := liveEventsAfter(c, database, session.Id, after, limitFromQuery(c))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot list public live events"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"events": events})
	}
}

func getLiveHistory(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		after, _ := strconv.Atoi(c.DefaultQuery("after", "0"))
		session, events, err := newLiveService(database).HistoryByChannel(c.Request.Context(), c.Param("channel"), currentUserID(c), after, limitFromQuery(c))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"session": session, "events": events})
	}
}

func getPublicLiveHistory(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, ok := findPublicLiveSessionByChannel(c, database)
		if !ok {
			return
		}

		after, _ := strconv.Atoi(c.DefaultQuery("after", "0"))
		events, err := liveEventsAfter(c, database, session.Id, after, limitFromQuery(c))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot read public live history"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"session": session, "events": events})
	}
}

func streamLiveChannel(database *gorm.DB) gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return func(c *gin.Context) {
		session, ok := findOwnedLiveSessionByChannel(c, database)
		if !ok {
			return
		}

		if !websocket.IsWebSocketUpgrade(c.Request) {
			flusher, ok := c.Writer.(http.Flusher)
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming non supporte"})
				return
			}

			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")
			c.Header("X-Accel-Buffering", "no")

			database.WithContext(c.Request.Context()).
				Model(&models.LiveSession{}).
				Where("id = ?", session.Id).
				UpdateColumn("viewer_count", gorm.Expr("viewer_count + ?", 1))
			defer database.WithContext(c.Request.Context()).
				Model(&models.LiveSession{}).
				Where("id = ? AND viewer_count > 0", session.Id).
				UpdateColumn("viewer_count", gorm.Expr("viewer_count - ?", 1))

			lastSequence, _ := strconv.Atoi(c.DefaultQuery("after", "0"))
			if err := writeSSEEvent(c, flusher, "live_connected", gin.H{
				"type":    "live_connected",
				"session": session,
			}); err != nil {
				return
			}

			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				events, err := liveEventsAfter(c, database, session.Id, lastSequence, 100)
				if err != nil {
					_ = writeSSEEvent(c, flusher, "error", gin.H{"type": "error", "error": "cannot read live events"})
					return
				}

				for _, event := range events {
					if err := writeSSEEvent(c, flusher, "live_event", gin.H{
						"type":  "live_event",
						"event": event,
					}); err != nil {
						return
					}
					lastSequence = event.Sequence
				}

				var refreshed models.LiveSession
				err = database.WithContext(c.Request.Context()).
					Where("id = ? AND owner_id = ?", session.Id, currentUserID(c)).
					First(&refreshed).Error
				if err != nil {
					return
				}
				if refreshed.Status == constants.LiveStatusEnded {
					_ = writeSSEEvent(c, flusher, "live_ended", gin.H{
						"type":    "live_ended",
						"session": refreshed,
					})
					return
				}

				select {
				case <-ticker.C:
					if err := writeSSEEvent(c, flusher, "heartbeat", gin.H{"type": "heartbeat", "ts": time.Now().UTC()}); err != nil {
						return
					}
				case <-c.Request.Context().Done():
					return
				}
			}
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		database.WithContext(c.Request.Context()).
			Model(&models.LiveSession{}).
			Where("id = ?", session.Id).
			UpdateColumn("viewer_count", gorm.Expr("viewer_count + ?", 1))
		defer database.WithContext(c.Request.Context()).
			Model(&models.LiveSession{}).
			Where("id = ? AND viewer_count > 0", session.Id).
			UpdateColumn("viewer_count", gorm.Expr("viewer_count - ?", 1))

		lastSequence, _ := strconv.Atoi(c.DefaultQuery("after", "0"))
		if err := writeWebSocketJSON(conn, gin.H{
			"type":    "live_connected",
			"session": session,
		}); err != nil {
			return
		}

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			events, err := liveEventsAfter(c, database, session.Id, lastSequence, 100)
			if err != nil {
				_ = writeWebSocketJSON(conn, gin.H{"type": "error", "error": "cannot read live events"})
				return
			}

			for _, event := range events {
				if err := writeWebSocketJSON(conn, gin.H{
					"type":  "live_event",
					"event": event,
				}); err != nil {
					return
				}
				lastSequence = event.Sequence
			}

			var refreshed models.LiveSession
			err = database.WithContext(c.Request.Context()).
				Where("id = ? AND owner_id = ?", session.Id, currentUserID(c)).
				First(&refreshed).Error
			if err != nil {
				return
			}
			if refreshed.Status == "ended" {
				_ = writeWebSocketJSON(conn, gin.H{
					"type":    "live_ended",
					"session": refreshed,
				})
				return
			}

			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
					return
				}
			case <-c.Request.Context().Done():
				return
			}
		}
	}
}

func streamPublicLiveChannel(database *gorm.DB) gin.HandlerFunc {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return func(c *gin.Context) {
		session, ok := findPublicLiveSessionByChannel(c, database)
		if !ok {
			return
		}

		if !websocket.IsWebSocketUpgrade(c.Request) {
			flusher, ok := c.Writer.(http.Flusher)
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming non supporte"})
				return
			}

			c.Header("Content-Type", "text/event-stream")
			c.Header("Cache-Control", "no-cache")
			c.Header("Connection", "keep-alive")
			c.Header("X-Accel-Buffering", "no")

			database.WithContext(c.Request.Context()).
				Model(&models.LiveSession{}).
				Where("id = ?", session.Id).
				UpdateColumn("viewer_count", gorm.Expr("viewer_count + ?", 1))
			defer database.WithContext(c.Request.Context()).
				Model(&models.LiveSession{}).
				Where("id = ? AND viewer_count > 0", session.Id).
				UpdateColumn("viewer_count", gorm.Expr("viewer_count - ?", 1))

			lastSequence, _ := strconv.Atoi(c.DefaultQuery("after", "0"))
			if err := writeSSEEvent(c, flusher, "live_connected", gin.H{
				"type":    "live_connected",
				"session": session,
			}); err != nil {
				return
			}

			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				events, err := liveEventsAfter(c, database, session.Id, lastSequence, 100)
				if err != nil {
					_ = writeSSEEvent(c, flusher, "error", gin.H{"type": "error", "error": "cannot read live events"})
					return
				}

				for _, event := range events {
					if err := writeSSEEvent(c, flusher, "live_event", gin.H{
						"type":  "live_event",
						"event": event,
					}); err != nil {
						return
					}
					lastSequence = event.Sequence
				}

				var refreshed models.LiveSession
				err = database.WithContext(c.Request.Context()).
					Where("id = ?", session.Id).
					First(&refreshed).Error
				if err != nil {
					return
				}
				if refreshed.Status == constants.LiveStatusEnded {
					_ = writeSSEEvent(c, flusher, "live_ended", gin.H{
						"type":    "live_ended",
						"session": refreshed,
					})
					return
				}

				select {
				case <-ticker.C:
					if err := writeSSEEvent(c, flusher, "heartbeat", gin.H{"type": "heartbeat", "ts": time.Now().UTC()}); err != nil {
						return
					}
				case <-c.Request.Context().Done():
					return
				}
			}
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		database.WithContext(c.Request.Context()).
			Model(&models.LiveSession{}).
			Where("id = ?", session.Id).
			UpdateColumn("viewer_count", gorm.Expr("viewer_count + ?", 1))
		defer database.WithContext(c.Request.Context()).
			Model(&models.LiveSession{}).
			Where("id = ? AND viewer_count > 0", session.Id).
			UpdateColumn("viewer_count", gorm.Expr("viewer_count - ?", 1))

		lastSequence, _ := strconv.Atoi(c.DefaultQuery("after", "0"))
		if err := writeWebSocketJSON(conn, gin.H{
			"type":    "live_connected",
			"session": session,
		}); err != nil {
			return
		}

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			events, err := liveEventsAfter(c, database, session.Id, lastSequence, 100)
			if err != nil {
				_ = writeWebSocketJSON(conn, gin.H{"type": "error", "error": "cannot read live events"})
				return
			}

			for _, event := range events {
				if err := writeWebSocketJSON(conn, gin.H{
					"type":  "live_event",
					"event": event,
				}); err != nil {
					return
				}
				lastSequence = event.Sequence
			}

			var refreshed models.LiveSession
			err = database.WithContext(c.Request.Context()).
				Where("id = ?", session.Id).
				First(&refreshed).Error
			if err != nil {
				return
			}
			if refreshed.Status == constants.LiveStatusEnded {
				_ = writeWebSocketJSON(conn, gin.H{
					"type":    "live_ended",
					"session": refreshed,
				})
				return
			}

			select {
			case <-ticker.C:
				if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
					return
				}
			case <-c.Request.Context().Done():
				return
			}
		}
	}
}

func endLiveSession(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		session, ok := findOwnedLiveSessionByID(c, database)
		if !ok {
			return
		}

		now := time.Now()
		err := database.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			var lastEvent models.LiveEvent
			lastSequence := 0
			if err := tx.
				Where("live_session_id = ?", session.Id).
				Order("sequence DESC").
				First(&lastEvent).Error; err == nil {
				lastSequence = lastEvent.Sequence
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			payload, _ := json.Marshal(gin.H{
				"status":  "ended",
				"channel": session.ChannelKey,
				"message": "live session ended",
			})
			event := models.LiveEvent{
				LiveSessionID: session.Id,
				Sequence:      lastSequence + 1,
				EventType:     "status",
				AuthorType:    "system",
				AuthorName:    "system",
				Payload:       datatypes.JSON(payload),
			}
			if err := tx.Create(&event).Error; err != nil {
				return err
			}

			return tx.Model(&models.LiveSession{}).
				Where("id = ? AND owner_id = ?", session.Id, currentUserID(c)).
				Updates(map[string]any{
					"status":        "ended",
					"ended_at":      &now,
					"last_event_at": &now,
				}).Error
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot end live session"})
			return
		}

		appResponseCache().InvalidateNamespace(c.Request.Context(), "live")
		session.Status = "ended"
		session.EndedAt = &now
		session.LastEventAt = &now
		c.JSON(http.StatusOK, gin.H{"session": session})
	}
}

func me(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.Users
		err := database.WithContext(c.Request.Context()).
			Where("id = ?", currentUserID(c)).
			First(&user).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"user": userResponse(&user)})
	}
}

func updateMe(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req updateUserRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user payload"})
			return
		}

		users := repository.NewUserRepository(database)
		user, err := users.GetByID(c.Request.Context(), currentUserID(c))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get user"})
			return
		}

		fields := make(map[string]any)

		if req.Email != nil {
			email := strings.TrimSpace(*req.Email)
			if email == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
				return
			}
			if email != user.Email {
				existingUser, err := users.GetByEmail(c.Request.Context(), email)
				if err == nil && existingUser != nil && existingUser.Id != user.Id {
					c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
					return
				}
				if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot check email"})
					return
				}
			}
			fields["email"] = email
		}

		if req.Pseudo != nil {
			pseudo := strings.TrimSpace(*req.Pseudo)
			if pseudo == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "pseudo is required"})
				return
			}
			fields["pseudo"] = pseudo
		}

		if req.BirthdayDate != nil {
			fields["birthday_date"] = strings.TrimSpace(*req.BirthdayDate)
		}

		if req.Avatar != nil {
			fields["avatar"] = strings.TrimSpace(*req.Avatar)
		}

		if req.NewPassword != nil {
			newPassword := *req.NewPassword
			currentPassword := ""
			if req.CurrentPassword != nil {
				currentPassword = *req.CurrentPassword
			}
			if strings.TrimSpace(newPassword) == "" || strings.TrimSpace(currentPassword) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "currentPassword and newPassword are required"})
				return
			}
			if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(currentPassword)); err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid current password"})
				return
			}
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot hash password"})
				return
			}
			fields["password"] = string(hashedPassword)
		}

		if len(fields) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no user fields to update"})
			return
		}

		if err := users.UpdateFields(c.Request.Context(), user.Id, fields); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot update user"})
			return
		}

		updatedUser, err := users.GetByID(c.Request.Context(), user.Id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot reload user"})
			return
		}

		token, err := makeJWT(updatedUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"updated":    true,
			"token":      token,
			"expires_in": int64((24 * time.Hour).Seconds()),
			"user":       userResponse(updatedUser),
		})
	}
}

func updateUserProgression(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
			return
		}

		var req progressionRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid progression payload"})
			return
		}
		if req.XP == nil && req.Coin == nil && req.XPDelta == nil && req.CoinDelta == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "xp, coin, xpDelta or coinDelta is required"})
			return
		}

		users := repository.NewUserRepository(database)
		updatedUser, err := users.UpdateProgression(c.Request.Context(), id, req.XP, req.Coin, req.XPDelta, req.CoinDelta)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"updated": true,
			"reason":  req.Reason,
			"user":    userResponse(updatedUser),
		})
	}
}

func hydrateBattleProfiles(c *gin.Context, database *gorm.DB, userID uint, req *battleRequest) error {
	if req.IA1ProfileID != 0 {
		var profile models.IAProfile
		err := database.WithContext(c.Request.Context()).
			Where("id = ? AND owner_id = ?", req.IA1ProfileID, userID).
			First(&profile).Error
		if err != nil {
			return fmt.Errorf("ia1 profile not found")
		}
		applyProfileToBattleRequest(&profile, req, 1)
	}

	if req.IA2ProfileID != 0 {
		var profile models.IAProfile
		err := database.WithContext(c.Request.Context()).
			Where("id = ? AND owner_id = ?", req.IA2ProfileID, userID).
			First(&profile).Error
		if err != nil {
			return fmt.Errorf("ia2 profile not found")
		}
		applyProfileToBattleRequest(&profile, req, 2)
	}

	return nil
}

func applyProfileToBattleRequest(profile *models.IAProfile, req *battleRequest, slot int) {
	if slot == 1 {
		req.Provider1 = defaultString(req.Provider1, profile.ProviderName)
		req.IAModels = defaultString(req.IAModels, profile.ModelName)
		req.IA1Name = defaultString(req.IA1Name, profile.Name)
		req.IA1Personality = defaultString(req.IA1Personality, profile.Personality)
		req.IA1Mindset = defaultString(req.IA1Mindset, profile.Mindset)
		req.IA1Style = defaultString(req.IA1Style, profile.Style)
		req.IA1Goal = defaultString(req.IA1Goal, profile.Goal)
		req.IA1Weakness = defaultString(req.IA1Weakness, profile.Weakness)
		return
	}

	req.Provider2 = defaultString(req.Provider2, profile.ProviderName)
	req.IAModels2 = defaultString(req.IAModels2, profile.ModelName)
	req.IA2Name = defaultString(req.IA2Name, profile.Name)
	req.IA2Personality = defaultString(req.IA2Personality, profile.Personality)
	req.IA2Mindset = defaultString(req.IA2Mindset, profile.Mindset)
	req.IA2Style = defaultString(req.IA2Style, profile.Style)
	req.IA2Goal = defaultString(req.IA2Goal, profile.Goal)
	req.IA2Weakness = defaultString(req.IA2Weakness, profile.Weakness)
}

func findOwnedBattle(c *gin.Context, database *gorm.DB) (models.BattleSave, bool) {
	var battle models.BattleSave
	err := database.WithContext(c.Request.Context()).
		Where("id = ? AND owner_id = ?", c.Param("id"), currentUserID(c)).
		First(&battle).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "battle not found"})
		return battle, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get battle"})
		return battle, false
	}

	return battle, true
}

func findPublicBattle(c *gin.Context, database *gorm.DB) (models.BattleSave, bool) {
	var battle models.BattleSave
	err := publicBattleScope(database.WithContext(c.Request.Context()).Model(&models.BattleSave{})).
		Where("battle_saves.id = ?", c.Param("id")).
		First(&battle).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "public battle not found"})
		return battle, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get public battle"})
		return battle, false
	}

	return battle, true
}

func findOwnedIAProfile(c *gin.Context, database *gorm.DB) (models.IAProfile, bool) {
	var profile models.IAProfile
	err := database.WithContext(c.Request.Context()).
		Where("id = ? AND owner_id = ?", c.Param("id"), currentUserID(c)).
		First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "ia profile not found"})
		return profile, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get ia profile"})
		return profile, false
	}

	return profile, true
}

func newBattleService(database *gorm.DB) *service.BattleService {
	return service.NewBattleService(
		repository.NewBattleRepository(database),
		repository.NewQuestRepository(database),
		repository.NewIAProfileRepository(database),
		service.NewLiveServiceWithCache(repository.NewLiveRepository(database), appResponseCache()),
		repository.NewAIUsageRepository(database),
		newAIOrchestrator(database),
	)
}

func newQuestService(database *gorm.DB) *service.QuestService {
	return service.NewQuestServiceWithCache(repository.NewQuestRepository(database), appResponseCache())
}

func newArenaService(database *gorm.DB) *service.ArenaService {
	return service.NewArenaService(
		repository.NewArenaRepository(database),
		repository.NewBattleRepository(database),
	)
}

func newRolePlayService(database *gorm.DB) *service.RolePlayService {
	return service.NewRolePlayService(
		repository.NewRolePlayRepository(database),
		repository.NewQuestRepository(database),
		repository.NewAIUsageRepository(database),
		repository.NewRolePlayCharacterRepository(database),
		newAIOrchestrator(database),
	)
}

func newRolePlayCharacterService(database *gorm.DB) *service.RolePlayCharacterService {
	return service.NewRolePlayCharacterService(
		repository.NewRolePlayCharacterRepository(database),
		repository.NewQuestRepository(database),
	)
}

func newCoopService(database *gorm.DB) *service.CoopService {
	return service.NewCoopServiceWithCache(
		repository.NewCoopRepository(database),
		repository.NewRolePlayCharacterRepository(database),
		appResponseCache(),
	)
}

func newLiveService(database *gorm.DB) *service.LiveService {
	return service.NewLiveServiceWithCache(repository.NewLiveRepository(database), appResponseCache())
}

func appResponseCache() *nexuscache.ResponseCache {
	responseCacheOnce.Do(func() {
		responseCache = nexuscache.NewResponseCache(nexuscache.NewRedisServiceFromEnv(), "battleia")
	})
	return responseCache
}

func toServiceBattleRequest(req battleRequest) service.BattleRequest {
	return service.BattleRequest{
		Provider1:            req.Provider1,
		Provider2:            req.Provider2,
		IAKey1:               req.IAKey1,
		IAKey2:               req.IAKey2,
		IAModels:             req.IAModels,
		IAModels2:            req.IAModels2,
		IA1ProfileID:         req.IA1ProfileID,
		IA2ProfileID:         req.IA2ProfileID,
		IA1Name:              req.IA1Name,
		IA1Personality:       req.IA1Personality,
		IA1Mindset:           req.IA1Mindset,
		IA1Style:             req.IA1Style,
		IA1Goal:              req.IA1Goal,
		IA1Weakness:          req.IA1Weakness,
		IA2Name:              req.IA2Name,
		IA2Personality:       req.IA2Personality,
		IA2Mindset:           req.IA2Mindset,
		IA2Style:             req.IA2Style,
		IA2Goal:              req.IA2Goal,
		IA2Weakness:          req.IA2Weakness,
		Question:             req.Question,
		QuestID:              req.QuestID,
		Title:                req.Title,
		Visibility:           req.Visibility,
		TotalRounds:          req.TotalRounds,
		RoundDurationSeconds: req.RoundDurationSeconds,
		PublicVote:           req.PublicVote,
		BillingMode:          req.BillingMode,
	}
}

func toBattleQuestInput(req battleQuestRequest) service.BattleQuestInput {
	return service.BattleQuestInput{
		Slug:     req.Slug,
		Title:    req.Title,
		Content:  req.Content,
		Level:    req.Level,
		Point:    req.Point,
		Theme:    req.Theme,
		Xp:       req.Xp,
		Coin:     req.Coin,
		Mode:     req.Mode,
		Source:   req.Source,
		Status:   req.Status,
		Metadata: req.Metadata,
	}
}

func toRolePlayArcInputs(reqs []rolePlayQuestArcRequest) []service.RolePlayQuestArcInput {
	if len(reqs) == 0 {
		return nil
	}
	arcs := make([]service.RolePlayQuestArcInput, 0, len(reqs))
	for _, req := range reqs {
		arcs = append(arcs, service.RolePlayQuestArcInput{
			Position:  req.Position,
			Slug:      req.Slug,
			Title:     req.Title,
			Summary:   req.Summary,
			Objective: req.Objective,
			Prompt:    req.Prompt,
			Metadata:  req.Metadata,
			Chapters:  toRolePlayChapterInputs(req.Chapters),
		})
	}
	return arcs
}

func toRolePlayChapterInputs(reqs []rolePlayQuestChapterRequest) []service.RolePlayQuestChapterInput {
	if len(reqs) == 0 {
		return nil
	}
	chapters := make([]service.RolePlayQuestChapterInput, 0, len(reqs))
	for _, req := range reqs {
		chapters = append(chapters, service.RolePlayQuestChapterInput{
			Position:      req.Position,
			Slug:          req.Slug,
			Title:         req.Title,
			Summary:       req.Summary,
			Objective:     req.Objective,
			IntroPrompt:   req.IntroPrompt,
			SuccessPrompt: req.SuccessPrompt,
			FailurePrompt: req.FailurePrompt,
			IsBoss:        req.IsBoss,
			Xp:            req.Xp,
			Coin:          req.Coin,
			Metadata:      req.Metadata,
		})
	}
	return chapters
}

func toRolePlayQuestInput(req rolePlayQuestRequest) service.RolePlayQuestInput {
	return service.RolePlayQuestInput{
		Slug:     req.Slug,
		Title:    req.Title,
		Summary:  req.Summary,
		Prompt:   req.Prompt,
		Theme:    req.Theme,
		Level:    req.Level,
		Xp:       req.Xp,
		Coin:     req.Coin,
		Source:   req.Source,
		Status:   req.Status,
		Metadata: req.Metadata,
		Arcs:     toRolePlayArcInputs(req.Arcs),
	}
}

func toRolePlayCharacterInput(req rolePlayCharacterRequest) service.RolePlayCharacterInput {
	return service.RolePlayCharacterInput{
		Name:         req.Name,
		Class:        req.Class,
		Archetype:    req.Archetype,
		Origin:       req.Origin,
		Race:         req.Race,
		Species:      req.Species,
		Alignment:    req.Alignment,
		Personality:  req.Personality,
		Background:   req.Background,
		PersonalGoal: defaultString(req.PersonalGoal, req.PersonalGoalAlt),
		Goal:         req.Goal,
		Level:        req.Level,
		HeroImageID:  req.HeroImageID,
		ImageURL:     req.ImageURL,
		Attributes:   req.Attributes,
		Skills:       req.Skills,
		Traits:       req.Traits,
		Inventory:    req.Inventory,
		Health:       req.Health,
		MaxHealth:    firstNonZero(req.MaxHealth, req.MaxHealthAlt),
		Stress:       req.Stress,
		Fatigue:      req.Fatigue,
		Morale:       req.Morale,
	}
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func parseUintParam(c *gin.Context, name string) (uint, error) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || value == 0 {
		return 0, fmt.Errorf("invalid %s", name)
	}

	return uint(value), nil
}

func findOwnedLiveSessionByID(c *gin.Context, database *gorm.DB) (models.LiveSession, bool) {
	var session models.LiveSession
	err := repository.PreloadLiveRolePlayQuestVisuals(database.WithContext(c.Request.Context())).
		Where("id = ? AND owner_id = ?", c.Param("id"), currentUserID(c)).
		First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "live session not found"})
		return session, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get live session"})
		return session, false
	}
	repository.ApplyLiveSessionImageCompatibility(&session)

	return session, true
}

func findPublicLiveSessionByID(c *gin.Context, database *gorm.DB) (models.LiveSession, bool) {
	var session models.LiveSession
	err := repository.PreloadLiveRolePlayQuestVisuals(publicLiveSessionScope(database.WithContext(c.Request.Context()).Model(&models.LiveSession{}))).
		Where("live_sessions.id = ?", c.Param("id")).
		First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "public live session not found"})
		return session, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get public live session"})
		return session, false
	}
	repository.ApplyLiveSessionImageCompatibility(&session)

	return session, true
}

func findOwnedLiveSessionByChannel(c *gin.Context, database *gorm.DB) (models.LiveSession, bool) {
	var session models.LiveSession
	err := repository.PreloadLiveRolePlayQuestVisuals(database.WithContext(c.Request.Context())).
		Where("channel_key = ? AND owner_id = ?", c.Param("channel"), currentUserID(c)).
		First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "live channel not found"})
		return session, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get live channel"})
		return session, false
	}
	repository.ApplyLiveSessionImageCompatibility(&session)

	return session, true
}

func findPublicLiveSessionByChannel(c *gin.Context, database *gorm.DB) (models.LiveSession, bool) {
	var session models.LiveSession
	err := repository.PreloadLiveRolePlayQuestVisuals(publicLiveSessionScope(database.WithContext(c.Request.Context()).Model(&models.LiveSession{}))).
		Where("live_sessions.channel_key = ?", c.Param("channel")).
		First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "public live channel not found"})
		return session, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get public live channel"})
		return session, false
	}
	repository.ApplyLiveSessionImageCompatibility(&session)

	return session, true
}

func findPublicArena(c *gin.Context, database *gorm.DB) (models.BattleArena, bool) {
	var arena models.BattleArena
	err := publicArenaScope(database.WithContext(c.Request.Context()).Model(&models.BattleArena{})).
		Where("battle_arenas.code = ?", c.Param("code")).
		First(&arena).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "public arena not found"})
		return arena, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot get public arena"})
		return arena, false
	}

	return arena, true
}

func publicBattleScope(tx *gorm.DB) *gorm.DB {
	return tx.Where(`battle_saves.visibility = ?
		OR EXISTS (
			SELECT 1
			FROM battle_arenas
			WHERE battle_arenas.battle_save_id = battle_saves.id
				AND battle_arenas.deleted_at IS NULL
				AND battle_arenas.allow_spectators = ?
		)
		OR EXISTS (
			SELECT 1
			FROM live_sessions
			WHERE live_sessions.battle_save_id = battle_saves.id
				AND live_sessions.deleted_at IS NULL
				AND (live_sessions.status <> ? OR live_sessions.allow_replay = ?)
		)`, constants.VisibilityPublic, true, constants.LiveStatusEnded, true)
}

func publicArenaScope(tx *gorm.DB) *gorm.DB {
	return tx.Where("battle_arenas.allow_spectators = ?", true).
		Where("battle_arenas.status NOT IN ?", []string{constants.ArenaStatusFinished, constants.ArenaStatusClosed})
}

func publicLiveSessionScope(tx *gorm.DB) *gorm.DB {
	return tx.Where("(live_sessions.status <> ? OR live_sessions.allow_replay = ?)", constants.LiveStatusEnded, true).
		Where("(live_sessions.battle_save_id IS NOT NULL OR live_sessions.arena_id IS NOT NULL OR live_sessions.role_play_session_id IS NOT NULL OR live_sessions.coop_party_id IS NOT NULL)")
}

func validateLiveAttachments(c *gin.Context, database *gorm.DB, userID uint, req liveSessionRequest) error {
	attachments := 0

	if req.BattleSaveID != nil {
		attachments++
		var count int64
		if err := database.WithContext(c.Request.Context()).
			Model(&models.BattleSave{}).
			Where("id = ? AND owner_id = ?", *req.BattleSaveID, userID).
			Count(&count).Error; err != nil || count == 0 {
			return fmt.Errorf("battleSaveId not found for user")
		}
	}

	if req.RolePlaySessionID != nil {
		attachments++
		var count int64
		if err := database.WithContext(c.Request.Context()).
			Model(&models.RolePlaySession{}).
			Where("id = ? AND owner_id = ?", *req.RolePlaySessionID, userID).
			Count(&count).Error; err != nil || count == 0 {
			return fmt.Errorf("rolePlaySessionId not found for user")
		}
	}

	if req.ArenaID != nil {
		attachments++
		var count int64
		if err := database.WithContext(c.Request.Context()).
			Model(&models.BattleArena{}).
			Where("id = ? AND host_user_id = ?", *req.ArenaID, userID).
			Count(&count).Error; err != nil || count == 0 {
			return fmt.Errorf("arenaId not found for user")
		}
	}

	if req.CoopPartyID != nil {
		attachments++
		var count int64
		if err := database.WithContext(c.Request.Context()).
			Model(&models.CoopParty{}).
			Where("id = ? AND host_user_id = ?", *req.CoopPartyID, userID).
			Count(&count).Error; err != nil || count == 0 {
			return fmt.Errorf("coopPartyId not found for user")
		}
	}

	if attachments > 1 {
		return fmt.Errorf("only one live attachment is allowed")
	}

	return nil
}

func liveEventsAfter(c *gin.Context, database *gorm.DB, sessionID uint, after int, limit int) ([]models.LiveEvent, error) {
	cacheKey := fmt.Sprintf("events:session:%d:after:%d:limit:%d", sessionID, after, limit)
	var events []models.LiveEvent
	if appResponseCache().GetJSON(c.Request.Context(), "live", cacheKey, &events) {
		return events, nil
	}
	err := database.WithContext(c.Request.Context()).
		Where("live_session_id = ? AND sequence > ?", sessionID, after).
		Order("sequence ASC").
		Limit(limit).
		Find(&events).Error
	if err == nil {
		appResponseCache().SetJSON(c.Request.Context(), "live", cacheKey, events, time.Second)
	}
	return events, err
}

func writeWebSocketJSON(conn *websocket.Conn, payload any) error {
	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	return conn.WriteJSON(payload)
}

func writeSSEEvent(c *gin.Context, flusher http.Flusher, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if _, err := c.Writer.Write([]byte("event: " + event + "\n")); err != nil {
		return err
	}
	if _, err := c.Writer.Write([]byte("data: " + string(data) + "\n\n")); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func randomHex(byteCount int) string {
	buffer := make([]byte, byteCount)
	if _, err := rand.Read(buffer); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}

	return hex.EncodeToString(buffer)
}

func bindPayload(c *gin.Context, out any) error {
	if strings.Contains(c.GetHeader("Content-Type"), "application/json") {
		return c.ShouldBindJSON(out)
	}

	return c.ShouldBind(out)
}

func writeNDJSON(c *gin.Context, flusher http.Flusher, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	_, _ = c.Writer.Write(data)
	_, _ = c.Writer.Write([]byte("\n"))
	flusher.Flush()
}

func battleEventKey(event scenarios.BattleStreamEvent) string {
	return fmt.Sprintf("%d:%s:%s", event.Round, event.IA, event.Type)
}

func providerURL(name string) (string, error) {
	return service.ProviderURL(name)
}

func providerTestErrorMessage(providerName string, err error) string {
	message := err.Error()
	if strings.Contains(message, "status=401") || strings.Contains(message, "401 Unauthorized") {
		switch strings.ToLower(strings.TrimSpace(providerName)) {
		case "xia", "xai", "x-ai":
			return "xAI a refuse la requete en 401. Verifie que la cle commence par xai-, qu'elle est active sur console.x.ai, et colle la cle seule sans le prefixe Bearer. Detail provider: " + message
		default:
			return "Provider refuse en 401. Verifie la cle API et colle la cle seule sans le prefixe Bearer. Detail provider: " + message
		}
	}
	return message
}

func applyQuestFilters(c *gin.Context, query *gorm.DB) *gorm.DB {
	status := c.Query("status")
	if status == "" {
		status = "published"
	}
	if status != "all" {
		query = query.Where("status = ?", status)
	}
	if theme := c.Query("theme"); theme != "" {
		query = query.Where("theme = ?", theme)
	}
	if level := c.Query("level"); level != "" {
		query = query.Where("level = ?", level)
	}

	return query
}

func limitFromQuery(c *gin.Context) int {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}

	return limit
}

func offsetFromQuery(c *gin.Context) int {
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		return 0
	}

	return offset
}

func paginatedQuestResponse(key string, items any, total int64, limit int, offset int) gin.H {
	nextOffset := offset + limit
	hasMore := int64(nextOffset) < total
	if !hasMore {
		nextOffset = offset
	}

	return gin.H{
		key:          items,
		"data":       items,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
		"hasMore":    hasMore,
		"nextOffset": nextOffset,
	}
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}

func mapKeys(value map[string]any) []string {
	if value == nil {
		return nil
	}
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	return keys
}

func aiProviderTestTimeout() time.Duration {
	value, err := strconv.Atoi(getEnv("AI_PROVIDER_TEST_TIMEOUT_SECONDS", "20"))
	if err != nil || value <= 0 {
		return 20 * time.Second
	}

	return time.Duration(value) * time.Second
}

func aiProviderGenerationTimeout() time.Duration {
	value, err := strconv.Atoi(getEnv("AI_PROVIDER_GENERATION_TIMEOUT_SECONDS", "60"))
	if err != nil || value <= 0 {
		return 60 * time.Second
	}

	return time.Duration(value) * time.Second
}

func truncateString(value string, maxLength int) string {
	if maxLength <= 0 || len(value) <= maxLength {
		return value
	}

	return value[:maxLength]
}

func slugify(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, slug)
	if slug == "" {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	return slug
}
func prepareInitialRolePlaySnapshot(snapshot map[string]any, title string, scenarioPrompt string) map[string]any {
	if snapshot == nil {
		snapshot = map[string]any{}
	}

	hero := mapFromAny(snapshot["hero"])

	heroName := strings.TrimSpace(fmt.Sprint(hero["name"]))
	if heroName == "" || heroName == "<nil>" {
		heroName = "Héros"
		hero["name"] = heroName
	}

	heroClass := strings.TrimSpace(fmt.Sprint(hero["class"]))
	if heroClass == "" || heroClass == "<nil>" {
		heroClass = strings.TrimSpace(fmt.Sprint(hero["characterClass"]))
	}
	if heroClass == "" || heroClass == "<nil>" {
		heroClass = "Aventurier"
		hero["class"] = heroClass
	}

	heroRace := strings.TrimSpace(fmt.Sprint(hero["race"]))
	if heroRace == "" || heroRace == "<nil>" {
		heroRace = "Inconnue"
		hero["race"] = heroRace
	}

	snapshot["hero"] = hero

	if _, exists := snapshot["activeNpcs"]; !exists {
		snapshot["activeNpcs"] = []map[string]any{
			{
				"id":             "npc_mentor_01",
				"name":           "Astra Vey",
				"role":           "Guide de mission",
				"archetype":      "mentor",
				"mood":           "observatrice",
				"personality":    "calme, lucide, exigeante",
				"goal":           "Tester le héros et l’orienter sans choisir à sa place.",
				"relationToHero": 5,
				"trust":          5,
				"fear":           0,
				"respect":        5,
			},
			{
				"id":             "npc_rival_01",
				"name":           "Kael Morvane",
				"role":           "Rival local",
				"archetype":      "rival",
				"mood":           "méfiant",
				"personality":    "sec, provocateur, calculateur",
				"goal":           "Tester les faiblesses du héros.",
				"relationToHero": -3,
				"trust":          0,
				"fear":           0,
				"respect":        2,
			},
		}
	}

	if _, exists := snapshot["npcRelations"]; !exists {
		snapshot["npcRelations"] = map[string]any{
			"npc_mentor_01": map[string]any{
				"trust":          5,
				"respect":        5,
				"fear":           0,
				"relationToHero": 5,
			},
			"npc_rival_01": map[string]any{
				"trust":          0,
				"respect":        2,
				"fear":           0,
				"relationToHero": -3,
			},
		}
	}

	if _, exists := snapshot["sceneDialogues"]; !exists {
		questTitle := strings.TrimSpace(title)
		if questTitle == "" {
			questTitle = "cette mission"
		}

		snapshot["sceneDialogues"] = []map[string]any{
			{
				"speakerId":   "npc_mentor_01",
				"speakerName": "Astra Vey",
				"speakerType": "npc",
				"tone":        "calme",
				"content":     fmt.Sprintf("%s, ton profil de %s attire déjà l’attention. Avant d’avancer dans %s, je veux voir comment tu réagis.", heroName, heroClass, questTitle),
			},
			{
				"speakerId":   "hero",
				"speakerName": heroName,
				"speakerType": "hero",
				"tone":        "déterminé",
				"content":     "Je suis prêt. Donne-moi les faits, pas les légendes.",
			},
			{
				"speakerId":   "npc_rival_01",
				"speakerName": "Kael Morvane",
				"speakerType": "npc",
				"tone":        "provocateur",
				"content":     "Les faits ? Tu n’as encore rien prouvé. Ici, chaque mauvais choix laisse une trace.",
			},
		}
	}

	return snapshot
}

func mapFromAny(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}

	if typed, ok := value.(map[string]any); ok {
		return typed
	}

	if typed, ok := value.(map[string]interface{}); ok {
		result := map[string]any{}
		for key, item := range typed {
			result[key] = item
		}
		return result
	}

	return map[string]any{}
}
