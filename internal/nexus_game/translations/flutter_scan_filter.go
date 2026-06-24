package translations

import (
	"path/filepath"
	"strings"
)

// FlutterScanSkipReason classifies rejected entries.
type FlutterScanSkipReason string

const (
	SkipTechnical     FlutterScanSkipReason = "technical"
	SkipAssets        FlutterScanSkipReason = "assets"
	SkipRoutes        FlutterScanSkipReason = "routes"
	SkipProviders     FlutterScanSkipReason = "providers"
	SkipStorageKeys   FlutterScanSkipReason = "storage_keys"
	SkipEmpty         FlutterScanSkipReason = "empty"
	SkipDeprecated    FlutterScanSkipReason = "deprecated"
)

var flutterProviderNames = []string{
	"openai", "grok", "mistral", "claude", "gemini", "openrouter",
	"gpt-", "claude-", "mistral-", "grok-",
}

// isTranslatableFlutterEntry returns false for technical/non-UI strings.
func isTranslatableFlutterEntry(entry NormalizedFlutterScanEntry) (bool, FlutterScanSkipReason) {
	fullKey := strings.TrimSpace(entry.FullKey)
	text := strings.TrimSpace(entry.Text)
	defaultText := strings.TrimSpace(entry.DefaultText)
	value := defaultText
	if value == "" {
		value = text
	}
	if fullKey == "" || value == "" {
		return false, SkipEmpty
	}

	domain, _ := splitDomainAndKey(fullKey)
	if !isRetainedTranslation(domain, fullKey) {
		return false, SkipDeprecated
	}

	lowerKey := strings.ToLower(fullKey)
	lowerText := strings.ToLower(value)
	lowerFile := strings.ToLower(entry.SourceFile)

	if isFlutterStorageKey(lowerKey, lowerText) {
		return false, SkipStorageKeys
	}
	if isFlutterRoute(value) {
		return false, SkipRoutes
	}
	if isFlutterAsset(value, lowerFile) {
		return false, SkipAssets
	}
	if isFlutterProviderName(value, lowerKey) {
		return false, SkipProviders
	}
	if isFlutterTechnicalKey(lowerKey) {
		return false, SkipTechnical
	}
	if isFlutterTechnicalText(value) {
		return false, SkipTechnical
	}

	return true, ""
}

func isFlutterStorageKey(key, text string) bool {
	patterns := []string{
		"_box", "_id", "_key", "anima_save_v2", "current_anima_v3",
		"shared_preferences", "hive", "provider_api_keys", "token", "jwt",
	}
	for _, p := range patterns {
		if strings.Contains(key, p) || strings.Contains(text, p) {
			return true
		}
	}
	if strings.HasPrefix(key, "anima.const.string.") ||
		strings.HasPrefix(key, "common.const.string.") {
		return true
	}
	return false
}

func isFlutterRoute(text string) bool {
	trimmed := strings.TrimSpace(text)
	return strings.HasPrefix(trimmed, "/") && !strings.Contains(trimmed, " ")
}

func isFlutterAsset(text, sourceFile string) bool {
	lower := strings.ToLower(text)
	if strings.HasPrefix(lower, "assets/") {
		return true
	}
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".webp", ".svg", ".mp3", ".wav"} {
		if strings.Contains(lower, ext) {
			return true
		}
	}
	return strings.Contains(sourceFile, "/assets/")
}

func isFlutterProviderName(text, key string) bool {
	lower := strings.ToLower(text + " " + key)
	for _, name := range flutterProviderNames {
		if strings.Contains(lower, name) {
			// Allow user-facing sentences that merely mention a provider in prose.
			if strings.Contains(lower, "tu es") || strings.Contains(lower, "réponds") {
				return false
			}
			if len(text) > 40 && strings.Contains(text, " ") {
				return false
			}
			return true
		}
	}
	return false
}

func isFlutterTechnicalKey(key string) bool {
	if strings.Contains(key, ".const.") {
		return true
	}
	if strings.HasSuffix(key, "_box") || strings.HasSuffix(key, "_id") {
		return true
	}
	return false
}

func isFlutterTechnicalText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}
	if isFlutterRoute(trimmed) || isFlutterAsset(trimmed, "") {
		return true
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return true
	}
	// camelCase identifier without spaces (non-displayed enum/storage key).
	if !strings.Contains(trimmed, " ") &&
		!strings.Contains(trimmed, "'") &&
		RegExpMatch(`^[a-z][a-zA-Z0-9_]*$`, trimmed) &&
		!strings.Contains(trimmed, "é") &&
		!strings.Contains(trimmed, "è") &&
		!strings.Contains(trimmed, "à") {
		return true
	}
	return false
}

// RegExpMatch is a tiny helper to avoid importing regexp in hot paths for tests.
func RegExpMatch(pattern, value string) bool {
	switch pattern {
	case `^[a-z][a-zA-Z0-9_]*$`:
		if len(value) == 0 {
			return false
		}
		if value[0] < 'a' || value[0] > 'z' {
			return false
		}
		for i := 1; i < len(value); i++ {
			ch := value[i]
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
				continue
			}
			return false
		}
		return true
	default:
		return false
	}
}

func classifySkippedEntry(entry NormalizedFlutterScanEntry) FlutterScanSkipReason {
	ok, reason := isTranslatableFlutterEntry(entry)
	if ok {
		return ""
	}
	return reason
}

func basenameNoExt(path string) string {
	base := filepath.Base(path)
	if dot := strings.LastIndex(base, "."); dot >= 0 {
		return base[:dot]
	}
	return base
}