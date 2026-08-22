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

// defaultNumPredict / defaultKeepAlive are the package's fallback values, used
// whenever a Client is built via New() and its NumPredict/KeepAlive fields are
// left at their zero value.
//
//   - defaultNumPredict caps how many tokens a single reply may generate. The
//     persona prompt already asks for 1-3 sentences, but nothing previously
//     stopped the model from ignoring that and rambling - and on a CPU-only 3B
//     model, generation time scales directly with output length, so an
//     unbounded reply is an unbounded wait. 220 tokens is comfortably more than
//     a normal reply needs, so real answers won't get cut short, but a
//     worst-case runaway is now capped instead of open-ended.
//
//   - defaultKeepAlive tells Ollama how long to keep the model resident in
//     memory after a reply. Ollama's own default is 5 minutes; a normal George
//     session can easily have gaps longer than that between turns while you're
//     reading or typing, and reloading a model from disk is a big chunk of the
//     "5-6 minutes" wait when it happens mid-conversation instead of only once.
const (
	defaultNumPredict = 220
	defaultKeepAlive  = "30m"

	// defaultTopP narrows sampling slightly below Ollama's own default (0.9).
	// The long tail of low-probability tokens is exactly where hallucinated
	// slang words tend to come from on a model that's less fluent in casual
	// Jakarta Indonesian than in formal registers - trimming that tail costs
	// almost nothing in naturalness but cuts down on nonsense word chains.
	defaultTopP = 0.85

	// defaultRepeatPenalty is a mild bump over Ollama's default (1.1). The
	// gibberish in George's closing replies reads like repetition-driven
	// degeneration (e.g. "Kelar kali ya? Jalan-jalannnya."); a slightly
	// stronger penalty discourages the model from looping into that kind of
	// filler without being aggressive enough to force unnatural phrasing.
	defaultRepeatPenalty = 1.15
)

// Client talks to a local Ollama instance.
type Client struct {
	BaseURL     string
	Model       string
	Temperature float64
	NumCtx      int
	HTTP        *http.Client

	// NumPredict overrides defaultNumPredict when non-zero. Left unset (0) by
	// New(), so existing callers get the default automatically.
	NumPredict int

	// KeepAlive overrides defaultKeepAlive when non-empty, using Ollama's
	// duration string format (e.g. "10m", "1h"). Left unset ("") by New(), so
	// existing callers get the default automatically.
	KeepAlive string

	// TopP overrides defaultTopP when non-zero.
	TopP float64

	// RepeatPenalty overrides defaultRepeatPenalty when non-zero.
	RepeatPenalty float64
}

// ChatOverrides carries per-call adjustments to Chat, layered on top of the
// Client's own defaults. The zero value changes nothing - passing no override at
// all, or a bare ChatOverrides{}, behaves exactly like before this type existed.
type ChatOverrides struct {
	// NumPredict overrides both the Client's NumPredict and defaultNumPredict for
	// this call only, when non-zero - e.g. router.WantsDetail matched, so this one
	// reply is allowed to run longer than George's usual short answers.
	NumPredict int
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
	Temperature   float64 `json:"temperature"`
	NumCtx        int     `json:"num_ctx"`
	NumPredict    int     `json:"num_predict"`
	TopP          float64 `json:"top_p"`
	RepeatPenalty float64 `json:"repeat_penalty"`
}

type chatRequest struct {
	Model     string      `json:"model"`
	Messages  []Message   `json:"messages"`
	Stream    bool        `json:"stream"`
	KeepAlive string      `json:"keep_alive"`
	Options   chatOptions `json:"options"`
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
func (c *Client) Chat(messages []Message, overrides ...ChatOverrides) (string, error) {
	numPredict := c.NumPredict
	if numPredict == 0 {
		numPredict = defaultNumPredict
	}
	if len(overrides) > 0 && overrides[0].NumPredict != 0 {
		numPredict = overrides[0].NumPredict
	}
	keepAlive := c.KeepAlive
	if keepAlive == "" {
		keepAlive = defaultKeepAlive
	}
	topP := c.TopP
	if topP == 0 {
		topP = defaultTopP
	}
	repeatPenalty := c.RepeatPenalty
	if repeatPenalty == 0 {
		repeatPenalty = defaultRepeatPenalty
	}

	reqBody := chatRequest{
		Model:     c.Model,
		Messages:  messages,
		Stream:    false,
		KeepAlive: keepAlive,
		Options: chatOptions{
			Temperature:   c.Temperature,
			NumCtx:        c.NumCtx,
			NumPredict:    numPredict,
			TopP:          topP,
			RepeatPenalty: repeatPenalty,
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
