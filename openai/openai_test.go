package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/zoobz-io/zyn"
)

func TestProviderCall(t *testing.T) {
	ctx := context.Background()
	// Create a test server that mimics OpenAI API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Bearer token, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var req chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Model != "gpt-3.5-turbo" {
			t.Errorf("Expected model gpt-3.5-turbo, got %s", req.Model)
		}
		if req.Temperature != 0.7 {
			t.Errorf("Expected temperature 0.7, got %f", req.Temperature)
		}
		if len(req.Messages) != 1 || req.Messages[0].Content != "test prompt" {
			t.Errorf("Unexpected prompt: %v", req.Messages)
		}

		// Send response
		resp := chatCompletionResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-3.5-turbo",
			Choices: []choice{
				{
					Index: 0,
					Message: message{
						Role:    zyn.RoleAssistant,
						Content: "test response",
					},
					FinishReason: "stop",
				},
			},
			Usage: usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with test server URL
	provider := New(Config{
		APIKey:  "test-key",
		Model:   "gpt-3.5-turbo",
		BaseURL: server.URL,
	})

	// Make a call
	response, err := provider.Call(ctx, []zyn.Message{{Role: zyn.RoleUser, Content: "test prompt"}}, 0.7)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if response.Content != "test response" {
		t.Errorf("Expected 'test response', got '%s'", response.Content)
	}
}

func TestOpenAIIntegration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	ctx := context.Background()
	provider := New(Config{
		APIKey: apiKey,
		Model:  "gpt-3.5-turbo",
	})

	response, err := provider.Call(ctx, []zyn.Message{{Role: zyn.RoleUser, Content: "Say 'test successful' and nothing else."}}, 0.7)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if response.Content == "" {
		t.Error("Expected non-empty response")
	}

	t.Logf("Response: %s", response.Content)
}

func TestProviderErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{
			name:       "Rate limit error",
			statusCode: http.StatusTooManyRequests,
			responseBody: `{
				"error": {
					"message": "Rate limit exceeded",
					"type": "rate_limit_error",
					"code": "rate_limit"
				}
			}`,
			expectedError: "rate limit exceeded",
		},
		{
			name:       "API error",
			statusCode: http.StatusBadRequest,
			responseBody: `{
				"error": {
					"message": "Invalid request",
					"type": "invalid_request_error"
				}
			}`,
			expectedError: "openai error (400): Invalid request",
		},
		{
			name:          "Generic error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `not json`,
			expectedError: "openai error: status 500",
		},
		{
			name:          "Empty response",
			statusCode:    http.StatusOK,
			responseBody:  `{"choices": []}`,
			expectedError: "no response choices returned",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			provider := New(Config{
				APIKey:  "test-key",
				BaseURL: server.URL,
			})

			_, err := provider.Call(ctx, []zyn.Message{{Role: zyn.RoleUser, Content: "test"}}, 0.7)
			if err == nil {
				t.Fatal("Expected error but got none")
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

func TestProviderCallWithTools(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		ctx := context.Background()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req chatCompletionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Failed to decode request: %v", err)
			}

			// Verify tools are in the request
			if len(req.Tools) != 1 {
				t.Fatalf("Expected 1 tool, got %d", len(req.Tools))
			}
			if req.Tools[0].Function.Name != "get_weather" {
				t.Errorf("Expected tool name 'get_weather', got '%s'", req.Tools[0].Function.Name)
			}
			if req.Tools[0].Type != "function" {
				t.Errorf("Expected tool type 'function', got '%s'", req.Tools[0].Type)
			}

			// No response_format when tools are present
			if req.ResponseFormat != nil {
				t.Error("Expected no response_format when tools are present")
			}

			// Return a tool call response
			resp := chatCompletionResponse{
				ID:    "test-id",
				Model: "gpt-4",
				Choices: []choice{{
					Index: 0,
					Message: message{
						Role: zyn.RoleAssistant,
						ToolCalls: []toolCall{{
							ID:   "call_abc123",
							Type: "function",
							Function: toolCallFunction{
								Name:      "get_weather",
								Arguments: `{"location": "San Francisco"}`,
							},
						}},
					},
					FinishReason: "tool_calls",
				}},
				Usage: usage{PromptTokens: 50, CompletionTokens: 25, TotalTokens: 75},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		provider := New(Config{APIKey: "test-key", BaseURL: server.URL})
		tools := []zyn.Tool{{
			Name:        "get_weather",
			Description: "Get the weather for a location",
			Parameters:  json.RawMessage(`{"type": "object", "properties": {"location": {"type": "string"}}}`),
		}}

		response, err := provider.CallWithTools(ctx, []zyn.Message{{Role: zyn.RoleUser, Content: "What's the weather in SF?"}}, 0.5, tools)
		if err != nil {
			t.Fatalf("CallWithTools failed: %v", err)
		}
		if response.StopReason != zyn.StopReasonToolUse {
			t.Errorf("Expected StopReason='tool_use', got '%s'", response.StopReason)
		}
		if len(response.ToolCalls) != 1 {
			t.Fatalf("Expected 1 tool call, got %d", len(response.ToolCalls))
		}
		if response.ToolCalls[0].ID != "call_abc123" {
			t.Errorf("Expected tool call ID 'call_abc123', got '%s'", response.ToolCalls[0].ID)
		}
		if response.ToolCalls[0].Name != "get_weather" {
			t.Errorf("Expected tool call name 'get_weather', got '%s'", response.ToolCalls[0].Name)
		}
		if string(response.ToolCalls[0].Input) != `{"location": "San Francisco"}` {
			t.Errorf("Unexpected tool call input: %s", response.ToolCalls[0].Input)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Tool result messages are sent correctly
		ctx := context.Background()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req chatCompletionRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Failed to decode request: %v", err)
			}

			// Find the tool result message
			var toolMsg *message
			for i := range req.Messages {
				if req.Messages[i].Role == zyn.RoleTool {
					toolMsg = &req.Messages[i]
					break
				}
			}
			if toolMsg == nil {
				t.Fatal("Expected a tool result message")
			}
			if toolMsg.ToolCallID != "call_abc123" {
				t.Errorf("Expected tool_call_id 'call_abc123', got '%s'", toolMsg.ToolCallID)
			}

			resp := chatCompletionResponse{
				ID:    "test-id",
				Model: "gpt-4",
				Choices: []choice{{
					Index: 0,
					Message: message{
						Role:    zyn.RoleAssistant,
						Content: "The weather in SF is 72°F.",
					},
					FinishReason: "stop",
				}},
				Usage: usage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		provider := New(Config{APIKey: "test-key", BaseURL: server.URL})
		messages := []zyn.Message{
			{Role: zyn.RoleUser, Content: "What's the weather?"},
			{Role: zyn.RoleAssistant, ToolCalls: []zyn.ToolCall{{ID: "call_abc123", Name: "get_weather", Input: json.RawMessage(`{"location":"SF"}`)}}},
			{Role: zyn.RoleTool, Content: `{"temp": 72}`, ToolCallID: "call_abc123"},
		}
		tools := []zyn.Tool{{Name: "get_weather"}}

		response, err := provider.CallWithTools(ctx, messages, 0.5, tools)
		if err != nil {
			t.Fatalf("CallWithTools failed: %v", err)
		}
		if response.StopReason != zyn.StopReasonEndTurn {
			t.Errorf("Expected StopReason='end_turn', got '%s'", response.StopReason)
		}
		if response.Content != "The weather in SF is 72°F." {
			t.Errorf("Unexpected content: %s", response.Content)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// Verify ToolProvider interface is satisfied
		provider := New(Config{APIKey: "test-key"})
		var _ zyn.ToolProvider = provider
	})
}

func TestMapFinishReason(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"stop", zyn.StopReasonEndTurn},
		{"tool_calls", zyn.StopReasonToolUse},
		{"length", zyn.StopReasonMaxTokens},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapFinishReason(tt.input)
			if got != tt.expected {
				t.Errorf("mapFinishReason(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestProviderName(t *testing.T) {
	provider := New(Config{
		APIKey: "test-key",
		Model:  "gpt-4",
	})

	name := provider.Name()
	if name != "openai" {
		t.Errorf("Expected 'openai', got '%s'", name)
	}
}
