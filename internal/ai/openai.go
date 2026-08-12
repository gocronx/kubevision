package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Message is an OpenAI chat-completions message. It doubles as both the request
// history element and the accumulated assistant reply.
type Message struct {
	Role       string     `json:"role"` // system | user | assistant | tool
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall is a single function invocation requested by the model.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // always "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall holds the name and raw JSON arguments of a tool call.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Tool is an OpenAI tool (function) definition advertised to the model.
type Tool struct {
	Type     string       `json:"type"` // always "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction describes a callable function and its JSON-schema parameters.
type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// Client is a minimal OpenAI-compatible chat-completions client. It speaks the
// /chat/completions streaming protocol, which is implemented by OpenAI,
// OpenRouter, DeepSeek, Together, Qwen/DashScope (compatible mode) and most
// other providers — so a single implementation covers them all.
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
}

// Model is the minimal model metadata used by the settings model picker.
type Model struct {
	ID string `json:"id"`
}

type modelsResponse struct {
	Data []Model `json:"data"`
}

// NewClient builds a client from the given config.
func NewClient(cfg Config) *Client {
	cfg = cfg.withDefaults()
	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		maxTokens:  cfg.MaxTokens,
		httpClient: &http.Client{Timeout: 5 * time.Minute},
	}
}

// ListModels returns model IDs exposed by an OpenAI-compatible /models endpoint.
func (c *Client) ListModels(ctx context.Context) ([]Model, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list LLM models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("model provider returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result modelsResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode model list: %w", err)
	}
	return result.Data, nil
}

type chatRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	Tools     []Tool    `json:"tools,omitempty"`
	Stream    bool      `json:"stream"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

// streamChunk is one delta frame from the streaming response.
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// Stream sends one chat-completions turn. Text deltas are delivered to onContent
// as they arrive; the fully accumulated assistant message (text plus any
// tool calls) is returned when the stream completes.
func (c *Client) Stream(
	ctx context.Context,
	messages []Message,
	tools []Tool,
	onContent func(string),
) (Message, error) {
	reqBody := chatRequest{
		Model:     c.model,
		Messages:  messages,
		Tools:     tools,
		Stream:    true,
		MaxTokens: c.maxTokens,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return Message{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("call LLM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return Message{}, fmt.Errorf("LLM returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return parseStream(resp.Body, onContent)
}

// parseStream consumes an SSE chat-completions stream, invoking onContent for
// each text delta and accumulating tool calls by index.
func parseStream(body io.Reader, onContent func(string)) (Message, error) {
	msg := Message{Role: "assistant"}
	// Tool calls arrive fragmented across chunks; accumulate by index.
	type acc struct {
		id, name string
		args     strings.Builder
	}
	calls := map[int]*acc{}
	var order []int

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Skip malformed keep-alive or comment frames.
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta

		if delta.Content != "" {
			msg.Content += delta.Content
			if onContent != nil {
				onContent(delta.Content)
			}
		}

		for _, tc := range delta.ToolCalls {
			a, ok := calls[tc.Index]
			if !ok {
				a = &acc{}
				calls[tc.Index] = a
				order = append(order, tc.Index)
			}
			if tc.ID != "" {
				a.id = tc.ID
			}
			if tc.Function.Name != "" {
				a.name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				a.args.WriteString(tc.Function.Arguments)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Message{}, fmt.Errorf("read LLM stream: %w", err)
	}

	for _, idx := range order {
		a := calls[idx]
		if a.name == "" {
			continue
		}
		msg.ToolCalls = append(msg.ToolCalls, ToolCall{
			ID:       a.id,
			Type:     "function",
			Function: FunctionCall{Name: a.name, Arguments: a.args.String()},
		})
	}

	return msg, nil
}
