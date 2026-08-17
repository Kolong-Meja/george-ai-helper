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
	BaseURL string
	Model   string
	HTTP    *http.Client
}

// New creates a client pointed at baseURL (e.g. "http://localhost:11434") using the given model tag.
func New(baseURL, model string) *Client {
	return &Client{
		BaseURL: baseURL,
		Model:   model,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	System string `json:"system,omitempty"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Generate sends prompt + system persona to Ollama and returns the model's reply.
// Stream is disabled for simplicity; swap to streaming later if you want George to
// print tokens as they arrive instead of waiting for the full response.
func (c *Client) Generate(prompt, system string) (string, error) {
	reqBody := generateRequest{
		Model:  c.Model,
		Prompt: prompt,
		System: system,
		Stream: false,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.HTTP.Post(c.BaseURL+"/api/generate", "application/json", bytes.NewReader(data))
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

	var out generateResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	return out.Response, nil
}