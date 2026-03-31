package zyn

import (
	"encoding/json"
	"testing"
)

func TestTool_Validate(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		tool := Tool{Name: "get_weather"}
		if err := tool.Validate(); err != nil {
			t.Errorf("valid tool should not error: %v", err)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Name required
		tool := Tool{}
		if err := tool.Validate(); err == nil {
			t.Error("tool with empty name should fail validation")
		}

		// Valid parameters
		tool = Tool{
			Name:       "search",
			Parameters: json.RawMessage(`{"type": "object", "properties": {"query": {"type": "string"}}}`),
		}
		if err := tool.Validate(); err != nil {
			t.Errorf("tool with valid JSON parameters should not error: %v", err)
		}

		// Invalid parameters
		tool = Tool{
			Name:       "search",
			Parameters: json.RawMessage(`{not valid json`),
		}
		if err := tool.Validate(); err == nil {
			t.Error("tool with invalid JSON parameters should fail validation")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// Nil parameters is valid (optional field)
		tool := Tool{
			Name:        "simple_tool",
			Description: "A tool with no parameters",
		}
		if err := tool.Validate(); err != nil {
			t.Errorf("tool with nil parameters should be valid: %v", err)
		}

		// Empty RawMessage is valid (treated as not present)
		tool = Tool{
			Name:       "another_tool",
			Parameters: json.RawMessage{},
		}
		if err := tool.Validate(); err != nil {
			t.Errorf("tool with empty parameters should be valid: %v", err)
		}
	})
}

func TestToolCall_Validate(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		tc := ToolCall{ID: "call_123", Name: "get_weather"}
		if err := tc.Validate(); err != nil {
			t.Errorf("valid tool call should not error: %v", err)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// Missing ID
		tc := ToolCall{Name: "get_weather"}
		if err := tc.Validate(); err == nil {
			t.Error("tool call with empty ID should fail validation")
		}

		// Missing Name
		tc = ToolCall{ID: "call_123"}
		if err := tc.Validate(); err == nil {
			t.Error("tool call with empty name should fail validation")
		}

		// Both missing
		tc = ToolCall{}
		if err := tc.Validate(); err == nil {
			t.Error("tool call with empty ID and name should fail validation")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// With input
		tc := ToolCall{
			ID:    "call_456",
			Name:  "search",
			Input: json.RawMessage(`{"query": "golang tool use"}`),
		}
		if err := tc.Validate(); err != nil {
			t.Errorf("tool call with input should be valid: %v", err)
		}

		// Nil input is valid (optional)
		tc = ToolCall{ID: "call_789", Name: "no_args"}
		if err := tc.Validate(); err != nil {
			t.Errorf("tool call with nil input should be valid: %v", err)
		}
	})
}

func TestTool_ZeroValue(t *testing.T) {
	tool := Tool{}
	if tool.Name != "" {
		t.Error("zero-value Tool should have empty Name")
	}
	if tool.Description != "" {
		t.Error("zero-value Tool should have empty Description")
	}
	if tool.Parameters != nil {
		t.Error("zero-value Tool should have nil Parameters")
	}
}

func TestToolCall_ZeroValue(t *testing.T) {
	tc := ToolCall{}
	if tc.ID != "" {
		t.Error("zero-value ToolCall should have empty ID")
	}
	if tc.Name != "" {
		t.Error("zero-value ToolCall should have empty Name")
	}
	if tc.Input != nil {
		t.Error("zero-value ToolCall should have nil Input")
	}
}
