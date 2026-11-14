package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LLM talks to any OpenAI compatible chat completions endpoint: DeepSeek,
// Gemini, Groq, OpenRouter, or OpenAI.
type LLM struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Complete sends one prompt and returns the assistant reply text.
func (l *LLM) Complete(ctx context.Context, system, user string) (string, error) {
	if l.BaseURL == "" {
		return "", fmt.Errorf("LLM_BASE_URL not set")
	}
	client := l.HTTP
	if client == nil {
		client = &http.Client{Timeout: 90 * time.Second}
	}
	url := strings.TrimSuffix(l.BaseURL, "/")
	if !strings.HasSuffix(url, "/chat/completions") {
		url += "/chat/completions"
	}
	body, err := json.Marshal(chatRequest{
		Model: l.Model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if l.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+l.APIKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}
	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("llm: bad response %s", resp.Status)
	}
	if resp.StatusCode != http.StatusOK {
		if out.Error != nil && out.Error.Message != "" {
			return "", fmt.Errorf("llm: %s", out.Error.Message)
		}
		return "", fmt.Errorf("llm: %s", resp.Status)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("llm: empty reply")
	}
	return out.Choices[0].Message.Content, nil
}
