package zyn

// StreamEventType identifies the kind of stream event.
type StreamEventType string

// Stream event type constants.
const (
	// StreamEventText indicates a text content chunk.
	StreamEventText StreamEventType = "text"

	// StreamEventToolStart indicates the beginning of a tool call in the stream.
	// The ToolCall field will have ID and Name set; Input may be empty or partial.
	StreamEventToolStart StreamEventType = "tool_start"

	// StreamEventToolDelta indicates a partial update to tool call input JSON.
	// The ToolDelta field contains the incremental JSON fragment.
	StreamEventToolDelta StreamEventType = "tool_delta"

	// StreamEventToolEnd indicates the completion of a tool call in the stream.
	StreamEventToolEnd StreamEventType = "tool_end"
)

// StreamEvent represents a typed event during streaming with tool support.
// Different event types populate different fields:
//   - text: Text field contains the content chunk
//   - tool_start: ToolCall field contains the initial tool call (ID, Name set)
//   - tool_delta: ToolDelta field contains partial Input JSON
//   - tool_end: Index identifies which tool call completed
type StreamEvent struct {
	Type      StreamEventType // The event type
	Text      string          // For text events: the content chunk
	ToolCall  *ToolCall       // For tool_start events: the initial tool call
	ToolDelta string          // For tool_delta events: partial Input JSON fragment
	Index     int             // Tool call index within the response
}

// StreamEventCallback receives typed stream events during tool-aware streaming.
// This is the tool-aware counterpart to StreamCallback.
type StreamEventCallback func(event StreamEvent)
