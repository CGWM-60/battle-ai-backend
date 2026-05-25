package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type Provider struct {
	name          string
	apiKey        string
	url           string
	model         string
	usageRecorder UsageRecorder
	usageMetadata UsageMetadata
}

type ProviderMessage struct {
	Role     string                  `json:"role"`
	Content  string                  `json:"-"`
	Content2 []SpecMessageTextOpenAI `json:"-"`
}

type ProviderChatRequest struct {
	Stream   bool              `json:"stream"`
	Messages []ProviderMessage `json:"messages"`
	Model    string            `json:"model"`
}

type UsageMetadata struct {
	Mode      string
	Operation string
	Phase     string
	Round     int
	ActorName string
}

type UsageRecord struct {
	ProviderHost     string
	Model            string
	Mode             string
	Operation        string
	Phase            string
	Round            int
	ActorName        string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	InputChars       int
	OutputChars      int
	Stream           bool
	Fallback         bool
	Estimated        bool
}

type UsageRecorder func(UsageRecord)

type ProviderStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content any `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

type ProviderCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content any    `json:"content"`
			Refusal string `json:"refusal"`
		} `json:"message"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Error *ProviderError `json:"error,omitempty"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

type ProviderErrorResponse struct {
	Error *ProviderError `json:"error,omitempty"`
}

type ProviderError struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
	Code    any    `json:"code,omitempty"`
}

type providerCallResult struct {
	Content          string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	Estimated        bool
}
type SpecMessageTextOpenAI struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	ImageUrl string `json:"image_url,omitempty"`
}

func NewsProvider(apiKey string, url string, model string) *Provider {
	return &Provider{
		apiKey: apiKey,
		url:    url,
		model:  model,
	}
}

func (p *Provider) WithUsageRecorder(recorder UsageRecorder) *Provider {
	p.usageRecorder = recorder
	return p
}

func (p *Provider) WithUsageMetadata(metadata UsageMetadata) *Provider {
	p.usageMetadata = metadata
	return p
}
func (m ProviderMessage) MarshalJSON() ([]byte, error) {
	type MessageString struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type MessageArray struct {
		Role    string                  `json:"role"`
		Content []SpecMessageTextOpenAI `json:"content"`
	}

	if len(m.Content2) > 0 {
		return json.Marshal(MessageArray{
			Role:    m.Role,
			Content: m.Content2,
		})
	}

	return json.Marshal(MessageString{
		Role:    m.Role,
		Content: m.Content,
	})
}

func (p *Provider) Chat(ctx context.Context, messages []ProviderMessage) (string, error) {
	result, err := p.chatCompletion(ctx, messages, false, nil)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(result.Content) == "" {
		return "", fmt.Errorf("provider returned empty non-stream response: model=%s host=%s", p.model, p.host())
	}
	p.recordUsage(messages, result, false, false)
	return result.Content, nil
}

func (p *Provider) ChatStream(
	ctx context.Context,
	messages []ProviderMessage,
	onChunk func(chunk string),
) (string, error) {
	result, err := p.chatCompletion(ctx, messages, true, onChunk)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(result.Content) != "" {
		p.recordUsage(messages, result, true, false)
		return result.Content, nil
	}

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	log.Printf("[provider] empty stream response host=%s model=%s retry=non_stream", p.host(), p.model)
	fallback, err := p.chatCompletion(ctx, messages, false, nil)
	if err != nil {
		return "", fmt.Errorf("empty stream then non-stream fallback failed: %w", err)
	}
	if strings.TrimSpace(fallback.Content) == "" {
		return "", fmt.Errorf("provider returned empty response after stream and non-stream fallback: model=%s host=%s", p.model, p.host())
	}
	if onChunk != nil {
		onChunk(fallback.Content)
	}
	p.recordUsage(messages, fallback, false, true)
	return fallback.Content, nil
}

func (p *Provider) chatCompletion(
	ctx context.Context,
	messages []ProviderMessage,
	stream bool,
	onChunk func(chunk string),
) (providerCallResult, error) {
	request := ProviderChatRequest{
		Stream:   stream,
		Messages: messages,
		Model:    p.model,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return providerCallResult{}, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.url,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return providerCallResult{}, err
	}

	apiKey := cleanProviderAPIKey(p.apiKey)

	req.Header.Set("Content-Type", "application/json")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if strings.Contains(p.url, "openrouter.ai") {
		req.Header.Set("X-Title", "go-battle-ia")
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return providerCallResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return providerCallResult{}, fmt.Errorf("provider error: status=%s body=%s", resp.Status, string(body))
	}

	if !stream {
		return readCompletionResponse(resp.Body)
	}

	return readStreamResponse(resp.Body, p.host(), p.model, onChunk)
}

func cleanProviderAPIKey(apiKey string) string {
	cleaned := strings.TrimSpace(apiKey)
	cleaned = strings.Trim(cleaned, `"'`)
	if strings.HasPrefix(strings.ToLower(cleaned), "bearer ") {
		cleaned = strings.TrimSpace(cleaned[len("bearer "):])
	}
	return strings.Trim(cleaned, `"'`)
}

func readCompletionResponse(body io.Reader) (providerCallResult, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return providerCallResult{}, err
	}

	var completion ProviderCompletionResponse
	if err := json.Unmarshal(bodyBytes, &completion); err != nil {
		return providerCallResult{}, fmt.Errorf("parse provider completion JSON: %w body=%s", err, preview(bodyBytes, 600))
	}
	if completion.Error != nil {
		return providerCallResult{}, fmt.Errorf("provider error: type=%s code=%v message=%s", completion.Error.Type, completion.Error.Code, completion.Error.Message)
	}

	var fullResponse strings.Builder
	for _, choice := range completion.Choices {
		content := contentToString(choice.Message.Content)
		if content != "" {
			fullResponse.WriteString(content)
		}
	}
	result := providerCallResult{Content: fullResponse.String(), Estimated: true}
	if completion.Usage != nil {
		result.PromptTokens = completion.Usage.PromptTokens
		result.CompletionTokens = completion.Usage.CompletionTokens
		result.TotalTokens = completion.Usage.TotalTokens
		result.Estimated = false
	}
	return result, nil
}

func readStreamResponse(body io.Reader, host string, model string, onChunk func(chunk string)) (providerCallResult, error) {
	var fullResponse strings.Builder
	chunkCount := 0
	emptyChoiceCount := 0
	lastData := ""

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		lastData = data

		if data == "[DONE]" {
			break
		}

		var providerErr ProviderErrorResponse
		if err := json.Unmarshal([]byte(data), &providerErr); err == nil && providerErr.Error != nil {
			return providerCallResult{}, fmt.Errorf("provider stream error: type=%s code=%v message=%s", providerErr.Error.Type, providerErr.Error.Code, providerErr.Error.Message)
		}

		var streamResp ProviderStreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			return providerCallResult{}, fmt.Errorf("parse provider stream JSON: %w data=%s", err, preview([]byte(data), 600))
		}

		for _, choice := range streamResp.Choices {
			content := contentToString(choice.Delta.Content)
			if content != "" {
				chunkCount++
				fullResponse.WriteString(content)
				if onChunk != nil {
					onChunk(content)
				}
			} else {
				emptyChoiceCount++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return providerCallResult{}, err
	}
	if fullResponse.Len() == 0 {
		log.Printf(
			"[provider] stream completed without content host=%s model=%s chunks=%d empty_choices=%d last_data=%s",
			host,
			model,
			chunkCount,
			emptyChoiceCount,
			preview([]byte(lastData), 600),
		)
	}

	return providerCallResult{Content: fullResponse.String(), Estimated: true}, nil
}

func (p *Provider) recordUsage(messages []ProviderMessage, result providerCallResult, stream bool, fallback bool) {
	if p.usageRecorder == nil {
		return
	}
	inputChars := CountMessageChars(messages)
	outputChars := len([]rune(result.Content))
	promptTokens := result.PromptTokens
	completionTokens := result.CompletionTokens
	totalTokens := result.TotalTokens
	estimated := result.Estimated
	if promptTokens <= 0 {
		promptTokens = EstimateTokensForMessages(messages)
		estimated = true
	}
	if completionTokens <= 0 {
		completionTokens = EstimateTokensForText(result.Content)
		estimated = true
	}
	if totalTokens <= 0 {
		totalTokens = promptTokens + completionTokens
	}
	metadata := p.usageMetadata
	p.usageRecorder(UsageRecord{
		ProviderHost:     p.host(),
		Model:            p.model,
		Mode:             metadata.Mode,
		Operation:        metadata.Operation,
		Phase:            metadata.Phase,
		Round:            metadata.Round,
		ActorName:        metadata.ActorName,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		InputChars:       inputChars,
		OutputChars:      outputChars,
		Stream:           stream,
		Fallback:         fallback,
		Estimated:        estimated,
	})
}

func CountMessageChars(messages []ProviderMessage) int {
	total := 0
	for _, message := range messages {
		total += len([]rune(message.Role))
		total += len([]rune(message.Content))
		for _, block := range message.Content2 {
			total += len([]rune(block.Type))
			total += len([]rune(block.Text))
			total += len([]rune(block.ImageUrl))
		}
	}
	return total
}

func EstimateTokensForMessages(messages []ProviderMessage) int {
	total := 0
	for _, message := range messages {
		total += 4
		total += EstimateTokensForText(message.Role)
		total += EstimateTokensForText(message.Content)
		for _, block := range message.Content2 {
			total += EstimateTokensForText(block.Type)
			total += EstimateTokensForText(block.Text)
			total += EstimateTokensForText(block.ImageUrl)
		}
	}
	if total < 1 {
		return 1
	}
	return total
}

func EstimateTokensForText(value string) int {
	runes := len([]rune(value))
	if runes <= 0 {
		return 0
	}
	tokens := (runes + 3) / 4
	if tokens < 1 {
		return 1
	}
	return tokens
}

func contentToString(value any) string {
	switch content := value.(type) {
	case string:
		return content
	case []any:
		var out strings.Builder
		for _, item := range content {
			part, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := part["text"].(string); ok {
				out.WriteString(text)
			}
		}
		return out.String()
	default:
		return ""
	}
}

func (p *Provider) host() string {
	parsed, err := url.Parse(p.url)
	if err != nil || parsed.Host == "" {
		return "unknown"
	}
	return parsed.Host
}

func preview(value []byte, maxLength int) string {
	clean := strings.ReplaceAll(string(value), "\n", " ")
	clean = strings.ReplaceAll(clean, "\r", " ")
	if len(clean) <= maxLength {
		return clean
	}
	return clean[:maxLength] + "...(truncated)"
}
