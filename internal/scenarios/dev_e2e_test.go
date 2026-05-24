package scenarios

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"
)

const devMistralURL = "https://api.mistral.ai/v1/chat/completions"

func TestDevMistralBattleAndRolePlayFlows(t *testing.T) {
	if os.Getenv("RUN_DEV_E2E") != "1" {
		t.Skip("set RUN_DEV_E2E=1 to run live Mistral dev flow")
	}

	loadDevEnv(t)

	apiKey := strings.TrimSpace(os.Getenv("DEV_API_KEY"))
	model := strings.TrimSpace(os.Getenv("DEV_MODEL"))
	if apiKey == "" || model == "" {
		t.Skip("DEV_API_KEY/DEV_MODEL absents")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Run("battle ia round 1 and round 2", func(t *testing.T) {
		events := make([]BattleStreamEvent, 0)
		ias := []models.BattleIAConfig{
			devIA("Le Stratège", apiKey, model, "analytique", "pragmatique"),
			devIA("La Contradictrice", apiKey, model, "critique", "direct"),
		}

		err := RunBattleScenarioSingleRoundStreamWithDurationContext(
			ctx,
			"Un assistant IA doit-il refuser une réponse si elle rend le joueur moins autonome ?",
			ias,
			nil,
			1,
			3,
			15*time.Second,
			func(event BattleStreamEvent) {
				events = append(events, event)
			},
		)
		if err != nil {
			if isRateLimitError(err) {
				t.Skipf("Mistral rate limit during round 1: %v", err)
			}
			t.Fatalf("round 1 failed: %v", err)
		}
		assertDoneForIA(t, events, "Le Stratège", 1)
		assertDoneForIA(t, events, "La Contradictrice", 1)

		history := eventsToHistory(events)
		events = events[:0]
		err = RunBattleScenarioSingleRoundStreamWithDurationContext(
			ctx,
			"Un assistant IA doit-il refuser une réponse si elle rend le joueur moins autonome ?",
			ias,
			history,
			2,
			3,
			15*time.Second,
			func(event BattleStreamEvent) {
				events = append(events, event)
			},
		)
		if err != nil {
			if isRateLimitError(err) {
				t.Skipf("Mistral rate limit during round 2: %v", err)
			}
			t.Fatalf("round 2 failed: %v", err)
		}
		assertDoneForIA(t, events, "Le Stratège", 2)
		assertDoneForIA(t, events, "La Contradictrice", 2)
	})

	t.Run("roleplay game master json", func(t *testing.T) {
		p := provider.NewsProvider(apiKey, devMistralURL, model)
		response, err := chatWithRetry(ctx, p, []provider.ProviderMessage{
			{
				Role:    "system",
				Content: "Tu es un maître du jeu IA. Réponds uniquement avec un objet JSON valide.",
			},
			{
				Role: "user",
				Content: `Crée le début d'une aventure RP cyberpunk courte.
Réponds avec:
{
  "narration": "...",
  "sceneTitle": "...",
  "sceneSummary": "...",
  "objective": "...",
  "zoneLabel": "...",
  "completed": false,
  "options": [
    {"id":"...","label":"...","caption":"...","promptSeed":"...","accentKey":"primary","iconKey":"terminal"}
  ]
}
Il faut exactement 3 options.`,
			},
		})
		if err != nil {
			if isRateLimitError(err) {
				t.Skipf("Mistral rate limit during roleplay generation: %v", err)
			}
			t.Fatalf("roleplay generation failed: %v", err)
		}

		var payload struct {
			Narration    string `json:"narration"`
			SceneTitle   string `json:"sceneTitle"`
			SceneSummary string `json:"sceneSummary"`
			Objective    string `json:"objective"`
			Completed    bool   `json:"completed"`
			Options      []struct {
				ID         string `json:"id"`
				Label      string `json:"label"`
				Caption    string `json:"caption"`
				PromptSeed string `json:"promptSeed"`
			} `json:"options"`
		}
		if err := json.Unmarshal([]byte(extractJSONObject(response)), &payload); err != nil {
			t.Fatalf("roleplay json invalid: %v\nresponse=%s", err, response)
		}
		if payload.Narration == "" || payload.SceneTitle == "" || payload.Objective == "" {
			t.Fatalf("roleplay json incomplete: %+v", payload)
		}
		if payload.Completed {
			t.Fatalf("roleplay should not complete at opening")
		}
		if len(payload.Options) != 3 {
			t.Fatalf("expected 3 roleplay options, got %d", len(payload.Options))
		}
	})
}

func chatWithRetry(ctx context.Context, p *provider.Provider, messages []provider.ProviderMessage) (string, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*15) * time.Second)
		}

		response, err := p.Chat(ctx, messages)
		if err == nil {
			return response, nil
		}
		lastErr = err
		if !isRateLimitError(err) {
			return "", err
		}
	}
	return "", lastErr
}

func isRateLimitError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "status=429")
}

func devIA(name string, apiKey string, model string, personality string, style string) models.BattleIAConfig {
	return models.BattleIAConfig{
		Name:         name,
		Provider:     provider.NewsProvider(apiKey, devMistralURL, model),
		ProviderName: "mistral",
		ModelName:    model,
		Personality:  personality,
		Mindset:      "débat structuré",
		Style:        style,
		Goal:         "faire avancer le raisonnement",
		Weakness:     "peut trop condenser ses nuances",
	}
}

func assertDoneForIA(t *testing.T, events []BattleStreamEvent, name string, round int) {
	t.Helper()
	for _, event := range events {
		if event.IA == name && event.Round == round && event.Done && event.Error == "" {
			return
		}
	}
	t.Fatalf("missing done event for %s round %d; events=%d", name, round, len(events))
}

func eventsToHistory(events []BattleStreamEvent) []models.BattleRoundMessage {
	buffers := make(map[string]*strings.Builder)
	order := make([]string, 0)
	for _, event := range events {
		if event.Content == "" || event.Error != "" {
			continue
		}
		key := event.IA
		if _, ok := buffers[key]; !ok {
			buffers[key] = &strings.Builder{}
			order = append(order, key)
		}
		buffers[key].WriteString(event.Content)
	}

	history := make([]models.BattleRoundMessage, 0, len(order))
	for _, ia := range order {
		content := strings.TrimSpace(buffers[ia].String())
		if content == "" {
			continue
		}
		history = append(history, models.BattleRoundMessage{
			IA:      ia,
			Round:   1,
			Content: content,
		})
	}
	return history
}

func extractJSONObject(raw string) string {
	clean := strings.TrimSpace(raw)
	start := strings.Index(clean, "{")
	end := strings.LastIndex(clean, "}")
	if start >= 0 && end > start {
		return clean[start : end+1]
	}
	return clean
}

func loadDevEnv(t *testing.T) {
	t.Helper()
	candidates := []string{
		".env.dev",
		filepath.Join("..", "Flutter dev", "battle_ia", ".env.dev"),
		filepath.Join("..", "..", "..", "Flutter dev", "battle_ia", ".env.dev"),
		filepath.Join("..", "..", "..", "..", "Flutter dev", "battle_ia", ".env.dev"),
	}
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				t.Setenv(key, value)
			}
		}
		return
	}
}
