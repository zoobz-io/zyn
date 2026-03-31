package zyn

import (
	"encoding/json"
	"fmt"
)

// Tool defines a tool that can be passed to an LLM provider.
// Tools describe functions the model can request to invoke during a conversation.
// Parameters uses json.RawMessage to carry a JSON Schema without parsing it —
// consumers handle schema interpretation.
type Tool struct {
	Name        string          // Tool name (required)
	Description string          // What the tool does
	Parameters  json.RawMessage // JSON Schema for input parameters (optional, must be valid JSON if present)
}

// Validate checks that the tool definition is valid.
// Name is required. Parameters, if present, must be valid JSON.
func (t Tool) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if len(t.Parameters) > 0 {
		if !json.Valid(t.Parameters) {
			return fmt.Errorf("tool parameters must be valid JSON")
		}
	}
	return nil
}

// ToolCall represents a tool invocation request from the model.
// When the model decides to use a tool, it returns one or more ToolCall values
// describing which tool to call and with what input.
type ToolCall struct {
	ID    string          // Provider-assigned ID for this tool call (required)
	Name  string          // Tool name the model wants to invoke (required)
	Input json.RawMessage // Tool input as JSON
}

// Validate checks that the tool call is valid.
// ID and Name are both required.
func (tc ToolCall) Validate() error {
	if tc.ID == "" {
		return fmt.Errorf("tool call ID is required")
	}
	if tc.Name == "" {
		return fmt.Errorf("tool call name is required")
	}
	return nil
}
