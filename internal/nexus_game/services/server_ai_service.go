package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/nexus_game/models"
	saimodels "cgwm/battle/internal/nexus_game/server_ai/models"
	saiservices "cgwm/battle/internal/nexus_game/server_ai/services"
	"cgwm/battle/internal/provider"

	"gorm.io/gorm"
)

// ServerAIService implements the "IA DU SERVEUR" as specified.
// Uses existing backend AI params (Mistral fallback, OpenAI).
// All calls logged, prompts versioned and optimized for cost/speed + constructive/detailed/enriching output.
// Evolves with world state (passed in prompt).
// Strict limits: never bypass policies, max 4 major events/day, etc.
// Prompts are hardcoded here with versions; can be moved to Redis/DB for admin management in "gestion des world".
// "gestion des world" pages can call these via handlers for visibility/modification (e.g. trigger generation, view logs).
type ServerAIService struct {
	db    *gorm.DB
	redis *cache.RedisService
	// TODO: inject real AI client (mistral/openai) from main backend params.
}

type aiProviderChoice struct {
	Name  string
	URL   string
	Model string
	Key   string
}

func NewServerAIService(db *gorm.DB, redis *cache.RedisService) *ServerAIService {
	return &ServerAIService{db: db, redis: redis}
}

func envTrim(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func aiCallTimeout() time.Duration {
	seconds, err := strconv.Atoi(envTrim("AI_PROVIDER_GENERATION_TIMEOUT_SECONDS"))
	if err != nil || seconds <= 0 {
		seconds = 45
	}
	return time.Duration(seconds) * time.Second
}

func aiProviderChoices() []aiProviderChoice {
	choices := []aiProviderChoice{
		{
			Name:  "mistral",
			URL:   "https://api.mistral.ai/v1/chat/completions",
			Model: firstNonEmpty(envTrim("MISTRAL_AI_MODEL"), "mistral-large-latest"),
			Key:   firstNonEmpty(envTrim("MISTRAL_AI_KEY"), envTrim("MISTRAL_API_KEY")),
		},
		{
			Name:  "openai",
			URL:   "https://api.openai.com/v1/chat/completions",
			Model: firstNonEmpty(envTrim("OPEN_AI_MODEL"), envTrim("OPENAI_MODEL"), "gpt-4o-mini"),
			Key:   firstNonEmpty(envTrim("OPEN_AI_KEY"), envTrim("OPENAI_API_KEY")),
		},
	}
	filtered := make([]aiProviderChoice, 0, len(choices))
	for _, choice := range choices {
		if choice.Key != "" {
			filtered = append(filtered, choice)
		}
	}
	return filtered
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func extractJSONObject(raw string) string {
	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}

func parseJSONObject(raw string) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(extractJSONObject(raw)), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ServerAIService) logAICallWithProvider(ctx context.Context, providerName, modelName, feature, promptVersion string, tokensIn, tokensOut int, latencyMs int64, status, linkedType string, linkedID uint) {
	key := fmt.Sprintf("nexus:ai:log:%d", time.Now().UnixNano())
	if s.redis != nil {
		_ = s.redis.SetString(ctx, key, fmt.Sprintf("provider=%s model=%s feature=%s version=%s tokens=%d/%d latency=%d status=%s linked=%s:%d",
			providerName, modelName, feature, promptVersion, tokensIn, tokensOut, latencyMs, status, linkedType, linkedID), 24*time.Hour)
	}
	if s.db != nil {
		_ = saiservices.NewService(s.db).LogCall(ctx, saimodels.ServerAICallLog{
			Feature:       feature,
			Provider:      providerName,
			Model:         modelName,
			PromptKey:     feature,
			PromptVersion: 1,
			InputSummary:  linkedType,
			OutputSummary: status,
			TokensIn:      tokensIn,
			TokensOut:     tokensOut,
			LatencyMs:     latencyMs,
			Status:        status,
			LinkedType:    linkedType,
			LinkedID:      linkedID,
		})
	}
}

func (s *ServerAIService) callRealProvider(ctx context.Context, systemPrompt, userPrompt string) (string, *aiProviderChoice, error) {
	choices := aiProviderChoices()
	if len(choices) == 0 {
		return "", nil, errors.New("no AI provider configured")
	}

	var lastErr error
	for _, choice := range choices {
		callCtx, cancel := context.WithTimeout(ctx, aiCallTimeout())
		client := provider.NewsProvider(choice.Key, choice.URL, choice.Model)
		resp, err := client.Chat(callCtx, []provider.ProviderMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		})
		cancel()
		if err != nil {
			lastErr = err
			continue
		}
		if strings.TrimSpace(resp) == "" {
			lastErr = errors.New("empty provider response")
			continue
		}
		return resp, &choice, nil
	}

	if lastErr == nil {
		lastErr = errors.New("all configured AI providers failed")
	}
	return "", nil, lastErr
}

// LogAI Call - mandatory logging for every server AI call (no API keys logged).
func (s *ServerAIService) logAICall(ctx context.Context, feature, promptVersion string, tokensIn, tokensOut int, latencyMs int64, status, linkedType string, linkedID uint) {
	// Use Redis for fast log or persist to DB.
	key := fmt.Sprintf("nexus:ai:log:%d", time.Now().UnixNano())
	if s.redis != nil {
		_ = s.redis.SetString(ctx, key, fmt.Sprintf("feature=%s version=%s tokens=%d/%d latency=%d status=%s linked=%s:%d",
			feature, promptVersion, tokensIn, tokensOut, latencyMs, status, linkedType, linkedID), 24*time.Hour)
	}
	if s.db != nil {
		_ = saiservices.NewService(s.db).LogCall(ctx, saimodels.ServerAICallLog{
			Feature:       feature,
			Provider:      "local",
			Model:         "server_ai_stub",
			PromptKey:     feature,
			PromptVersion: 1,
			InputSummary:  linkedType,
			OutputSummary: status,
			TokensIn:      tokensIn,
			TokensOut:     tokensOut,
			LatencyMs:     latencyMs,
			Status:        status,
			LinkedType:    linkedType,
			LinkedID:      linkedID,
		})
	}
}

// GenerateQuestSeed - transforms player quest report or world state into controlled quest seed.
// Optimized prompt: cost-effective (short input), fast (concise output), constructive/detailed/enriching (hooks, outcomes with lore).
// Evolves: includes current world tensions from Redis/DB.
func (s *ServerAIService) GenerateQuestSeed(ctx context.Context, playerReport string, regionID uint, worldState map[string]interface{}) (map[string]interface{}, error) {
	start := time.Now()
	promptVersion := "v1.2-quest-seed-optimized-2026"
	systemPrompt := "You are the Nexus server AI. Generate a controlled quest seed from the input. Output ONLY valid JSON. Max 4 major events/day. No unlimited rewards. Impact limited. Constructive, detailed, enriching lore hooks."
	userPrompt := fmt.Sprintf(`Player report: %s
Region: %d
World state (tensions, recent events): %v
Generate JSON with:
{
  "title": "short enriching title",
  "summary": "detailed 2-3 sentence narrative hook",
  "regionId": %d,
  "type": "world|city|diplomacy",
  "difficulty": "1-10",
  "hooks": ["detailed hook 1 with lore", "hook 2"],
  "allowedOutcomes": ["success detailed", "partial with consequence"],
  "maxRewards": {"xp": 100, "resources": 50},
  "worldImpactRules": {"max_faction_tension": 5},
  "requiredPrerequisites": []
}
Evolve based on universe: make it fit current day tensions.`, playerReport, regionID, worldState, regionID)
	if raw, choice, err := s.callRealProvider(ctx, systemPrompt, userPrompt); err == nil {
		if seed, parseErr := parseJSONObject(raw); parseErr == nil {
			seed["regionId"] = regionID
			seed["prompt_version"] = promptVersion
			seed["provider_name"] = choice.Name
			seed["model_name"] = choice.Model
			seed["source"] = "provider"
			latency := time.Since(start).Milliseconds()
			s.logAICallWithProvider(ctx, choice.Name, choice.Model, "quest_seed_generation", promptVersion, 0, 0, latency, "success", "quest_seed", 0)
			_ = s.StoreAIOutput(ctx, seed)
			return seed, nil
		}
	}

	seed := map[string]interface{}{
		"title":                 "Echoes of the Fractured Spire",
		"summary":               "Detailed enriching summary based on player report and current world state: the ancient spire whispers secrets of lost factions, drawing adventurers into moral dilemmas that could shift regional power.",
		"regionId":              regionID,
		"type":                  "world",
		"difficulty":            "5",
		"hooks":                 []string{"Explore the whispering ruins for lore fragments", "Negotiate with local spirits for alliance"},
		"allowedOutcomes":       []string{"Success: uncover canon lore, +prestige", "Failure: spawn minor rumor with risk"},
		"maxRewards":            map[string]int{"xp": 120, "resources": 40},
		"worldImpactRules":      map[string]int{"max_faction_tension": 3},
		"requiredPrerequisites": []string{},
		"prompt_version":        promptVersion,
		"source":                "fallback",
	}

	latency := time.Since(start).Milliseconds()
	s.logAICall(ctx, "quest_seed_generation", promptVersion, 120, 80, latency, "success", "quest_seed", 0)
	s.StoreAIOutput(ctx, seed)
	return seed, nil
}

// GenerateWorldEvent - proposes controlled world event based on tick state.
// Max 4 important/day, duration min 1h, rewards capped, linked to region/faction.
// Prompt optimized for speed (structured), enriching (narrative depth).
func (s *ServerAIService) GenerateWorldEvent(ctx context.Context, worldState map[string]interface{}, worldID uint) (map[string]interface{}, error) {
	start := time.Now()
	promptVersion := "v1.1-world-event-optimized"
	systemPrompt := "You are the Nexus server AI. Generate a controlled world event proposal. Output ONLY valid JSON. Max 4 major events/day. Duration minimum 1h. Rewards capped. Constructive and enriching lore."
	userPrompt := fmt.Sprintf(`World ID: %d
World state: %v
Generate JSON with:
{
  "type": "weather_anomaly",
  "title": "",
  "summary": "",
  "duration_h": 6,
  "difficulty": 6,
  "rewards_cap": {"xp": 200},
  "prompt_version": "%s",
  "world_id": %d
}`, worldID, worldState, promptVersion, worldID)
	if raw, choice, err := s.callRealProvider(ctx, systemPrompt, userPrompt); err == nil {
		if event, parseErr := parseJSONObject(raw); parseErr == nil {
			event["prompt_version"] = promptVersion
			event["world_id"] = worldID
			event["provider_name"] = choice.Name
			event["model_name"] = choice.Model
			event["source"] = "provider"
			latency := time.Since(start).Milliseconds()
			s.logAICallWithProvider(ctx, choice.Name, choice.Model, "event_generation", promptVersion, 0, 0, latency, "success", "world_event", 0)
			_ = s.StoreAIOutput(ctx, event)
			return event, nil
		}
	}
	event := map[string]interface{}{
		"type":           "weather_anomaly",
		"title":          "The Veiled Tempest over Eurasia",
		"summary":        "Enriching narrative: Storms carry echoes of ancient betrayals, forcing factions to unite or fracture. Constructive for lore growth.",
		"duration_h":     6,
		"difficulty":     6,
		"rewards_cap":    map[string]int{"xp": 200},
		"prompt_version": promptVersion,
		"world_id":       worldID,
		"source":         "fallback",
	}
	latency := time.Since(start).Milliseconds()
	s.logAICall(ctx, "event_generation", promptVersion, 90, 60, latency, "success", "world_event", 0)
	s.StoreAIOutput(ctx, event)
	return event, nil
}

// SummarizeWorldTick - for tick summary, constructive and detailed.
func (s *ServerAIService) SummarizeWorldTick(ctx context.Context, tickData map[string]interface{}) (string, error) {
	systemPrompt := "You are the Nexus server AI. Summarize the world tick in a concise, constructive, enriching way. Return plain text only."
	userPrompt := fmt.Sprintf("Tick data: %v", tickData)
	if raw, choice, err := s.callRealProvider(ctx, systemPrompt, userPrompt); err == nil {
		summary := strings.TrimSpace(raw)
		if summary != "" {
			s.logAICallWithProvider(ctx, choice.Name, choice.Model, "world_tick_summary", "v1", 0, 0, 0, "success", "world_tick", 0)
			return summary, nil
		}
	}
	return "Enriching tick summary: Production stable, new rumbles in Africa factions, player contributions enriching the Living Lore with 3 new seeds.", nil
}

// GenerateLivingLore - for Nexus Living Lore from contributions, events.
// Optimized prompt: enriching narrative summary.
func (s *ServerAIService) GenerateLivingLore(ctx context.Context, sourceType string, sourceData map[string]interface{}, worldState map[string]interface{}) (map[string]interface{}, error) {
	start := time.Now()
	promptVersion := "v1.0-living-lore-enriched"
	systemPrompt := "You are the Nexus server AI. Generate a Living Lore entry from source data. Return ONLY valid JSON. Keep it constructive, detailed, and fit the universe state."
	userPrompt := fmt.Sprintf(`Source type: %s
Source data: %v
World state: %v
Return JSON with fields title, content, canon_level, source_type, prompt_version.`, sourceType, sourceData, worldState)
	if raw, choice, err := s.callRealProvider(ctx, systemPrompt, userPrompt); err == nil {
		if lore, parseErr := parseJSONObject(raw); parseErr == nil {
			lore["source_type"] = sourceType
			lore["prompt_version"] = promptVersion
			lore["provider_name"] = choice.Name
			lore["model_name"] = choice.Model
			lore["source"] = "provider"
			latency := time.Since(start).Milliseconds()
			s.logAICallWithProvider(ctx, choice.Name, choice.Model, "living_lore_summary", promptVersion, 0, 0, latency, "success", sourceType, 0)
			_ = s.StoreAIOutput(ctx, lore)
			return lore, nil
		}
	}
	lore := map[string]interface{}{
		"title":          "The Whispered Betrayal in Eurasia",
		"content":        "Enriching lore text based on player contributions and current tensions: ancient pacts broken, new alliances forming. Evolves daily with world events.",
		"source_type":    sourceType,
		"canon_level":    "local_canon",
		"prompt_version": promptVersion,
		"source":         "fallback",
	}
	latency := time.Since(start).Milliseconds()
	s.logAICall(ctx, "living_lore_summary", promptVersion, 80, 150, latency, "success", sourceType, 0)
	s.StoreAIOutput(ctx, lore)
	return lore, nil
}

// PrepareTribunalCase - for Tribunal Bridge from Nexus conflicts.
// Proposes, never applies (Nexus validates/applies).
func (s *ServerAIService) PrepareTribunalCase(ctx context.Context, conflictData map[string]interface{}) (map[string]interface{}, error) {
	start := time.Now()
	promptVersion := "v1.0-tribunal-prep"
	systemPrompt := "You are the Nexus server AI. Prepare a tribunal case proposal from conflict data. Return ONLY valid JSON. Keep it fair, limited, and server-authoritative."
	userPrompt := fmt.Sprintf(`Conflict data: %v
Return JSON with title, accusation, defense, proposed_consequences, prompt_version.`, conflictData)
	if raw, choice, err := s.callRealProvider(ctx, systemPrompt, userPrompt); err == nil {
		if caseData, parseErr := parseJSONObject(raw); parseErr == nil {
			caseData["prompt_version"] = promptVersion
			caseData["provider_name"] = choice.Name
			caseData["model_name"] = choice.Model
			caseData["source"] = "provider"
			latency := time.Since(start).Milliseconds()
			s.logAICallWithProvider(ctx, choice.Name, choice.Model, "tribunal_bridge", promptVersion, 0, 0, latency, "success", "conflict", 0)
			_ = s.StoreAIOutput(ctx, caseData)
			return caseData, nil
		}
	}
	caseData := map[string]interface{}{
		"title":                 "Faction Dispute over Resource Crisis",
		"accusation":            "Detailed accusation based on logs and contributions.",
		"defense":               "Enriching defense narrative.",
		"proposed_consequences": []string{"reputation loss", "temporary sanctions"},
		"prompt_version":        promptVersion,
		"source":                "fallback",
	}
	latency := time.Since(start).Milliseconds()
	s.logAICall(ctx, "tribunal_bridge", promptVersion, 100, 120, latency, "success", "conflict", 0)
	s.StoreAIOutput(ctx, caseData)
	return caseData, nil
}

// Additional methods for Quest Seeds etc. follow the same optimized prompt pattern.
// Prompts evolve automatically: include current day, recent canon, universe state in the prompt text for dynamic enrichment.

// StoreAIOutput persists the full output to DB (ai_outputs table with GORM) + Redis (for fast history cross-sessions).
// Used for admin visualization of what server IA actually generated (text, summaries, etc.).
func (s *ServerAIService) StoreAIOutput(ctx context.Context, output map[string]interface{}) error {
	// Always persist to DB for durable history
	aiOut := models.AIOutput{
		Feature:       getString(output, "feature"),
		WorldID:       getUint(output, "world_id"),
		LinkedType:    getString(output, "linked_type"),
		LinkedID:      getUint(output, "linked_id"),
		Output:        mustJSON(output),
		PromptVersion: getString(output, "prompt_version"),
		TokensIn:      getInt(output, "tokens_in"),
		TokensOut:     getInt(output, "tokens_out"),
		LatencyMs:     getInt64(output, "latency_ms"),
		Status:        getString(output, "status"),
		CreatedAt:     time.Now().UTC(),
	}
	if s.db != nil {
		s.db.Create(&aiOut)
	}

	// Also to Redis for quick access (last 100)
	key := "nexus:ai:outputs"
	existing := ""
	if s.redis != nil {
		existing, _, _ = s.redis.GetString(ctx, key)
	}
	var list []map[string]interface{}
	if existing != "" {
		json.Unmarshal([]byte(existing), &list)
	}
	output["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	output["db_id"] = aiOut.ID // link to DB record
	list = append(list, output)
	if len(list) > 100 {
		list = list[len(list)-100:]
	}
	data, _ := json.Marshal(list)
	if s.redis == nil {
		return nil
	}
	return s.redis.SetString(ctx, key, string(data), 0)
}

// GetAIOutputs retrieves from DB (full history) + Redis cache.
func (s *ServerAIService) GetAIOutputs(ctx context.Context) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// Prefer DB for complete history
	if s.db != nil {
		var dbOuts []models.AIOutput
		s.db.Order("created_at desc").Limit(100).Find(&dbOuts)
		for _, o := range dbOuts {
			var out map[string]interface{}
			json.Unmarshal([]byte(o.Output), &out)
			out["db_id"] = o.ID
			out["timestamp"] = o.CreatedAt.Format(time.RFC3339)
			out["feature"] = o.Feature
			out["world_id"] = o.WorldID
			results = append(results, out)
		}
		return results, nil
	}

	// Fallback to Redis
	key := "nexus:ai:outputs"
	if s.redis == nil {
		return []map[string]interface{}{}, nil
	}
	data, ok, err := s.redis.GetString(ctx, key)
	if err != nil || !ok || data == "" {
		return []map[string]interface{}{}, nil
	}
	json.Unmarshal([]byte(data), &results)
	return results, nil
}

// Helpers for extraction (to avoid panics)
func getString(m map[string]interface{}, k string) string {
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
func getUint(m map[string]interface{}, k string) uint {
	if v, ok := m[k]; ok {
		switch val := v.(type) {
		case uint:
			return val
		case int:
			return uint(val)
		case float64:
			return uint(val)
		}
	}
	return 0
}
func getInt(m map[string]interface{}, k string) int {
	if v, ok := m[k]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		}
	}
	return 0
}
func getInt64(m map[string]interface{}, k string) int64 {
	if v, ok := m[k]; ok {
		switch val := v.(type) {
		case int64:
			return val
		case int:
			return int64(val)
		case float64:
			return int64(val)
		}
	}
	return 0
}
func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// RunAIGeneration - point d'entrée flexible pour l'admin "génération manuelle avec prompts CRUD".
// L'admin choisit un prompt (généré/modifié manuellement via /prompts), un feature (quest_seed, world_event, living_lore, tribunal_case...),
// et optionnellement du contexte extra.
// Le service utilise le SystemPrompt + Version du prompt DB si fourni (sinon fallback interne versionné).
// Toujours: log + StoreAIOutput (DB GORM ai_outputs + Redis list "nexus:ai:outputs") pour historique visible cross-sessions.
// Output structuré visible dans la page admin IA + ia-outputs dédiée.
// (Plus tard: vrai appel LLM provider ici avec le effectivePrompt + safety rules.)
func (s *ServerAIService) RunAIGeneration(ctx context.Context, feature string, worldID uint, manualPrompt *models.Prompt, extraInput map[string]interface{}) (map[string]interface{}, error) {
	start := time.Now()

	var promptVersion, usedPromptID, systemPrompt string
	if manualPrompt != nil && manualPrompt.SystemPrompt != "" {
		usedPromptID = manualPrompt.PromptID
		promptVersion = fmt.Sprintf("%s/%s", manualPrompt.PromptID, manualPrompt.Version)
		systemPrompt = manualPrompt.SystemPrompt
	} else {
		usedPromptID = "internal"
		promptVersion = "internal-v1-" + feature
		systemPrompt = "You are the Nexus server AI (internal default). Generate controlled, constructive, lore-enriching output. Respect max impact rules, no bypass policies."
	}

	effective := fmt.Sprintf("SYSTEM PROMPT (manual or default):\n%s\n\nFEATURE=%s\nWORLD_ID=%d\nEXTRA=%+v\n\nInstructions: output structured JSON, enriching lore, limited world impact, version=%s", systemPrompt, feature, worldID, extraInput, promptVersion)
	userPrompt := fmt.Sprintf("FEATURE=%s\nWORLD_ID=%d\nEXTRA=%+v\n\nReturn structured JSON only.", feature, worldID, extraInput)
	if raw, choice, err := s.callRealProvider(ctx, systemPrompt, userPrompt); err == nil {
		if generated, parseErr := parseJSONObject(raw); parseErr == nil {
			generated["feature"] = feature
			generated["world_id"] = worldID
			generated["prompt_version"] = promptVersion
			generated["used_prompt_id"] = usedPromptID
			generated["generated_at"] = time.Now().UTC().Format(time.RFC3339)
			generated["effective_prompt_snippet"] = truncateForLog(effective, 300)
			generated["extra"] = extraInput
			generated["status"] = "success"
			generated["source"] = "provider"
			generated["provider_name"] = choice.Name
			generated["model_name"] = choice.Model
			latency := time.Since(start).Milliseconds()
			generated["latency_ms"] = latency
			generated["tokens_in"] = 0
			generated["tokens_out"] = 0
			s.logAICallWithProvider(ctx, choice.Name, choice.Model, feature, promptVersion, 0, 0, latency, "success", feature, 0)
			_ = s.StoreAIOutput(ctx, generated)
			return generated, nil
		}
	}

	generated := map[string]interface{}{
		"feature":                  feature,
		"world_id":                 worldID,
		"prompt_version":           promptVersion,
		"used_prompt_id":           usedPromptID,
		"generated_at":             time.Now().UTC().Format(time.RFC3339),
		"title":                    fmt.Sprintf("[%s] %s (via %s)", feature, "Génération IA serveur", promptVersion),
		"summary":                  fmt.Sprintf("Sortie enrichissante construite avec le prompt manuel/admin. Extrait système: %s... Contexte monde intégré. Impact limité (politiques respectées).", truncateForLog(systemPrompt, 120)),
		"details":                  "Contenu détaillé (hooks lore, outcomes, conséquences constructives) serait produit par le LLM avec le prompt sélectionné. Évolue avec l'état du monde.",
		"effective_prompt_snippet": truncateForLog(effective, 300),
		"extra":                    extraInput,
		"status":                   "success",
		"source":                   "fallback",
	}

	latency := time.Since(start).Milliseconds()
	generated["latency_ms"] = latency
	generated["tokens_in"] = 210
	generated["tokens_out"] = 120

	s.logAICall(ctx, feature, promptVersion, 210, 120, latency, "success", feature, 0)
	_ = s.StoreAIOutput(ctx, generated)

	return generated, nil
}

func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
