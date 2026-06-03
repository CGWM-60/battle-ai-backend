package adapters

import "testing"


func TestLocalChatCompletionsURL(t *testing.T) {
	cases := map[string]string{
		"http://localhost:11434":                       "http://localhost:11434/v1/chat/completions",
		"http://localhost:11434/":                      "http://localhost:11434/v1/chat/completions",
		"http://localhost:1234/v1":                     "http://localhost:1234/v1/chat/completions",
		"http://localhost:1234/v1/chat/completions":    "http://localhost:1234/v1/chat/completions",
		"https://example.test/openai/chat/completions": "https://example.test/openai/chat/completions",
	}

	for input, expected := range cases {
		got, err := LocalChatCompletionsURL(input)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", input, err)
		}
		if got != expected {
			t.Fatalf("expected %s, got %s", expected, got)
		}
	}
}

func TestLocalChatCompletionsURLRejectsInvalidEndpoint(t *testing.T) {
	for _, input := range []string{"", "localhost:11434", "not a url"} {
		if _, err := LocalChatCompletionsURL(input); err == nil {
			t.Fatalf("expected invalid endpoint %q to fail", input)
		}
	}
}

func TestResolveProviderUsesEnvKeyForRemote(t *testing.T) {
	adapter := NewAIProviderAdapter(func(providerType string) string {
		if providerType != "openai" {
			t.Fatalf("unexpected provider %s", providerType)
		}
		return "env-key"
	})
	resolved, err := adapter.resolve(GenerateRequest{ProviderType: "openai", Model: "gpt-4o-mini"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved.apiKey != "env-key" {
		t.Fatalf("expected env-key, got %q", resolved.apiKey)
	}
	if resolved.isLocal {
		t.Fatal("openai must not resolve as local")
	}
}

func TestResolveProviderRequiresRemoteKey(t *testing.T) {
	adapter := NewAIProviderAdapter(func(providerType string) string { return "" })
	if _, err := adapter.resolve(GenerateRequest{ProviderType: "openai", Model: "gpt-4o-mini"}); err == nil {
		t.Fatal("expected missing remote key error")
	}
}

func TestResolveLocalProviderDoesNotRequireKey(t *testing.T) {
	adapter := NewAIProviderAdapter(nil)
	resolved, err := adapter.resolve(GenerateRequest{
		ProviderType:  "ollama",
		Model:         "llama3.2",
		LocalEndpoint: "http://localhost:11434",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resolved.isLocal {
		t.Fatal("ollama must resolve as local")
	}
	if resolved.apiKey != "local-no-key" {
		t.Fatalf("expected local-no-key, got %q", resolved.apiKey)
	}
}
