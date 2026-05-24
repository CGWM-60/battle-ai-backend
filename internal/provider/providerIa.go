package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Provider struct {
	name   string
	apiKey string
	url    string
	model  string
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

type ProviderStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
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
	return p.ChatStream(ctx, messages, nil)
}

func (p *Provider) ChatStream(
	ctx context.Context,
	messages []ProviderMessage,
	onChunk func(chunk string),
) (string, error) {
	request := ProviderChatRequest{
		Stream:   true,
		Messages: messages,
		Model:    p.model,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.url,
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", err
	}

	apiKey := p.apiKey
	if apiKey == "" {
		apiKey = p.apiKey
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("mistral error: status=%s body=%s", resp.Status, string(body))
	}

	var fullResponse strings.Builder

	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			break
		}

		var streamResp ProviderStreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			return "", err
		}

		for _, choice := range streamResp.Choices {
			content := choice.Delta.Content
			if content != "" {
				fmt.Print(content) // affichage en live dans le terminal
				fullResponse.WriteString(content)
				if onChunk != nil {
					onChunk(content)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return fullResponse.String(), nil
}
