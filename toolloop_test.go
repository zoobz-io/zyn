package zyn

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// mockToolHandler is a test-local ToolHandler implementation.
type mockToolHandler struct {
	tools    []Tool
	execFunc func(ctx context.Context, call ToolCall) (string, error)
}

func newMockToolHandler(tools []Tool) *mockToolHandler {
	return &mockToolHandler{
		tools: tools,
		execFunc: func(_ context.Context, call ToolCall) (string, error) {
			return fmt.Sprintf("result for %s", call.Name), nil
		},
	}
}

func (m *mockToolHandler) ListTools() []Tool { return m.tools }

func (m *mockToolHandler) Execute(ctx context.Context, call ToolCall) (string, error) {
	return m.execFunc(ctx, call)
}

func (m *mockToolHandler) withExecFunc(fn func(context.Context, ToolCall) (string, error)) *mockToolHandler {
	m.execFunc = fn
	return m
}

// multiTurnProvider returns tool_use for the first N calls, then end_turn.
func multiTurnProvider(toolUseCount int) *MockToolProvider {
	callCount := 0
	return NewMockToolProvider().WithToolResponse(func(_ []Message, tools []Tool) *ProviderResponse {
		callCount++
		if callCount <= toolUseCount {
			toolName := "search"
			if len(tools) > 0 {
				toolName = tools[0].Name
			}
			return &ProviderResponse{
				StopReason: StopReasonToolUse,
				ToolCalls: []ToolCall{{
					ID:    fmt.Sprintf("call_%d", callCount),
					Name:  toolName,
					Input: json.RawMessage(`{}`),
				}},
				Usage: TokenUsage{Prompt: 100, Completion: 50, Total: 150},
			}
		}
		return &ProviderResponse{
			Content:    "final answer",
			StopReason: StopReasonEndTurn,
			Usage:      TokenUsage{Prompt: 100, Completion: 50, Total: 150},
		}
	})
}

func TestToolLoop(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		handler := newMockToolHandler([]Tool{{Name: "search"}})
		provider := NewMockToolProvider()
		synapse, err := ToolLoop(handler, provider)
		if err != nil {
			t.Fatalf("ToolLoop creation failed: %v", err)
		}
		if synapse == nil {
			t.Fatal("ToolLoop returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Nil handler
		_, err := ToolLoop(nil, NewMockToolProvider())
		if err == nil {
			t.Error("Expected error with nil handler")
		}

		// Non-ToolProvider
		handler := newMockToolHandler([]Tool{{Name: "search"}})
		_, err = ToolLoop(handler, NewMockProvider())
		if err == nil {
			t.Error("Expected error with non-ToolProvider")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		handler := newMockToolHandler([]Tool{{Name: "search"}})
		provider := NewMockToolProvider()
		synapse, err := ToolLoop(handler, provider,
			WithRetry(3),
			WithTimeout(10*time.Second))
		if err != nil {
			t.Fatalf("ToolLoop with options failed: %v", err)
		}
		if synapse.GetPipeline() == nil {
			t.Error("GetPipeline returned nil")
		}
	})
}

func TestToolLoopSynapse_Fire(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		// Single iteration — provider returns end_turn immediately
		provider := multiTurnProvider(0)
		handler := newMockToolHandler([]Tool{{Name: "search"}})
		synapse, _ := ToolLoop(handler, provider)

		result, err := synapse.Fire(context.Background(), NewSession(), "hello")
		if err != nil {
			t.Fatalf("Fire failed: %v", err)
		}
		if result != "final answer" {
			t.Errorf("Expected 'final answer', got '%s'", result)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Multi-turn: 2 tool_use iterations then end_turn
		provider := multiTurnProvider(2)
		handler := newMockToolHandler([]Tool{{Name: "search"}})
		synapse, _ := ToolLoop(handler, provider)

		session := NewSession()
		result, err := synapse.Fire(context.Background(), session, "search for something")
		if err != nil {
			t.Fatalf("Fire failed: %v", err)
		}
		if result != "final answer" {
			t.Errorf("Expected 'final answer', got '%s'", result)
		}
		// Session should have messages (user + assistant/tool pairs + final assistant)
		if session.Len() == 0 {
			t.Error("Expected session to have messages after loop")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// Max iterations hit — Completed should be false
		provider := multiTurnProvider(100) // Always returns tool_use
		handler := newMockToolHandler([]Tool{{Name: "search"}})
		synapse, _ := ToolLoop(handler, provider)
		synapse.WithMaxIterations(3)

		resp, err := synapse.FireWithDetails(context.Background(), NewSession(), "loop forever")
		if err != nil {
			t.Fatalf("FireWithDetails failed: %v", err)
		}
		if resp.Completed {
			t.Error("Expected Completed=false when max iterations hit")
		}
		if resp.Iterations != 3 {
			t.Errorf("Expected 3 iterations, got %d", resp.Iterations)
		}
	})
}

func TestToolLoopSynapse_FireWithDetails(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := multiTurnProvider(1)
		handler := newMockToolHandler([]Tool{{Name: "search"}})
		synapse, _ := ToolLoop(handler, provider)

		resp, err := synapse.FireWithDetails(context.Background(), NewSession(), "test")
		if err != nil {
			t.Fatalf("FireWithDetails failed: %v", err)
		}
		if resp.Content != "final answer" {
			t.Errorf("Expected content 'final answer', got '%s'", resp.Content)
		}
		if !resp.Completed {
			t.Error("Expected Completed=true")
		}
		if resp.Iterations != 2 {
			t.Errorf("Expected 2 iterations (1 tool_use + 1 end_turn), got %d", resp.Iterations)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Verify aggregated usage
		provider := multiTurnProvider(2)
		handler := newMockToolHandler([]Tool{{Name: "search"}})
		synapse, _ := ToolLoop(handler, provider)

		resp, err := synapse.FireWithDetails(context.Background(), NewSession(), "test")
		if err != nil {
			t.Fatalf("FireWithDetails failed: %v", err)
		}
		// 3 iterations × 150 total tokens each
		if resp.Usage.Total != 450 {
			t.Errorf("Expected aggregated Total=450, got %d", resp.Usage.Total)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// Verify ToolCallRecords
		provider := multiTurnProvider(2)
		handler := newMockToolHandler([]Tool{{Name: "search"}})
		synapse, _ := ToolLoop(handler, provider)

		resp, err := synapse.FireWithDetails(context.Background(), NewSession(), "test")
		if err != nil {
			t.Fatalf("FireWithDetails failed: %v", err)
		}
		if len(resp.Calls) != 2 {
			t.Fatalf("Expected 2 call records, got %d", len(resp.Calls))
		}
		if resp.Calls[0].Name != "search" {
			t.Errorf("Expected call name 'search', got '%s'", resp.Calls[0].Name)
		}
		if resp.Calls[0].Output != "result for search" {
			t.Errorf("Expected output 'result for search', got '%s'", resp.Calls[0].Output)
		}
		if resp.Calls[0].Iteration != 1 {
			t.Errorf("Expected iteration 1, got %d", resp.Calls[0].Iteration)
		}
		if resp.Calls[1].Iteration != 2 {
			t.Errorf("Expected iteration 2, got %d", resp.Calls[1].Iteration)
		}
	})
}

func TestToolLoopSynapse_FireStream(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		// Use a streaming provider that returns end_turn with content
		provider := NewMockToolStreamingProvider()
		provider.WithToolResponse(func(_ []Message, _ []Tool) *ProviderResponse {
			return &ProviderResponse{
				Content:    "streamed answer",
				StopReason: StopReasonEndTurn,
				Usage:      TokenUsage{Prompt: 100, Completion: 50, Total: 150},
			}
		})
		handler := newMockToolHandler([]Tool{{Name: "search"}})
		synapse, err := ToolLoop(handler, provider)
		if err != nil {
			t.Fatalf("ToolLoop creation failed: %v", err)
		}

		var events []StreamEvent
		callback := func(e StreamEvent) { events = append(events, e) }

		result, err := synapse.FireStream(context.Background(), NewSession(), "test", callback)
		if err != nil {
			t.Fatalf("FireStream failed: %v", err)
		}
		if result != "streamed answer" {
			t.Errorf("Expected 'streamed answer', got '%s'", result)
		}
		if len(events) == 0 {
			t.Error("Expected stream events")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Nil callback — falls back to non-streaming tool call
		provider := multiTurnProvider(1)
		handler := newMockToolHandler([]Tool{{Name: "search"}})
		synapse, _ := ToolLoop(handler, provider)

		result, err := synapse.FireStream(context.Background(), NewSession(), "test", nil)
		if err != nil {
			t.Fatalf("FireStream with nil callback failed: %v", err)
		}
		if result != "final answer" {
			t.Errorf("Expected 'final answer', got '%s'", result)
		}
	})
}

func TestToolLoopSynapse_FireStream_Error(t *testing.T) {
	provider := NewMockToolProvider()
	provider.SetAvailable(false)
	handler := newMockToolHandler([]Tool{{Name: "search"}})
	synapse, _ := ToolLoop(handler, provider)

	_, err := synapse.FireStream(context.Background(), NewSession(), "test", func(_ StreamEvent) {})
	if err == nil {
		t.Error("Expected error from unavailable provider")
	}
}

func TestToolLoopSynapse_HandlerError(t *testing.T) {
	t.Run("dispatch_failure", func(t *testing.T) {
		provider := multiTurnProvider(1)
		handler := newMockToolHandler([]Tool{{Name: "search"}}).
			withExecFunc(func(_ context.Context, _ ToolCall) (string, error) {
				return "", fmt.Errorf("tool not found")
			})
		synapse, _ := ToolLoop(handler, provider)

		_, err := synapse.Fire(context.Background(), NewSession(), "test")
		if err == nil {
			t.Error("Expected error from handler dispatch failure")
		}
	})

	t.Run("tool_error_continues", func(t *testing.T) {
		// Tool returns error content (not dispatch error) — loop continues
		provider := multiTurnProvider(1)
		handler := newMockToolHandler([]Tool{{Name: "search"}}).
			withExecFunc(func(_ context.Context, _ ToolCall) (string, error) {
				return "error: invalid input", nil
			})
		synapse, _ := ToolLoop(handler, provider)

		result, err := synapse.Fire(context.Background(), NewSession(), "test")
		if err != nil {
			t.Fatalf("Fire should not fail on tool error content: %v", err)
		}
		if result != "final answer" {
			t.Errorf("Expected 'final answer', got '%s'", result)
		}
	})
}

func TestToolLoopSynapse_SessionUpdate(t *testing.T) {
	provider := multiTurnProvider(1)
	handler := newMockToolHandler([]Tool{{Name: "search"}})
	synapse, _ := ToolLoop(handler, provider)

	session := NewSession()
	_, err := synapse.Fire(context.Background(), session, "test input")
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	msgs := session.Messages()
	if len(msgs) == 0 {
		t.Fatal("Expected messages in session")
	}
	// First message should be user input
	if msgs[0].Role != RoleUser || msgs[0].Content != "test input" {
		t.Errorf("Expected first message to be user input, got role=%s content=%s", msgs[0].Role, msgs[0].Content)
	}
	// Should have assistant message with tool calls
	hasAssistantToolCall := false
	hasToolResult := false
	for _, msg := range msgs {
		if msg.Role == RoleAssistant && len(msg.ToolCalls) > 0 {
			hasAssistantToolCall = true
		}
		if msg.Role == RoleTool {
			hasToolResult = true
		}
	}
	if !hasAssistantToolCall {
		t.Error("Expected assistant message with tool calls in session")
	}
	if !hasToolResult {
		t.Error("Expected tool result message in session")
	}
}

func TestToolLoopSynapse_WithMaxIterations(t *testing.T) {
	handler := newMockToolHandler([]Tool{{Name: "search"}})
	synapse, _ := ToolLoop(handler, NewMockToolProvider())

	result := synapse.WithMaxIterations(5)
	if result != synapse {
		t.Error("WithMaxIterations should return the same synapse for chaining")
	}
	if synapse.maxIterations != 5 {
		t.Errorf("Expected maxIterations=5, got %d", synapse.maxIterations)
	}
}

func TestToolLoopSynapse_GetPipeline(t *testing.T) {
	handler := newMockToolHandler([]Tool{{Name: "search"}})
	synapse, _ := ToolLoop(handler, NewMockToolProvider())

	pipeline := synapse.GetPipeline()
	if pipeline == nil {
		t.Error("GetPipeline returned nil")
	}

	// Implements ServiceProvider
	var _ ServiceProvider = synapse
}

func TestToolLoopResponse_Validate(t *testing.T) {
	resp := ToolLoopResponse{}
	if err := resp.Validate(); err != nil {
		t.Errorf("ToolLoopResponse.Validate should not error: %v", err)
	}
}
