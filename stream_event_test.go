package zyn

import "testing"

func TestStreamEventType_Constants(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		if StreamEventText != "text" {
			t.Errorf("expected StreamEventText='text', got '%s'", StreamEventText)
		}
		if StreamEventToolStart != "tool_start" {
			t.Errorf("expected StreamEventToolStart='tool_start', got '%s'", StreamEventToolStart)
		}
		if StreamEventToolDelta != "tool_delta" {
			t.Errorf("expected StreamEventToolDelta='tool_delta', got '%s'", StreamEventToolDelta)
		}
		if StreamEventToolEnd != "tool_end" {
			t.Errorf("expected StreamEventToolEnd='tool_end', got '%s'", StreamEventToolEnd)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		// All event types are distinct
		types := []StreamEventType{StreamEventText, StreamEventToolStart, StreamEventToolDelta, StreamEventToolEnd}
		seen := make(map[StreamEventType]bool)
		for _, typ := range types {
			if seen[typ] {
				t.Errorf("duplicate stream event type: %s", typ)
			}
			seen[typ] = true
		}
	})
}

func TestStreamEvent_ZeroValue(t *testing.T) {
	event := StreamEvent{}
	if event.Type != "" {
		t.Error("zero-value StreamEvent should have empty Type")
	}
	if event.Text != "" {
		t.Error("zero-value StreamEvent should have empty Text")
	}
	if event.ToolCall != nil {
		t.Error("zero-value StreamEvent should have nil ToolCall")
	}
	if event.ToolDelta != "" {
		t.Error("zero-value StreamEvent should have empty ToolDelta")
	}
	if event.Index != 0 {
		t.Error("zero-value StreamEvent should have zero Index")
	}
}

func TestStreamEvent_TextEvent(t *testing.T) {
	event := StreamEvent{
		Type: StreamEventText,
		Text: "Hello, world",
	}
	if event.Type != StreamEventText {
		t.Errorf("expected type text, got %s", event.Type)
	}
	if event.Text != "Hello, world" {
		t.Errorf("expected text 'Hello, world', got '%s'", event.Text)
	}
}

func TestStreamEvent_ToolStartEvent(t *testing.T) {
	event := StreamEvent{
		Type: StreamEventToolStart,
		ToolCall: &ToolCall{
			ID:   "call_123",
			Name: "get_weather",
		},
		Index: 0,
	}
	if event.Type != StreamEventToolStart {
		t.Errorf("expected type tool_start, got %s", event.Type)
	}
	if event.ToolCall == nil {
		t.Fatal("expected non-nil ToolCall")
	}
	if event.ToolCall.ID != "call_123" {
		t.Errorf("expected ToolCall.ID='call_123', got '%s'", event.ToolCall.ID)
	}
	if event.ToolCall.Name != "get_weather" {
		t.Errorf("expected ToolCall.Name='get_weather', got '%s'", event.ToolCall.Name)
	}
}

func TestStreamEvent_ToolDeltaEvent(t *testing.T) {
	event := StreamEvent{
		Type:      StreamEventToolDelta,
		ToolDelta: `{"city": "San`,
		Index:     0,
	}
	if event.Type != StreamEventToolDelta {
		t.Errorf("expected type tool_delta, got %s", event.Type)
	}
	if event.ToolDelta != `{"city": "San` {
		t.Errorf("unexpected ToolDelta value: %s", event.ToolDelta)
	}
}

func TestStreamEvent_ToolEndEvent(t *testing.T) {
	event := StreamEvent{
		Type:  StreamEventToolEnd,
		Index: 1,
	}
	if event.Type != StreamEventToolEnd {
		t.Errorf("expected type tool_end, got %s", event.Type)
	}
	if event.Index != 1 {
		t.Errorf("expected Index=1, got %d", event.Index)
	}
}

func TestStreamEventCallback_Type(t *testing.T) {
	var received StreamEvent
	var callback StreamEventCallback = func(event StreamEvent) {
		received = event
	}

	callback(StreamEvent{Type: StreamEventText, Text: "test"})

	if received.Type != StreamEventText {
		t.Errorf("expected received type text, got %s", received.Type)
	}
	if received.Text != "test" {
		t.Errorf("expected received text 'test', got '%s'", received.Text)
	}
}
