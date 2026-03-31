package zyn

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// MockFixedProviderName is the name for the fixed mock provider.
const MockFixedProviderName = "mock-fixed"

// MockProvider simulates LLM behavior for testing.
// It returns deterministic responses based on prompt patterns.
type MockProvider struct {
	name      string
	available bool
}

// NewMockProvider creates a new mock provider for testing.
func NewMockProvider() Provider {
	return &MockProvider{
		name:      "mock",
		available: true,
	}
}

// NewMockProviderWithName creates a new mock provider with a specific name.
func NewMockProviderWithName(name string) *MockProvider {
	return &MockProvider{
		name:      name,
		available: true,
	}
}

// Call simulates an LLM call with deterministic responses.
// For testing, it uses the last message content as the prompt.
func (m *MockProvider) Call(_ context.Context, messages []Message, _ float32) (*ProviderResponse, error) {
	if !m.available {
		return nil, fmt.Errorf("provider %s is unavailable", m.name)
	}

	// Extract last message (the new user message) for generating response
	var prompt string
	if len(messages) > 0 {
		prompt = messages[len(messages)-1].Content
	}

	// Generate response based on prompt patterns
	return &ProviderResponse{
		Content: m.generateResponse(prompt),
		Usage: TokenUsage{
			Prompt:     100, // Mock token counts
			Completion: 50,
			Total:      150,
		},
	}, nil
}

// Name returns the provider identifier.
func (m *MockProvider) Name() string {
	return m.name
}

// SetAvailable sets the availability status (for testing failures).
func (m *MockProvider) SetAvailable(available bool) {
	m.available = available
}

// generateResponse creates a response based on prompt patterns.
func (m *MockProvider) generateResponse(prompt string) string {
	// Check for JSON response request
	if strings.Contains(prompt, "Response JSON Schema:") {
		// Classification pattern
		if strings.Contains(prompt, "Categories:") {
			return m.generateClassificationResponse(prompt)
		}

		// Ranking pattern
		if strings.Contains(prompt, "Items:") {
			return m.generateRankingResponse(prompt)
		}

		// Sentiment pattern
		if strings.Contains(prompt, "sentiment") || strings.Contains(prompt, "Sentiment") {
			return m.generateSentimentResponse(prompt)
		}

		// Transform pattern
		if strings.Contains(prompt, "transform") || strings.Contains(prompt, "Transform") {
			return `{"output": "transformed text", "confidence": 0.9, "changes": ["change1"], "reasoning": ["mock"]}`
		}

		// Analyze pattern
		if strings.Contains(prompt, "analyze") || strings.Contains(prompt, "Analyze") {
			return `{"analysis": "mock analysis", "confidence": 0.9, "findings": ["finding1"], "reasoning": ["mock"]}`
		}

		// Binary decision pattern
		if strings.Contains(prompt, "valid email") || strings.Contains(prompt, "email") {
			return m.generateEmailValidationResponse(prompt)
		}

		// Default binary response
		return `{"decision": true, "confidence": 0.8, "reasoning": ["Mock response generated"]}`
	}

	// Default response
	return "Mock response"
}

// generateClassificationResponse creates classification responses.
func (*MockProvider) generateClassificationResponse(prompt string) string {
	// Extract first category as primary
	categories := extractCategories(prompt)
	primary := "unknown"
	if len(categories) > 0 {
		primary = categories[0]
	}

	response := struct {
		Primary    string   `json:"primary"`
		Secondary  string   `json:"secondary"`
		Confidence float64  `json:"confidence"`
		Reasoning  []string `json:"reasoning"`
	}{
		Primary:    primary,
		Secondary:  "",
		Confidence: 0.85,
		Reasoning:  []string{"Mock classification"},
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return `{"primary": "unknown", "secondary": "", "confidence": 0.85, "reasoning": ["Mock classification"]}`
	}
	return string(jsonBytes)
}

// generateRankingResponse creates ranking responses.
func (*MockProvider) generateRankingResponse(prompt string) string {
	items := extractItems(prompt)

	response := struct {
		Ranked     []string `json:"ranked"`
		Confidence float64  `json:"confidence"`
		Reasoning  []string `json:"reasoning"`
	}{
		Ranked:     items,
		Confidence: 0.85,
		Reasoning:  []string{"Mock ranking"},
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return `{"ranked": [], "confidence": 0.85, "reasoning": ["Mock ranking"]}`
	}
	return string(jsonBytes)
}

// generateSentimentResponse creates sentiment responses.
func (*MockProvider) generateSentimentResponse(_ string) string {
	response := struct {
		Overall    string  `json:"overall"`
		Confidence float64 `json:"confidence"`
		Scores     struct {
			Positive float64 `json:"positive"`
			Negative float64 `json:"negative"`
			Neutral  float64 `json:"neutral"`
		} `json:"scores"`
		Aspects   map[string]string `json:"aspects"`
		Emotions  []string          `json:"emotions"`
		Reasoning []string          `json:"reasoning"`
	}{
		Overall:    "positive",
		Confidence: 0.85,
		Scores: struct {
			Positive float64 `json:"positive"`
			Negative float64 `json:"negative"`
			Neutral  float64 `json:"neutral"`
		}{Positive: 0.7, Negative: 0.1, Neutral: 0.2},
		Aspects:   map[string]string{},
		Emotions:  []string{"joy"},
		Reasoning: []string{"Mock sentiment"},
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return `{"overall": "positive", "confidence": 0.85, "scores": {"positive": 0.7, "negative": 0.1, "neutral": 0.2}, "aspects": {}, "emotions": ["joy"], "reasoning": ["Mock sentiment"]}`
	}
	return string(jsonBytes)
}

// extractCategories extracts categories from prompt.
func extractCategories(prompt string) []string {
	var categories []string
	inCategories := false

	for _, line := range strings.Split(prompt, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "Categories:" {
			inCategories = true
			continue
		}
		if inCategories {
			if strings.HasPrefix(trimmed, "1. ") || strings.HasPrefix(trimmed, "2. ") || strings.HasPrefix(trimmed, "3. ") {
				// Extract category name after number
				parts := strings.SplitN(trimmed, ". ", 2)
				if len(parts) == 2 {
					categories = append(categories, parts[1])
				}
			} else if trimmed != "" && !strings.Contains(trimmed, ":") {
				break
			}
		}
	}

	return categories
}

// extractItems extracts items from prompt.
func extractItems(prompt string) []string {
	var items []string
	inItems := false

	for _, line := range strings.Split(prompt, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "Items:" {
			inItems = true
			continue
		}
		if inItems {
			if strings.HasPrefix(trimmed, "1. ") || strings.HasPrefix(trimmed, "2. ") || strings.HasPrefix(trimmed, "3. ") {
				// Extract item after number
				parts := strings.SplitN(trimmed, ". ", 2)
				if len(parts) == 2 {
					items = append(items, parts[1])
				}
			} else if trimmed != "" && !strings.Contains(trimmed, ":") {
				break
			}
		}
	}

	return items
}

// generateEmailValidationResponse creates email validation responses.
func (*MockProvider) generateEmailValidationResponse(prompt string) string {
	// Extract the subject from prompt
	subject := extractSubject(prompt)

	// Validate email format
	isValid := strings.Contains(subject, "@") &&
		strings.Contains(subject, ".") &&
		!strings.HasPrefix(subject, "@")

	response := struct {
		Decision   bool     `json:"decision"`
		Confidence float64  `json:"confidence"`
		Reasoning  []string `json:"reasoning"`
	}{
		Decision:   isValid,
		Confidence: 0.85,
		Reasoning: []string{
			"Checked for @ symbol",
			"Verified domain extension",
			"Validated format structure",
		},
	}

	if !isValid {
		response.Reasoning = append(response.Reasoning, "Invalid email format detected")
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return "Mock response"
	}
	return string(jsonBytes)
}

// extractSubject extracts the subject from a prompt.
func extractSubject(prompt string) string {
	// Look for "Input: " pattern
	if idx := strings.Index(prompt, "Input: "); idx != -1 {
		start := idx + 7
		end := strings.Index(prompt[start:], "\n")
		if end == -1 {
			return strings.TrimSpace(prompt[start:])
		}
		return strings.TrimSpace(prompt[start : start+end])
	}

	return ""
}

// NewMockProviderWithResponse creates a mock that always returns a specific response.
func NewMockProviderWithResponse(response string) Provider {
	return &mockProviderFixed{response: response}
}

// NewMockProviderWithCallback creates a mock that calls a function to generate responses.
func NewMockProviderWithCallback(callback func(prompt string, temperature float32) (string, error)) Provider {
	return &mockProviderCallback{callback: callback}
}

// NewMockProviderWithError creates a mock that always returns an error.
func NewMockProviderWithError(errMsg string) Provider {
	return &mockProviderError{errMsg: errMsg}
}

// mockProviderFixed always returns a fixed response.
type mockProviderFixed struct {
	response string
}

func (m *mockProviderFixed) Call(_ context.Context, _ []Message, _ float32) (*ProviderResponse, error) {
	return &ProviderResponse{
		Content: m.response,
		Usage: TokenUsage{
			Prompt:     100,
			Completion: 50,
			Total:      150,
		},
	}, nil
}

func (*mockProviderFixed) Name() string {
	return MockFixedProviderName
}

// mockProviderCallback uses a callback to generate responses.
type mockProviderCallback struct {
	callback func(string, float32) (string, error)
}

func (m *mockProviderCallback) Call(_ context.Context, messages []Message, temperature float32) (*ProviderResponse, error) {
	// Extract last message for callback
	var prompt string
	if len(messages) > 0 {
		prompt = messages[len(messages)-1].Content
	}
	content, err := m.callback(prompt, temperature)
	if err != nil {
		return nil, err
	}
	return &ProviderResponse{
		Content: content,
		Usage: TokenUsage{
			Prompt:     100,
			Completion: 50,
			Total:      150,
		},
	}, nil
}

func (*mockProviderCallback) Name() string {
	return "mock-callback"
}

// mockProviderError always returns an error.
type mockProviderError struct {
	errMsg string
}

func (m *mockProviderError) Call(_ context.Context, _ []Message, _ float32) (*ProviderResponse, error) {
	return nil, fmt.Errorf("%s", m.errMsg)
}

func (*mockProviderError) Name() string {
	return "mock-error"
}

// MockStreamingProvider simulates streaming by splitting the response into chunks.
// It implements both Provider (via embedding) and StreamingProvider.
type MockStreamingProvider struct {
	MockProvider
	chunkSize int // Number of characters per chunk. 0 means deliver entire response as one chunk.
}

// NewMockStreamingProvider creates a mock that simulates streaming responses.
// chunkSize controls how the response is split into chunks for the callback.
func NewMockStreamingProvider(chunkSize int) *MockStreamingProvider {
	return &MockStreamingProvider{
		MockProvider: MockProvider{
			name:      "mock-streaming",
			available: true,
		},
		chunkSize: chunkSize,
	}
}

// Stream simulates streaming by splitting the response into chunks and calling the callback.
// After all chunks are delivered, returns the full ProviderResponse with usage.
func (m *MockStreamingProvider) Stream(ctx context.Context, messages []Message, temperature float32, callback StreamCallback) (*ProviderResponse, error) {
	resp, err := m.Call(ctx, messages, temperature)
	if err != nil {
		return nil, err
	}

	// Deliver chunks via callback
	content := resp.Content
	if m.chunkSize > 0 && callback != nil {
		for i := 0; i < len(content); i += m.chunkSize {
			end := i + m.chunkSize
			if end > len(content) {
				end = len(content)
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				callback(content[i:end])
			}
		}
	} else if callback != nil {
		callback(content)
	}

	return resp, nil
}

// MockToolProvider simulates tool use by returning tool calls based on the tools provided.
// It implements Provider (via embedding) and ToolProvider.
type MockToolProvider struct {
	MockProvider
	toolResponse func(messages []Message, tools []Tool) *ProviderResponse
}

// NewMockToolProvider creates a mock that simulates tool use.
// By default it returns a tool call for the first tool with a mock input.
func NewMockToolProvider() *MockToolProvider {
	return &MockToolProvider{
		MockProvider: MockProvider{
			name:      "mock-tool",
			available: true,
		},
	}
}

// WithToolResponse sets a custom function for generating tool responses.
func (m *MockToolProvider) WithToolResponse(fn func(messages []Message, tools []Tool) *ProviderResponse) *MockToolProvider {
	m.toolResponse = fn
	return m
}

// CallWithTools simulates a tool-using LLM call.
// If a custom toolResponse function is set, it is used.
// Otherwise, returns a tool call for the first tool with mock input.
func (m *MockToolProvider) CallWithTools(_ context.Context, messages []Message, _ float32, tools []Tool) (*ProviderResponse, error) {
	if !m.available {
		return nil, fmt.Errorf("provider %s is unavailable", m.name)
	}

	if m.toolResponse != nil {
		return m.toolResponse(messages, tools), nil
	}

	// Default: call the first tool with empty input
	if len(tools) == 0 {
		return &ProviderResponse{
			Content:    "No tools provided",
			StopReason: StopReasonEndTurn,
			Usage:      TokenUsage{Prompt: 100, Completion: 50, Total: 150},
		}, nil
	}

	return &ProviderResponse{
		Content:    "",
		StopReason: StopReasonToolUse,
		ToolCalls: []ToolCall{
			{
				ID:    "call_mock_001",
				Name:  tools[0].Name,
				Input: json.RawMessage(`{}`),
			},
		},
		Usage: TokenUsage{Prompt: 100, Completion: 50, Total: 150},
	}, nil
}

// MockToolStreamingProvider simulates streaming with tool use by emitting
// StreamEvents for both text and tool calls. It implements Provider (via embedding),
// ToolProvider (via MockToolProvider), and ToolStreamingProvider.
type MockToolStreamingProvider struct {
	MockToolProvider
}

// NewMockToolStreamingProvider creates a mock that simulates streaming with tools.
func NewMockToolStreamingProvider() *MockToolStreamingProvider {
	return &MockToolStreamingProvider{
		MockToolProvider: MockToolProvider{
			MockProvider: MockProvider{
				name:      "mock-tool-streaming",
				available: true,
			},
		},
	}
}

// StreamWithTools simulates streaming a tool-using response by emitting events
// for each piece of the response, then returning the full assembled result.
func (m *MockToolStreamingProvider) StreamWithTools(ctx context.Context, messages []Message, temperature float32, tools []Tool, callback StreamEventCallback) (*ProviderResponse, error) {
	// Get the full response using the non-streaming tool path
	resp, err := m.CallWithTools(ctx, messages, temperature, tools)
	if err != nil {
		return nil, err
	}

	if callback == nil {
		return resp, nil
	}

	// Emit text content as a text event
	if resp.Content != "" {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			callback(StreamEvent{Type: StreamEventText, Text: resp.Content})
		}
	}

	// Emit tool calls as start/end events
	for i, tc := range resp.ToolCalls {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			tcCopy := tc
			callback(StreamEvent{
				Type:     StreamEventToolStart,
				ToolCall: &tcCopy,
				Index:    i,
			})
			if len(tc.Input) > 0 {
				callback(StreamEvent{
					Type:      StreamEventToolDelta,
					ToolDelta: string(tc.Input),
					Index:     i,
				})
			}
			callback(StreamEvent{
				Type:  StreamEventToolEnd,
				Index: i,
			})
		}
	}

	return resp, nil
}
