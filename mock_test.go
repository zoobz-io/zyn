package zyn

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewMockProvider(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()

		if provider == nil {
			t.Fatal("NewMockProvider returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()

		ctx := context.Background()
		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: "test prompt"}}, 0.5)
		if err != nil {
			t.Errorf("Call failed: %v", err)
		}
		if response.Content == "" {
			t.Error("Expected non-empty response")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()

		name := provider.Name()
		if name == "" {
			t.Error("Provider name should not be empty")
		}
	})
}

func TestNewMockProviderWithName(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithName("test-provider")

		if provider == nil {
			t.Fatal("NewMockProviderWithName returned nil")
		}
		if provider.Name() != "test-provider" {
			t.Errorf("Expected name 'test-provider', got '%s'", provider.Name())
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithName("reliable-provider")

		ctx := context.Background()
		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: "test"}}, 0.5)
		if err != nil {
			t.Errorf("Call failed: %v", err)
		}
		if response.Content == "" {
			t.Error("Expected response from named provider")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithName("provider1")
		provider2 := NewMockProviderWithName("provider2")

		if provider.Name() == provider2.Name() {
			t.Error("Different providers should have different names")
		}
	})
}

func TestMockProvider_Call(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()

		ctx := context.Background()
		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: "test prompt"}}, 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}
		if response.Content == "" {
			t.Error("Expected non-empty response")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithName("test")

		ctx := context.Background()
		response1, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: "prompt1"}}, 0.5)
		if err != nil {
			t.Errorf("First call failed: %v", err)
		}

		response2, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: "prompt2"}}, 0.5)
		if err != nil {
			t.Errorf("Second call failed: %v", err)
		}

		if response1.Content == "" || response2.Content == "" {
			t.Error("Expected responses from both calls")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()

		ctx := context.Background()
		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: "test"}}, 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}

		// Response should be parseable as various types
		if response.Content == "" {
			t.Error("Expected valid response for chaining")
		}
	})
}

func TestMockProvider_Name(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()

		name := provider.Name()
		if name == "" {
			t.Error("Name returned empty string")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithName("custom-name")

		name := provider.Name()
		if name != "custom-name" {
			t.Errorf("Expected 'custom-name', got '%s'", name)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithName("test")

		name1 := provider.Name()
		name2 := provider.Name()
		if name1 != name2 {
			t.Error("Name should be consistent")
		}
	})
}

func TestMockProvider_SetAvailable(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithName("test")

		provider.SetAvailable(false)

		ctx := context.Background()
		_, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: "test"}}, 0.5)
		if err == nil {
			t.Error("Expected error when unavailable")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithName("test")

		ctx := context.Background()

		// Initially available
		_, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: "test"}}, 0.5)
		if err != nil {
			t.Errorf("Provider should be available initially: %v", err)
		}

		// Set unavailable
		provider.SetAvailable(false)
		_, err = provider.Call(ctx, []Message{{Role: RoleUser, Content: "test"}}, 0.5)
		if err == nil {
			t.Error("Expected error when unavailable")
		}
		if !strings.Contains(err.Error(), "unavailable") {
			t.Errorf("Expected 'unavailable' in error, got: %v", err)
		}

		// Set available again
		provider.SetAvailable(true)
		_, err = provider.Call(ctx, []Message{{Role: RoleUser, Content: "test"}}, 0.5)
		if err != nil {
			t.Errorf("Provider should be available again: %v", err)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithName("test")
		ctx := context.Background()

		provider.SetAvailable(false)
		_, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: "test"}}, 0.5)
		if err == nil {
			t.Error("Expected unavailable error")
		}

		provider.SetAvailable(true)
		_, err = provider.Call(ctx, []Message{{Role: RoleUser, Content: "test"}}, 0.5)
		if err != nil {
			t.Error("Should be available after re-enabling")
		}
	})
}

func TestNewMockProviderWithResponse(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"test": "value"}`)

		if provider == nil {
			t.Fatal("NewMockProviderWithResponse returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		expectedResponse := `{"decision": true, "confidence": 0.9, "reasoning": ["test"]}`
		provider := NewMockProviderWithResponse(expectedResponse)

		ctx := context.Background()
		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: "any prompt"}}, 0.5)
		if err != nil {
			t.Errorf("Call failed: %v", err)
		}
		if response.Content != expectedResponse {
			t.Errorf("Expected fixed response '%s', got '%s'", expectedResponse, response.Content)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"test": "fixed"}`)

		ctx := context.Background()
		response1, _ := provider.Call(ctx, []Message{{Role: RoleUser, Content: "prompt1"}}, 0.5)
		response2, _ := provider.Call(ctx, []Message{{Role: RoleUser, Content: "prompt2"}}, 0.5)

		if response1.Content != response2.Content {
			t.Error("Fixed response provider should return same response")
		}
	})
}

func TestNewMockProviderWithCallback(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithCallback(func(_ string, _ float32) (string, error) {
			return "callback response", nil
		})

		if provider == nil {
			t.Fatal("NewMockProviderWithCallback returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		callCount := 0
		provider := NewMockProviderWithCallback(func(prompt string, _ float32) (string, error) {
			callCount++
			return "response " + prompt, nil
		})

		ctx := context.Background()
		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: "test"}}, 0.5)
		if err != nil {
			t.Errorf("Call failed: %v", err)
		}
		if response.Content != "response test" {
			t.Errorf("Expected 'response test', got '%s'", response.Content)
		}
		if callCount != 1 {
			t.Errorf("Expected callback to be called once, got %d", callCount)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithCallback(func(prompt string, _ float32) (string, error) {
			if strings.Contains(prompt, "error") {
				return "", nil
			}
			return `{"result": "` + prompt + `"}`, nil
		})

		ctx := context.Background()
		response1, _ := provider.Call(ctx, []Message{{Role: RoleUser, Content: "prompt1"}}, 0.5)
		response2, _ := provider.Call(ctx, []Message{{Role: RoleUser, Content: "prompt2"}}, 0.5)

		if response1.Content == response2.Content {
			t.Error("Callback should produce different responses for different prompts")
		}
	})
}

func TestNewMockProviderWithError(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithError("test error")

		if provider == nil {
			t.Fatal("NewMockProviderWithError returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		expectedError := "simulated failure"
		provider := NewMockProviderWithError(expectedError)

		ctx := context.Background()
		_, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: "test"}}, 0.5)
		if err == nil {
			t.Error("Expected error but got none")
		}
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error containing '%s', got '%v'", expectedError, err)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithError("persistent error")

		ctx := context.Background()
		_, err1 := provider.Call(ctx, []Message{{Role: RoleUser, Content: "test1"}}, 0.5)
		_, err2 := provider.Call(ctx, []Message{{Role: RoleUser, Content: "test2"}}, 0.5)

		if err1 == nil || err2 == nil {
			t.Error("Error provider should always return error")
		}
	})
}

func TestMockProvider_GenerateRankingResponse(t *testing.T) {
	t.Run("with_items", func(t *testing.T) {
		provider := NewMockProvider()
		ctx := context.Background()

		// Prompt that triggers ranking response path
		prompt := `Response JSON Schema:
{"type": "object"}

Items:
1. apple
2. banana
3. cherry`

		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: prompt}}, 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}

		if !strings.Contains(response.Content, "ranked") {
			t.Errorf("Expected ranking response with 'ranked', got: %s", response.Content)
		}
		if !strings.Contains(response.Content, "apple") {
			t.Errorf("Expected response to contain extracted item 'apple', got: %s", response.Content)
		}
	})

	t.Run("empty_items", func(t *testing.T) {
		provider := NewMockProvider()
		ctx := context.Background()

		prompt := `Response JSON Schema:
{"type": "object"}

Items:`

		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: prompt}}, 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}

		if !strings.Contains(response.Content, "ranked") {
			t.Errorf("Expected ranking response, got: %s", response.Content)
		}
	})
}

func TestMockProvider_GenerateSentimentResponse(t *testing.T) {
	t.Run("sentiment_keyword", func(t *testing.T) {
		provider := NewMockProvider()
		ctx := context.Background()

		prompt := `Response JSON Schema:
{"type": "object"}

Analyze the sentiment of this text.`

		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: prompt}}, 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}

		if !strings.Contains(response.Content, "overall") {
			t.Errorf("Expected sentiment response with 'overall', got: %s", response.Content)
		}
		if !strings.Contains(response.Content, "positive") {
			t.Errorf("Expected sentiment response with sentiment value, got: %s", response.Content)
		}
	})

	t.Run("Sentiment_capitalized", func(t *testing.T) {
		provider := NewMockProvider()
		ctx := context.Background()

		prompt := `Response JSON Schema:
{"type": "object"}

Sentiment analysis required.`

		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: prompt}}, 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}

		if !strings.Contains(response.Content, "overall") {
			t.Errorf("Expected sentiment response, got: %s", response.Content)
		}
	})
}

func TestMockProvider_GenerateEmailValidationResponse(t *testing.T) {
	t.Run("valid_email", func(t *testing.T) {
		provider := NewMockProvider()
		ctx := context.Background()

		prompt := `Response JSON Schema:
{"type": "object"}

Is this a valid email address?

Input: user@example.com`

		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: prompt}}, 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}

		if !strings.Contains(response.Content, `"decision":true`) && !strings.Contains(response.Content, `"decision": true`) {
			t.Errorf("Expected valid email to return true decision, got: %s", response.Content)
		}
	})

	t.Run("invalid_email", func(t *testing.T) {
		provider := NewMockProvider()
		ctx := context.Background()

		prompt := `Response JSON Schema:
{"type": "object"}

Is this a valid email address?

Input: not-an-email`

		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: prompt}}, 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}

		if !strings.Contains(response.Content, `"decision":false`) && !strings.Contains(response.Content, `"decision": false`) {
			t.Errorf("Expected invalid email to return false decision, got: %s", response.Content)
		}
	})

	t.Run("email_at_start", func(t *testing.T) {
		provider := NewMockProvider()
		ctx := context.Background()

		prompt := `Response JSON Schema:
{"type": "object"}

Is this a valid email?

Input: @invalid.com`

		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: prompt}}, 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}

		if !strings.Contains(response.Content, `"decision":false`) && !strings.Contains(response.Content, `"decision": false`) {
			t.Errorf("Expected email starting with @ to be invalid, got: %s", response.Content)
		}
	})

	t.Run("no_input_line", func(t *testing.T) {
		provider := NewMockProvider()
		ctx := context.Background()

		prompt := `Response JSON Schema:
{"type": "object"}

Check this email: test@test.com`

		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: prompt}}, 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}

		// Should still return a response (extractSubject returns empty string)
		if !strings.Contains(response.Content, "decision") {
			t.Errorf("Expected decision in response, got: %s", response.Content)
		}
	})
}

func TestMockProvider_ExtractSubject(t *testing.T) {
	t.Run("input_with_newline", func(t *testing.T) {
		provider := NewMockProvider()
		ctx := context.Background()

		prompt := `Response JSON Schema:
{"type": "object"}

valid email check

Input: test@domain.org
Some other text`

		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: prompt}}, 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}

		// Should extract just "test@domain.org" not including next line
		if !strings.Contains(response.Content, `"decision":true`) && !strings.Contains(response.Content, `"decision": true`) {
			t.Errorf("Expected valid email decision, got: %s", response.Content)
		}
	})

	t.Run("input_at_end", func(t *testing.T) {
		provider := NewMockProvider()
		ctx := context.Background()

		prompt := `Response JSON Schema:
{"type": "object"}

valid email

Input: final@test.com`

		response, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: prompt}}, 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}

		if !strings.Contains(response.Content, `"decision":true`) && !strings.Contains(response.Content, `"decision": true`) {
			t.Errorf("Expected valid email at end of prompt, got: %s", response.Content)
		}
	})
}

func TestNewMockStreamingProvider(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockStreamingProvider(5)
		if provider == nil {
			t.Fatal("NewMockStreamingProvider returned nil")
		}
		if provider.Name() != "mock-streaming" {
			t.Errorf("Expected name 'mock-streaming', got '%s'", provider.Name())
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockStreamingProvider(3)
		var chunks []string
		callback := func(chunk string) { chunks = append(chunks, chunk) }

		ctx := context.Background()
		prompt := "Response JSON Schema:\n{}\n\nTransform: test\n\nInput: hello world"
		resp, err := provider.Stream(ctx, []Message{{Role: RoleUser, Content: prompt}}, 0.5, callback)
		if err != nil {
			t.Fatalf("Stream failed: %v", err)
		}
		if resp.Content == "" {
			t.Error("Expected non-empty content")
		}
		if len(chunks) == 0 {
			t.Error("Expected chunks from streaming")
		}
		// Verify chunks reassemble to full content
		combined := strings.Join(chunks, "")
		if combined != resp.Content {
			t.Errorf("Chunks should reassemble to full content: got '%s', want '%s'", combined, resp.Content)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// Also works as a regular Provider via Call
		provider := NewMockStreamingProvider(5)
		ctx := context.Background()
		resp, err := provider.Call(ctx, []Message{{Role: RoleUser, Content: "test"}}, 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}
		if resp.Content == "" {
			t.Error("Expected non-empty content from Call")
		}
	})
}

func TestMockStreamingProvider_ZeroChunkSize(t *testing.T) {
	provider := NewMockStreamingProvider(0)
	var chunks []string
	callback := func(chunk string) { chunks = append(chunks, chunk) }

	ctx := context.Background()
	resp, err := provider.Stream(ctx, []Message{{Role: RoleUser, Content: "test"}}, 0.5, callback)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk with chunkSize=0, got %d", len(chunks))
	}
	if chunks[0] != resp.Content {
		t.Error("Single chunk should equal full content")
	}
}

func TestMockStreamingProvider_ContextCancellation(t *testing.T) {
	provider := NewMockStreamingProvider(1) // 1 char per chunk to maximize iterations
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	prompt := "Response JSON Schema:\n{}\n\nTransform: test\n\nInput: hello world this is a long text"
	var chunks []string
	callback := func(chunk string) { chunks = append(chunks, chunk) }

	_, err := provider.Stream(ctx, []Message{{Role: RoleUser, Content: prompt}}, 0.5, callback)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

func TestMockStreamingProvider_Unavailable(t *testing.T) {
	provider := NewMockStreamingProvider(5)
	provider.SetAvailable(false)

	ctx := context.Background()
	_, err := provider.Stream(ctx, []Message{{Role: RoleUser, Content: "test"}}, 0.5, func(_ string) {})
	if err == nil {
		t.Error("Expected error from unavailable provider")
	}
}

func TestMockStreamingProvider_NilCallback(t *testing.T) {
	provider := NewMockStreamingProvider(5)
	ctx := context.Background()
	resp, err := provider.Stream(ctx, []Message{{Role: RoleUser, Content: "test"}}, 0.5, nil)
	if err != nil {
		t.Fatalf("Stream with nil callback failed: %v", err)
	}
	if resp.Content == "" {
		t.Error("Expected non-empty content")
	}
}

func TestNewMockToolProvider(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockToolProvider()
		if provider == nil {
			t.Fatal("NewMockToolProvider returned nil")
		}
		if provider.Name() != "mock-tool" {
			t.Errorf("Expected name 'mock-tool', got '%s'", provider.Name())
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Default response: calls first tool with empty input
		provider := NewMockToolProvider()
		ctx := context.Background()
		tools := []Tool{{Name: "get_weather", Description: "Get weather"}}
		resp, err := provider.CallWithTools(ctx, nil, 0.5, tools)
		if err != nil {
			t.Fatalf("CallWithTools failed: %v", err)
		}
		if resp.StopReason != StopReasonToolUse {
			t.Errorf("Expected StopReason='tool_use', got '%s'", resp.StopReason)
		}
		if len(resp.ToolCalls) != 1 {
			t.Fatalf("Expected 1 tool call, got %d", len(resp.ToolCalls))
		}
		if resp.ToolCalls[0].Name != "get_weather" {
			t.Errorf("Expected tool call name 'get_weather', got '%s'", resp.ToolCalls[0].Name)
		}
		if resp.ToolCalls[0].ID == "" {
			t.Error("Expected non-empty tool call ID")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// No tools — returns text response
		provider := NewMockToolProvider()
		ctx := context.Background()
		resp, err := provider.CallWithTools(ctx, nil, 0.5, nil)
		if err != nil {
			t.Fatalf("CallWithTools with no tools failed: %v", err)
		}
		if resp.StopReason != StopReasonEndTurn {
			t.Errorf("Expected StopReason='end_turn', got '%s'", resp.StopReason)
		}
		if resp.Content == "" {
			t.Error("Expected non-empty content when no tools provided")
		}
	})
}

func TestMockToolProvider_CustomResponse(t *testing.T) {
	provider := NewMockToolProvider().WithToolResponse(func(_ []Message, _ []Tool) *ProviderResponse {
		return &ProviderResponse{
			Content:    "custom response",
			StopReason: StopReasonEndTurn,
			Usage:      TokenUsage{Prompt: 50, Completion: 25, Total: 75},
		}
	})

	ctx := context.Background()
	resp, err := provider.CallWithTools(ctx, nil, 0.5, []Tool{{Name: "test"}})
	if err != nil {
		t.Fatalf("CallWithTools failed: %v", err)
	}
	if resp.Content != "custom response" {
		t.Errorf("Expected 'custom response', got '%s'", resp.Content)
	}
}

func TestMockToolProvider_Unavailable(t *testing.T) {
	provider := NewMockToolProvider()
	provider.SetAvailable(false)

	ctx := context.Background()
	_, err := provider.CallWithTools(ctx, nil, 0.5, []Tool{{Name: "test"}})
	if err == nil {
		t.Error("Expected error from unavailable provider")
	}
}

func TestNewMockToolStreamingProvider(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockToolStreamingProvider()
		if provider == nil {
			t.Fatal("NewMockToolStreamingProvider returned nil")
		}
		if provider.Name() != "mock-tool-streaming" {
			t.Errorf("Expected name 'mock-tool-streaming', got '%s'", provider.Name())
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Emits events for tool calls
		provider := NewMockToolStreamingProvider()
		var events []StreamEvent
		callback := func(e StreamEvent) { events = append(events, e) }

		ctx := context.Background()
		tools := []Tool{{Name: "get_weather"}}
		resp, err := provider.StreamWithTools(ctx, nil, 0.5, tools, callback)
		if err != nil {
			t.Fatalf("StreamWithTools failed: %v", err)
		}
		if resp.StopReason != StopReasonToolUse {
			t.Errorf("Expected StopReason='tool_use', got '%s'", resp.StopReason)
		}
		// Should have tool_start, tool_delta (for empty input {}), tool_end
		hasStart := false
		hasEnd := false
		for _, e := range events {
			if e.Type == StreamEventToolStart {
				hasStart = true
			}
			if e.Type == StreamEventToolEnd {
				hasEnd = true
			}
		}
		if !hasStart {
			t.Error("Expected tool_start event")
		}
		if !hasEnd {
			t.Error("Expected tool_end event")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// Nil callback — no events, still returns response
		provider := NewMockToolStreamingProvider()
		ctx := context.Background()
		tools := []Tool{{Name: "search"}}
		resp, err := provider.StreamWithTools(ctx, nil, 0.5, tools, nil)
		if err != nil {
			t.Fatalf("StreamWithTools with nil callback failed: %v", err)
		}
		if len(resp.ToolCalls) != 1 {
			t.Errorf("Expected 1 tool call, got %d", len(resp.ToolCalls))
		}
	})
}

func TestMockToolStreamingProvider_ErrorPath(t *testing.T) {
	provider := NewMockToolStreamingProvider()
	provider.SetAvailable(false)

	var events []StreamEvent
	_, err := provider.StreamWithTools(context.Background(), nil, 0.5, []Tool{{Name: "test"}}, func(e StreamEvent) {
		events = append(events, e)
	})
	if err == nil {
		t.Error("Expected error from unavailable provider")
	}
	if len(events) != 0 {
		t.Error("Should not emit events on error")
	}
}

func TestMockToolStreamingProvider_TextContent(t *testing.T) {
	// Custom response with both text and tool calls to cover text emission path
	provider := NewMockToolStreamingProvider()
	provider.WithToolResponse(func(_ []Message, _ []Tool) *ProviderResponse {
		return &ProviderResponse{
			Content:    "Here is the result",
			StopReason: StopReasonToolUse,
			ToolCalls: []ToolCall{
				{ID: "call_1", Name: "search", Input: json.RawMessage(`{"q":"test"}`)},
			},
			Usage: TokenUsage{Prompt: 100, Completion: 50, Total: 150},
		}
	})

	var events []StreamEvent
	callback := func(e StreamEvent) { events = append(events, e) }

	resp, err := provider.StreamWithTools(context.Background(), nil, 0.5, []Tool{{Name: "search"}}, callback)
	if err != nil {
		t.Fatalf("StreamWithTools failed: %v", err)
	}
	if resp.Content != "Here is the result" {
		t.Errorf("Expected content 'Here is the result', got '%s'", resp.Content)
	}

	// Should have text event + tool_start + tool_delta + tool_end
	hasText := false
	hasToolStart := false
	for _, e := range events {
		if e.Type == StreamEventText && e.Text == "Here is the result" {
			hasText = true
		}
		if e.Type == StreamEventToolStart {
			hasToolStart = true
		}
	}
	if !hasText {
		t.Error("Expected text event for content")
	}
	if !hasToolStart {
		t.Error("Expected tool_start event")
	}
}

func TestMockToolStreamingProvider_NoTools(t *testing.T) {
	// No tools — returns text response with text event
	provider := NewMockToolStreamingProvider()
	var events []StreamEvent
	callback := func(e StreamEvent) { events = append(events, e) }

	resp, err := provider.StreamWithTools(context.Background(), nil, 0.5, nil, callback)
	if err != nil {
		t.Fatalf("StreamWithTools failed: %v", err)
	}
	if resp.Content == "" {
		t.Error("Expected text content when no tools")
	}
	hasText := false
	for _, e := range events {
		if e.Type == StreamEventText {
			hasText = true
		}
	}
	if !hasText {
		t.Error("Expected text event when no tools provided")
	}
}

func TestMockToolStreamingProvider_ContextCancellation(t *testing.T) {
	t.Run("before_call", func(t *testing.T) {
		provider := NewMockToolStreamingProvider()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		tools := []Tool{{Name: "search"}}
		_, err := provider.StreamWithTools(ctx, nil, 0.5, tools, func(_ StreamEvent) {})
		if err == nil {
			t.Error("Expected context cancellation error")
		}
	})

	t.Run("cancel_before_text", func(t *testing.T) {
		// Context already canceled when text emission happens
		provider := NewMockToolStreamingProvider()
		provider.WithToolResponse(func(_ []Message, _ []Tool) *ProviderResponse {
			return &ProviderResponse{
				Content:    "some text",
				StopReason: StopReasonEndTurn,
				Usage:      TokenUsage{Prompt: 10, Completion: 5, Total: 15},
			}
		})

		// CallWithTools doesn't check ctx, so cancel after construction but it will
		// be detected at the select before text emission
		ctx, cancel := context.WithCancel(context.Background())
		// We need ctx canceled before the select runs. Use a callback that wraps
		// a provider returning content. Cancel ctx right before calling StreamWithTools
		// won't work because CallWithTools still succeeds.
		// Instead, wrap with a provider that cancels ctx during CallWithTools:
		cancel()
		_, err := provider.StreamWithTools(ctx, nil, 0.5, nil, func(_ StreamEvent) {})
		// CallWithTools on unavailable=true provider succeeds, but ctx is done
		// so the select before text emission should catch it
		if err == nil {
			t.Error("Expected context cancellation error")
		}
	})

	t.Run("cancel_before_tool_loop", func(t *testing.T) {
		// Cancel in the text callback so tool loop hits ctx.Done
		provider := NewMockToolStreamingProvider()
		provider.WithToolResponse(func(_ []Message, _ []Tool) *ProviderResponse {
			return &ProviderResponse{
				Content:    "text first",
				StopReason: StopReasonToolUse,
				ToolCalls:  []ToolCall{{ID: "call_1", Name: "search", Input: json.RawMessage(`{}`)}},
				Usage:      TokenUsage{Prompt: 10, Completion: 5, Total: 15},
			}
		})

		ctx, cancel := context.WithCancel(context.Background())
		var events []StreamEvent
		callback := func(e StreamEvent) {
			events = append(events, e)
			cancel() // Cancel after first event (text), tool loop should see ctx.Done
		}

		_, err := provider.StreamWithTools(ctx, nil, 0.5, []Tool{{Name: "search"}}, callback)
		if err == nil {
			t.Error("Expected context cancellation error after callback cancels")
		}
		if len(events) == 0 {
			t.Error("Expected at least one event before cancellation")
		}
	})
}

func TestMockProviderFixed_Name(t *testing.T) {
	provider := NewMockProviderWithResponse(`{"test": "value"}`)
	name := provider.Name()
	if name != MockFixedProviderName {
		t.Errorf("Expected '%s', got '%s'", MockFixedProviderName, name)
	}
}

func TestMockProviderCallback_Name(t *testing.T) {
	provider := NewMockProviderWithCallback(func(_ string, _ float32) (string, error) {
		return "test", nil
	})
	name := provider.Name()
	if name != "mock-callback" {
		t.Errorf("Expected 'mock-callback', got '%s'", name)
	}
}

func TestMockProviderError_Name(t *testing.T) {
	provider := NewMockProviderWithError("error")
	name := provider.Name()
	if name != "mock-error" {
		t.Errorf("Expected 'mock-error', got '%s'", name)
	}
}
