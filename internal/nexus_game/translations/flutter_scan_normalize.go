package translations

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FlutterScanPayload wraps entries from extracted_texts.json.
type FlutterScanPayload struct {
	GeneratedAt string             `json:"generatedAt"`
	Source      string             `json:"source"`
	Entries     []FlutterScanEntry `json:"entries"`
}

// ParamBinding maps a template placeholder to a Dart expression.
type ParamBinding struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

// FlutterScanEntry accepts both legacy backend and current Flutter extractor shapes.
type FlutterScanEntry struct {
	ID string `json:"id"`

	// Backend-oriented fields
	Text               string   `json:"text"`
	SourceFile         string   `json:"sourceFile"`
	SourceLine         int      `json:"sourceLine"`
	Usage              string   `json:"usage"`
	SuggestedNamespace string   `json:"suggestedNamespace"`
	SuggestedKey       string   `json:"suggestedKey"`

	// Flutter extractor fields
	FullKey      string         `json:"fullKey"`
	Domain       string         `json:"domain"`
	File         string         `json:"file"`
	Line         int            `json:"line"`
	OriginalText string         `json:"originalText"`
	Context      string         `json:"context"`
	Kind         string         `json:"kind"`
	Module       string         `json:"module"`
	Tags         []string       `json:"tags"`
	NeedsParams  bool           `json:"needsParams"`
	Params       []ParamBinding `json:"params"`
	DefaultText  string         `json:"defaultText"`
}

// NormalizedFlutterScanEntry is the canonical import shape.
type NormalizedFlutterScanEntry struct {
	ID                 string
	FullKey            string
	Domain             string
	Namespace          string
	Text               string
	DefaultText        string
	SourceFile         string
	SourceLine         int
	Usage              string
	Kind               string
	Module             string
	Tags               []string
	SuggestedNamespace string
}

func (e FlutterScanEntry) Normalized() NormalizedFlutterScanEntry {
	fullKey := strings.TrimSpace(e.SuggestedKey)
	if fullKey == "" {
		fullKey = strings.TrimSpace(e.FullKey)
	}

	sourceFile := strings.TrimSpace(e.SourceFile)
	if sourceFile == "" {
		sourceFile = strings.TrimSpace(e.File)
	}

	sourceLine := e.SourceLine
	if sourceLine == 0 {
		sourceLine = e.Line
	}

	text := strings.TrimSpace(e.Text)
	if text == "" {
		text = strings.TrimSpace(e.OriginalText)
	}
	if text == "" {
		text = strings.TrimSpace(e.DefaultText)
	}

	defaultText := strings.TrimSpace(e.DefaultText)
	if defaultText == "" {
		defaultText = text
	}

	namespace := strings.TrimSpace(e.SuggestedNamespace)
	if namespace == "" && fullKey != "" {
		parts := strings.Split(fullKey, ".")
		if len(parts) > 1 {
			namespace = strings.Join(parts[:len(parts)-1], ".")
		}
	}

	domain := strings.TrimSpace(e.Domain)
	if domain == "" && fullKey != "" {
		domain = strings.Split(fullKey, ".")[0]
	}
	if domain == "" && namespace != "" {
		domain = strings.Split(namespace, ".")[0]
	}

	module := strings.TrimSpace(e.Module)
	if module == "" {
		module = domain
	}

	kind := strings.TrimSpace(e.Kind)
	if kind == "" {
		kind = strings.TrimSpace(e.Usage)
	}

	usage := strings.TrimSpace(e.Usage)
	if usage == "" {
		usage = kind
	}

	tags := normalizeFlutterScanTags(e.Tags, module, kind, e.Context)

	return NormalizedFlutterScanEntry{
		ID:                 strings.TrimSpace(e.ID),
		FullKey:            fullKey,
		Domain:             domain,
		Namespace:          namespace,
		Text:               text,
		DefaultText:        defaultText,
		SourceFile:         sourceFile,
		SourceLine:         sourceLine,
		Usage:              usage,
		Kind:               kind,
		Module:             module,
		Tags:               tags,
		SuggestedNamespace: namespace,
	}
}

func normalizeFlutterScanTags(tags []string, module, kind, context string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(tags)+3)
	add := func(v string) {
		v = strings.TrimSpace(strings.ToLower(v))
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	for _, tag := range tags {
		add(tag)
	}
	add(module)
	add(kind)
	if context != "" {
		add(strings.ReplaceAll(context, ".", "_"))
	}
	return out
}

// ParseFlutterScanBytes accepts a raw array, {entries:[]}, or full extracted_texts.json.
func ParseFlutterScanBytes(raw []byte) ([]FlutterScanEntry, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, fmt.Errorf("empty flutter scan payload")
	}

	var entries []FlutterScanEntry
	if err := json.Unmarshal(raw, &entries); err == nil && len(entries) > 0 {
		return entries, nil
	}

	var payload FlutterScanPayload
	if err := json.Unmarshal(raw, &payload); err == nil && len(payload.Entries) > 0 {
		return payload.Entries, nil
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if nested, ok := envelope["entries"]; ok {
		if err := json.Unmarshal(nested, &entries); err != nil {
			return nil, err
		}
		if len(entries) == 0 {
			return nil, fmt.Errorf("flutter scan payload contains no entries")
		}
		return entries, nil
	}

	return nil, fmt.Errorf("flutter scan payload contains no entries")
}