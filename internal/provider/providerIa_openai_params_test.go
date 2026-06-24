package provider

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenAIGpt5MiniNeverUsesMaxTokens(t *testing.T) {
	payload, err := marshalOpenAICompatibleChatRequest(
		"gpt-5-mini",
		[]ProviderMessage{{Role: "user", Content: "OK"}},
		false,
		32,
	)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	raw := string(payload)
	if strings.Contains(raw, `"max_tokens"`) {
		t.Fatalf("gpt-5-mini must not use max_tokens: %s", raw)
	}
	if !strings.Contains(raw, `"max_completion_tokens"`) {
		t.Fatalf("gpt-5-mini must use max_completion_tokens: %s", raw)
	}
}

func TestOpenRouterKeepsMaxTokens(t *testing.T) {
	payload, err := marshalOpenAICompatibleChatRequest(
		"openai/gpt-4o-mini",
		[]ProviderMessage{{Role: "user", Content: "OK"}},
		false,
		32,
	)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if _, ok := decoded["max_tokens"]; !ok {
		t.Fatalf("openrouter-style model should keep max_tokens: %s", string(payload))
	}
	if _, ok := decoded["max_completion_tokens"]; ok {
		t.Fatalf("openrouter-style model must not use max_completion_tokens: %s", string(payload))
	}
}