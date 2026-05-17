package aigateway

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Streamer interface {
	StreamChat(ctx context.Context, system string, msgs []Message, onToken func(string) error) (string, error)
	StreamChatWithTools(ctx context.Context, system string, msgs []Message, tools []Tool, onToken func(string) error) (string, []ToolCall, error)
	Chat(ctx context.Context, system string, msgs []Message, tools []Tool) (string, []ToolCall, error)
}

type Client struct {
	APIKey  string
	Model   string
	BaseURL string
	HTTP    *http.Client
}

func New(apiKey, model, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://ai-gateway.vercel.sh/v1"
	}
	return &Client{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Tools    []Tool    `json:"tools,omitempty"`
}

type sseChunk struct {
	Choices []struct {
		Delta struct {
			Content   string             `json:"content"`
			ToolCalls []sseToolCallDelta `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
}

type sseToolCallDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	} `json:"function"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Role      string     `json:"role"`
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *Client) buildMessages(system string, msgs []Message) []Message {
	all := make([]Message, 0, len(msgs)+1)
	if system != "" {
		all = append(all, Message{Role: "system", Content: system})
	}
	return append(all, msgs...)
}

func (c *Client) Chat(ctx context.Context, system string, msgs []Message, tools []Tool) (string, []ToolCall, error) {
	if c.APIKey == "" {
		return "", nil, errors.New("aigateway: missing API key")
	}
	body, err := json.Marshal(chatRequest{
		Model:    c.Model,
		Messages: c.buildMessages(system, msgs),
		Tools:    tools,
	})
	if err != nil {
		return "", nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", nil, fmt.Errorf("aigateway: status %d: %s", resp.StatusCode, string(b))
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", nil, err
	}
	if len(out.Choices) == 0 {
		return "", nil, nil
	}
	m := out.Choices[0].Message
	return m.Content, m.ToolCalls, nil
}

func (c *Client) StreamChat(ctx context.Context, system string, msgs []Message, onToken func(string) error) (string, error) {
	if c.APIKey == "" {
		return "", errors.New("aigateway: missing API key")
	}
	body, err := json.Marshal(chatRequest{
		Model:    c.Model,
		Messages: c.buildMessages(system, msgs),
		Stream:   true,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("aigateway: status %d: %s", resp.StatusCode, string(b))
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var full strings.Builder
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			if payload == "[DONE]" {
				break
			}
			continue
		}
		var chunk sseChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		for _, ch := range chunk.Choices {
			tok := ch.Delta.Content
			if tok == "" {
				continue
			}
			full.WriteString(tok)
			if onToken != nil {
				if err := onToken(tok); err != nil {
					return full.String(), err
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		return full.String(), err
	}
	return full.String(), nil
}

func (c *Client) StreamChatWithTools(ctx context.Context, system string, msgs []Message, tools []Tool, onToken func(string) error) (string, []ToolCall, error) {
	if c.APIKey == "" {
		return "", nil, errors.New("aigateway: missing API key")
	}
	body, err := json.Marshal(chatRequest{
		Model:    c.Model,
		Messages: c.buildMessages(system, msgs),
		Stream:   true,
		Tools:    tools,
	})
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", nil, fmt.Errorf("aigateway: status %d: %s", resp.StatusCode, string(b))
	}

	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var full strings.Builder
	toolAccum := map[int]*ToolCall{}
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			break
		}
		var chunk sseChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		for _, ch := range chunk.Choices {
			if tok := ch.Delta.Content; tok != "" {
				full.WriteString(tok)
				if onToken != nil {
					if err := onToken(tok); err != nil {
						return full.String(), nil, err
					}
				}
			}
			for _, tcd := range ch.Delta.ToolCalls {
				tc, ok := toolAccum[tcd.Index]
				if !ok {
					tc = &ToolCall{Type: "function"}
					toolAccum[tcd.Index] = tc
				}
				if tcd.ID != "" {
					tc.ID = tcd.ID
				}
				if tcd.Type != "" {
					tc.Type = tcd.Type
				}
				if tcd.Function.Name != "" {
					tc.Function.Name = tcd.Function.Name
				}
				if tcd.Function.Arguments != "" {
					tc.Function.Arguments += tcd.Function.Arguments
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		return full.String(), nil, err
	}

	tcs := make([]ToolCall, 0, len(toolAccum))
	for i := 0; i < len(toolAccum); i++ {
		if tc, ok := toolAccum[i]; ok {
			tcs = append(tcs, *tc)
		}
	}
	return full.String(), tcs, nil
}
