// Package ollama is a minimal client for the local Ollama REST API
// (http://localhost:11434), used so George never needs a paid third-party AI service.
package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client talks to a local Ollama instance.
type Client struct {
	BaseURL     string
	Model       string
	Temperature float64
	NumCtx      int
	HTTP        *http.Client
}

// New creates a client pointed at baseURL (e.g. "http://localhost:11434") using the
// given model tag. temperature controls response randomness (lower = more focused/
// context-grounded, higher = more creative but prone to rambling off-topic); numCtx
// sets the context window in tokens - how much of the conversation George can see at once.
func New(baseURL, model string, temperature float64, numCtx int) *Client {
	return &Client{
		BaseURL:     baseURL,
		Model:       model,
		Temperature: temperature,
		NumCtx:      numCtx,
		HTTP:        &http.Client{Timeout: 60 * time.Second},
	}
}

// Message is one turn in a conversation, following Ollama's /api/chat role convention.
type Message struct {
	Role    string `json:"role"` // "system", "user", or "assistant"
	Content string `json:"content"`
}

type chatOptions struct {
	Temperature float64 `json:"temperature"`
	NumCtx      int     `json:"num_ctx"`
}

type chatRequest struct {
	Model    string      `json:"model"`
	Messages []Message   `json:"messages"`
	Stream   bool        `json:"stream"`
	Options  chatOptions `json:"options"`
}

type chatResponse struct {
	Message Message `json:"message"`
	Done    bool    `json:"done"`
}

// Chat sends the full conversation (system persona + everything said so far + the
// newest user message) to Ollama's /api/chat endpoint. This matters for two reasons:
// it applies qwen2.5's own chat template correctly (better quality than hand-rolling
// prompt text), and it's what actually gives George memory within a session - without
// history, every reply is generated blind, with zero idea what was just talked about.
func (c *Client) Chat(messages []Message) (string, error) {
	reqBody := chatRequest{
		Model:    c.Model,
		Messages: messages,
		Stream:   false,
		Options: chatOptions{
			Temperature: c.Temperature,
			NumCtx:      c.NumCtx,
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.HTTP.Post(c.BaseURL+"/api/chat", "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("call ollama (is it running? try: ollama serve): %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var out chatResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	return out.Message.Content, nil
}
