package provider

import (
	"context"
	"strings"
	"testing"
)

func TestProviderWithEmptyKeyDoesNotBecomeMock(t *testing.T) {
	p := NewsProvider("", "https://api.openai.com/v1/chat/completions", "gpt-4o-mini")
	if p.isMock() {
		t.Fatal("empty api key must not activate implicit mock mode")
	}
	_, err := p.Chat(context.Background(), []ProviderMessage{{Role: "user", Content: "OK"}})
	if err == nil {
		t.Fatal("expected error for missing api key")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "api key is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProviderWithEmptyURLDoesNotBecomeMock(t *testing.T) {
	p := NewsProvider("sk-test", "", "gpt-4o-mini")
	if p.isMock() {
		t.Fatal("empty url must not activate implicit mock mode")
	}
	_, err := p.Chat(context.Background(), []ProviderMessage{{Role: "user", Content: "OK"}})
	if err == nil {
		t.Fatal("expected error for missing url")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "url is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExplicitMockProviderOnly(t *testing.T) {
	p := NewMockProvider("openai", "gpt-test")
	if !p.isMock() {
		t.Fatal("explicit mock provider must be marked as mock")
	}
}