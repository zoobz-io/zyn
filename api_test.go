package zyn

import "testing"

func TestRoleConstants(t *testing.T) {
	t.Run("role_user", func(t *testing.T) {
		if RoleUser != "user" {
			t.Errorf("expected RoleUser='user', got '%s'", RoleUser)
		}
	})

	t.Run("role_assistant", func(t *testing.T) {
		if RoleAssistant != "assistant" {
			t.Errorf("expected RoleAssistant='assistant', got '%s'", RoleAssistant)
		}
	})

	t.Run("role_system", func(t *testing.T) {
		if RoleSystem != "system" {
			t.Errorf("expected RoleSystem='system', got '%s'", RoleSystem)
		}
	})
}

func TestTemperatureConstants(t *testing.T) {
	t.Run("temperature_unset", func(t *testing.T) {
		if TemperatureUnset >= 0 {
			t.Error("TemperatureUnset should be negative to distinguish from valid temperatures")
		}
	})

	t.Run("temperature_zero", func(t *testing.T) {
		if TemperatureZero <= 0 {
			t.Error("TemperatureZero should be positive (near-zero)")
		}
		if TemperatureZero >= 0.01 {
			t.Error("TemperatureZero should be very small")
		}
	})

	t.Run("temperature_deterministic", func(t *testing.T) {
		if DefaultTemperatureDeterministic <= 0 || DefaultTemperatureDeterministic > 1 {
			t.Errorf("DefaultTemperatureDeterministic should be 0-1, got %f", DefaultTemperatureDeterministic)
		}
	})

	t.Run("temperature_analytical", func(t *testing.T) {
		if DefaultTemperatureAnalytical <= 0 || DefaultTemperatureAnalytical > 1 {
			t.Errorf("DefaultTemperatureAnalytical should be 0-1, got %f", DefaultTemperatureAnalytical)
		}
	})

	t.Run("temperature_creative", func(t *testing.T) {
		if DefaultTemperatureCreative <= 0 || DefaultTemperatureCreative > 1 {
			t.Errorf("DefaultTemperatureCreative should be 0-1, got %f", DefaultTemperatureCreative)
		}
	})

	t.Run("temperature_ordering", func(t *testing.T) {
		if DefaultTemperatureDeterministic >= DefaultTemperatureAnalytical {
			t.Error("Deterministic temperature should be lower than Analytical")
		}
		if DefaultTemperatureAnalytical >= DefaultTemperatureCreative {
			t.Error("Analytical temperature should be lower than Creative")
		}
	})
}

func TestTokenUsage(t *testing.T) {
	t.Run("zero_values", func(t *testing.T) {
		usage := TokenUsage{}
		if usage.Prompt != 0 || usage.Completion != 0 || usage.Total != 0 {
			t.Error("zero-value TokenUsage should have all zeros")
		}
	})

	t.Run("field_assignment", func(t *testing.T) {
		usage := TokenUsage{
			Prompt:     100,
			Completion: 50,
			Total:      150,
		}
		if usage.Prompt != 100 {
			t.Errorf("expected Prompt=100, got %d", usage.Prompt)
		}
		if usage.Completion != 50 {
			t.Errorf("expected Completion=50, got %d", usage.Completion)
		}
		if usage.Total != 150 {
			t.Errorf("expected Total=150, got %d", usage.Total)
		}
	})
}

func TestMessage(t *testing.T) {
	t.Run("user_message", func(t *testing.T) {
		msg := Message{Role: RoleUser, Content: "hello"}
		if msg.Role != "user" {
			t.Errorf("expected Role='user', got '%s'", msg.Role)
		}
		if msg.Content != "hello" {
			t.Errorf("expected Content='hello', got '%s'", msg.Content)
		}
	})

	t.Run("assistant_message", func(t *testing.T) {
		msg := Message{Role: RoleAssistant, Content: "hi there"}
		if msg.Role != "assistant" {
			t.Errorf("expected Role='assistant', got '%s'", msg.Role)
		}
	})

	t.Run("system_message", func(t *testing.T) {
		msg := Message{Role: RoleSystem, Content: "you are helpful"}
		if msg.Role != "system" {
			t.Errorf("expected Role='system', got '%s'", msg.Role)
		}
	})
}

func TestSynapseRequest(t *testing.T) {
	t.Run("zero_value", func(t *testing.T) {
		req := SynapseRequest{}
		if req.Prompt != nil {
			t.Error("zero-value SynapseRequest should have nil Prompt")
		}
		if req.Temperature != 0 {
			t.Error("zero-value SynapseRequest should have 0 Temperature")
		}
		if req.Error != nil {
			t.Error("zero-value SynapseRequest should have nil Error")
		}
		if req.StreamCallback != nil {
			t.Error("zero-value SynapseRequest should have nil StreamCallback")
		}
	})

	t.Run("field_assignment", func(t *testing.T) {
		req := SynapseRequest{
			SessionID:    "session-123",
			RequestID:    "request-456",
			SynapseType:  "binary",
			ProviderName: "mock",
			Temperature:  0.5,
		}
		if req.SessionID != "session-123" {
			t.Errorf("expected SessionID='session-123', got '%s'", req.SessionID)
		}
		if req.SynapseType != "binary" {
			t.Errorf("expected SynapseType='binary', got '%s'", req.SynapseType)
		}
	})

	t.Run("stream_callback_field", func(t *testing.T) {
		called := false
		req := SynapseRequest{
			StreamCallback: func(_ string) { called = true },
		}
		req.StreamCallback("test")
		if !called {
			t.Error("StreamCallback should have been called")
		}
	})
}

func TestStreamingProvider(t *testing.T) {
	t.Run("simple", func(_ *testing.T) {
		// MockStreamingProvider satisfies StreamingProvider
		var _ StreamingProvider = NewMockStreamingProvider(5)
	})

	t.Run("reliability", func(t *testing.T) {
		// Regular MockProvider does NOT implement StreamingProvider
		provider := NewMockProvider()
		_, ok := provider.(StreamingProvider)
		if ok {
			t.Error("MockProvider should not implement StreamingProvider")
		}
	})

	t.Run("chaining", func(_ *testing.T) {
		// MockStreamingProvider also satisfies Provider
		var _ Provider = NewMockStreamingProvider(5)
	})
}

func TestRoleTool(t *testing.T) {
	if RoleTool != "tool" {
		t.Errorf("expected RoleTool='tool', got '%s'", RoleTool)
	}
}

func TestStopReasonConstants(t *testing.T) {
	t.Run("end_turn", func(t *testing.T) {
		if StopReasonEndTurn != "end_turn" {
			t.Errorf("expected StopReasonEndTurn='end_turn', got '%s'", StopReasonEndTurn)
		}
	})

	t.Run("tool_use", func(t *testing.T) {
		if StopReasonToolUse != "tool_use" {
			t.Errorf("expected StopReasonToolUse='tool_use', got '%s'", StopReasonToolUse)
		}
	})

	t.Run("max_tokens", func(t *testing.T) {
		if StopReasonMaxTokens != "max_tokens" {
			t.Errorf("expected StopReasonMaxTokens='max_tokens', got '%s'", StopReasonMaxTokens)
		}
	})
}

func TestProviderResponse_ToolFields(t *testing.T) {
	t.Run("zero_value", func(t *testing.T) {
		resp := ProviderResponse{}
		if resp.ToolCalls != nil {
			t.Error("zero-value ProviderResponse should have nil ToolCalls")
		}
		if resp.StopReason != "" {
			t.Error("zero-value ProviderResponse should have empty StopReason")
		}
	})

	t.Run("backward_compatible", func(t *testing.T) {
		// Existing code that constructs ProviderResponse without new fields still works
		resp := ProviderResponse{
			Content: "hello",
			Usage:   TokenUsage{Prompt: 10, Completion: 5, Total: 15},
		}
		if resp.Content != "hello" {
			t.Errorf("expected Content='hello', got '%s'", resp.Content)
		}
		if resp.ToolCalls != nil {
			t.Error("omitted ToolCalls should be nil")
		}
		if resp.StopReason != "" {
			t.Error("omitted StopReason should be empty")
		}
	})
}

func TestMessage_ToolFields(t *testing.T) {
	t.Run("zero_value", func(t *testing.T) {
		msg := Message{}
		if msg.ToolCalls != nil {
			t.Error("zero-value Message should have nil ToolCalls")
		}
		if msg.ToolCallID != "" {
			t.Error("zero-value Message should have empty ToolCallID")
		}
	})

	t.Run("backward_compatible", func(t *testing.T) {
		// Existing code that constructs Message without new fields still works
		msg := Message{Role: RoleUser, Content: "hello"}
		if msg.Role != RoleUser {
			t.Errorf("expected Role='user', got '%s'", msg.Role)
		}
		if msg.ToolCalls != nil {
			t.Error("omitted ToolCalls should be nil")
		}
		if msg.ToolCallID != "" {
			t.Error("omitted ToolCallID should be empty")
		}
	})

	t.Run("tool_message", func(t *testing.T) {
		msg := Message{
			Role:       RoleTool,
			Content:    `{"temperature": 72}`,
			ToolCallID: "call_123",
		}
		if msg.Role != "tool" {
			t.Errorf("expected Role='tool', got '%s'", msg.Role)
		}
		if msg.ToolCallID != "call_123" {
			t.Errorf("expected ToolCallID='call_123', got '%s'", msg.ToolCallID)
		}
	})
}

func TestToolProvider(t *testing.T) {
	t.Run("simple", func(_ *testing.T) {
		// MockToolProvider satisfies ToolProvider
		var _ ToolProvider = NewMockToolProvider()
	})

	t.Run("reliability", func(t *testing.T) {
		// Regular MockProvider does NOT implement ToolProvider
		provider := NewMockProvider()
		_, ok := provider.(ToolProvider)
		if ok {
			t.Error("MockProvider should not implement ToolProvider")
		}
	})

	t.Run("chaining", func(_ *testing.T) {
		// MockToolProvider also satisfies Provider
		var _ Provider = NewMockToolProvider()
	})
}

func TestSynapseRequest_ToolsField(t *testing.T) {
	t.Run("zero_value", func(t *testing.T) {
		req := SynapseRequest{}
		if req.Tools != nil {
			t.Error("zero-value SynapseRequest should have nil Tools")
		}
	})

	t.Run("with_tools", func(t *testing.T) {
		req := SynapseRequest{
			Tools: []Tool{
				{Name: "search", Description: "Search the web"},
			},
		}
		if len(req.Tools) != 1 {
			t.Errorf("expected 1 tool, got %d", len(req.Tools))
		}
		if req.Tools[0].Name != "search" {
			t.Errorf("expected tool name 'search', got '%s'", req.Tools[0].Name)
		}
	})
}
