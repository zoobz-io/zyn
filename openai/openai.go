package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zoobz-io/capitan"
	"github.com/zoobz-io/zyn"
)

// Provider implements the zyn Provider interface for OpenAI API.
type Provider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
	name       string
}

// Config holds configuration for the OpenAI provider.
type Config struct {
	APIKey  string
	Model   string        // e.g. "gpt-4", "gpt-3.5-turbo"
	BaseURL string        // Optional, defaults to "https://api.openai.com/v1"
	Timeout time.Duration // Optional, defaults to 30s
}

// New creates a new OpenAI provider.
func New(config Config) *Provider {
	if config.Model == "" {
		config.Model = "gpt-3.5-turbo"
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Provider{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: config.BaseURL,
		name:    "openai",
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return p.name
}

// Call sends messages to OpenAI and returns the response with usage stats.
// OpenAI automatically handles prompt caching for prompts >1024 tokens.
func (p *Provider) Call(ctx context.Context, messages []zyn.Message, temperature float32) (*zyn.ProviderResponse, error) {
	requestBody := chatCompletionRequest{
		Model:       p.model,
		Messages:    convertMessages(messages),
		Temperature: temperature,
		ResponseFormat: &responseFormat{
			Type: "json_object",
		},
	}
	return p.do(ctx, requestBody)
}

// CallWithTools sends messages with tool definitions to OpenAI.
// The response may contain tool calls instead of (or alongside) text content.
// When tool calls are present, the caller should execute the tools and send
// results back as RoleTool messages in a subsequent call.
func (p *Provider) CallWithTools(ctx context.Context, messages []zyn.Message, temperature float32, tools []zyn.Tool) (*zyn.ProviderResponse, error) {
	apiTools := make([]tool, len(tools))
	for i, t := range tools {
		apiTools[i] = tool{
			Type: "function",
			Function: toolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		}
	}

	requestBody := chatCompletionRequest{
		Model:       p.model,
		Messages:    convertMessages(messages),
		Temperature: temperature,
		Tools:       apiTools,
	}
	return p.do(ctx, requestBody)
}

// convertMessages converts zyn messages to OpenAI wire format.
func convertMessages(messages []zyn.Message) []message {
	apiMessages := make([]message, len(messages))
	for i, msg := range messages {
		m := message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}
		// Convert tool calls from zyn format to OpenAI format
		if len(msg.ToolCalls) > 0 {
			m.ToolCalls = make([]toolCall, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				m.ToolCalls[j] = toolCall{
					ID:   tc.ID,
					Type: "function",
					Function: toolCallFunction{
						Name:      tc.Name,
						Arguments: string(tc.Input),
					},
				}
			}
		}
		apiMessages[i] = m
	}
	return apiMessages
}

// do executes the HTTP request and parses the response.
func (p *Provider) do(ctx context.Context, requestBody chatCompletionRequest) (*zyn.ProviderResponse, error) {
	startTime := time.Now()

	// Emit provider.call.started hook
	capitan.Info(ctx, zyn.ProviderCallStarted,
		zyn.ProviderKey.Field(p.name),
		zyn.ModelKey.Field(p.model),
	)

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Make the request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle errors
	if resp.StatusCode != http.StatusOK {
		return nil, p.handleError(ctx, body, resp.StatusCode, startTime)
	}

	// Parse successful response
	var completionResp chatCompletionResponse
	if err := json.Unmarshal(body, &completionResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(completionResp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned")
	}

	// Calculate duration
	duration := time.Since(startTime)

	// Emit provider.call.completed hook with token usage and metadata
	fields := []capitan.Field{
		zyn.ProviderKey.Field(p.name),
		zyn.ModelKey.Field(completionResp.Model),
		zyn.PromptTokensKey.Field(completionResp.Usage.PromptTokens),
		zyn.CompletionTokensKey.Field(completionResp.Usage.CompletionTokens),
		zyn.TotalTokensKey.Field(completionResp.Usage.TotalTokens),
		zyn.DurationMsKey.Field(int(duration.Milliseconds())),
		zyn.HTTPStatusCodeKey.Field(resp.StatusCode),
		zyn.ResponseIDKey.Field(completionResp.ID),
		zyn.ResponseCreatedKey.Field(int(completionResp.Created)),
	}

	if completionResp.Choices[0].FinishReason != "" {
		fields = append(fields, zyn.ResponseFinishReasonKey.Field(completionResp.Choices[0].FinishReason))
	}

	capitan.Info(ctx, zyn.ProviderCallCompleted, fields...)

	// Build response with tool calls if present
	choice := completionResp.Choices[0]
	providerResp := &zyn.ProviderResponse{
		Content:    choice.Message.Content,
		StopReason: mapFinishReason(choice.FinishReason),
		Usage: zyn.TokenUsage{
			Prompt:     completionResp.Usage.PromptTokens,
			Completion: completionResp.Usage.CompletionTokens,
			Total:      completionResp.Usage.TotalTokens,
		},
	}

	// Map tool calls from OpenAI format to zyn format
	if len(choice.Message.ToolCalls) > 0 {
		providerResp.ToolCalls = make([]zyn.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			providerResp.ToolCalls[i] = zyn.ToolCall{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: json.RawMessage(tc.Function.Arguments),
			}
		}
	}

	return providerResp, nil
}

// handleError processes error responses from OpenAI.
func (p *Provider) handleError(ctx context.Context, body []byte, statusCode int, startTime time.Time) error {
	duration := time.Since(startTime)
	var errorResp errorResponse

	fields := []capitan.Field{
		zyn.ProviderKey.Field(p.name),
		zyn.ModelKey.Field(p.model),
		zyn.HTTPStatusCodeKey.Field(statusCode),
		zyn.DurationMsKey.Field(int(duration.Milliseconds())),
	}

	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
		fields = append(fields,
			zyn.ErrorKey.Field(errorResp.Error.Message),
			zyn.APIErrorTypeKey.Field(errorResp.Error.Type),
		)
		if errorResp.Error.Code != "" {
			fields = append(fields, zyn.APIErrorCodeKey.Field(errorResp.Error.Code))
		}

		capitan.Error(ctx, zyn.ProviderCallFailed, fields...)

		if statusCode == http.StatusTooManyRequests {
			return fmt.Errorf("rate limit exceeded: %s", errorResp.Error.Message)
		}
		return fmt.Errorf("openai error (%d): %s", statusCode, errorResp.Error.Message)
	}

	fields = append(fields, zyn.ErrorKey.Field(fmt.Sprintf("status %d", statusCode)))
	capitan.Error(ctx, zyn.ProviderCallFailed, fields...)
	return fmt.Errorf("openai error: status %d", statusCode)
}

// mapFinishReason converts OpenAI finish reasons to zyn StopReason constants.
func mapFinishReason(reason string) string {
	switch reason {
	case "stop":
		return zyn.StopReasonEndTurn
	case "tool_calls":
		return zyn.StopReasonToolUse
	case "length":
		return zyn.StopReasonMaxTokens
	default:
		return reason
	}
}

// Request/Response types for OpenAI API

type responseFormat struct {
	Type string `json:"type"`
}

type chatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       []message       `json:"messages"`
	Temperature    float32         `json:"temperature"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	Tools          []tool          `json:"tools,omitempty"`
}

type message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type tool struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type toolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function toolCallFunction `json:"function"`
}

type toolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type choice struct {
	Index        int     `json:"index"`
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}
