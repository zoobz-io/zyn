package zyn

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/zoobz-io/capitan"
	"github.com/zoobz-io/pipz"
)

// ToolHandler dispatches tool calls back to consumer code.
// Intentionally compatible with ago.Executor.
//
// Execute error semantics:
//   - Returns ("", err) → dispatch failure, loop stops, error propagated
//   - Returns ("error: ...", nil) → tool error, sent as tool result, loop continues
type ToolHandler interface {
	ListTools() []Tool
	Execute(ctx context.Context, call ToolCall) (string, error)
}

// ToolLoopResponse contains the result of a multi-turn tool execution loop.
type ToolLoopResponse struct {
	Content    string           `json:"content"`
	Completed  bool             `json:"completed"`
	Iterations int              `json:"iterations"`
	Calls      []ToolCallRecord `json:"calls"`
	Usage      TokenUsage       `json:"usage"`
}

// Validate checks that the response is valid.
func (ToolLoopResponse) Validate() error {
	return nil
}

// ToolCallRecord describes a single tool invocation within the loop.
type ToolCallRecord struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	Output    string          `json:"output"`
	Error     bool            `json:"error"`
	Iteration int             `json:"iteration"`
}

// ToolLoopSynapse wraps a multi-turn tool execution loop with reliability,
// session management, and signal emission. Each provider call within the loop
// runs through a pipz pipeline with the configured options.
type ToolLoopSynapse struct {
	handler       ToolHandler
	pipeline      pipz.Chainable[*SynapseRequest]
	providerName  string
	maxIterations int
	temperature   float32
}

// ToolLoop creates a new tool loop synapse.
// The provider must implement ToolProvider.
// Options (retry, timeout, circuit breaker, etc.) apply per provider call, not per loop.
func ToolLoop(handler ToolHandler, provider Provider, opts ...Option) (*ToolLoopSynapse, error) {
	if handler == nil {
		return nil, fmt.Errorf("tool handler is required")
	}
	if _, ok := provider.(ToolProvider); !ok {
		return nil, fmt.Errorf("provider %s does not implement ToolProvider", provider.Name())
	}

	pipeline := NewRawTerminal(provider)
	for _, opt := range opts {
		pipeline = opt(pipeline)
	}

	return &ToolLoopSynapse{
		handler:       handler,
		pipeline:      pipeline,
		providerName:  provider.Name(),
		maxIterations: 10,
		temperature:   DefaultTemperatureDeterministic,
	}, nil
}

// WithMaxIterations sets the maximum number of loop iterations.
// Each iteration is one provider call. Default is 10.
func (t *ToolLoopSynapse) WithMaxIterations(n int) *ToolLoopSynapse {
	t.maxIterations = n
	return t
}

// GetPipeline returns the underlying pipeline. Implements ServiceProvider.
func (t *ToolLoopSynapse) GetPipeline() pipz.Chainable[*SynapseRequest] {
	return t.pipeline
}

// Fire runs the tool loop and returns the final text content.
func (t *ToolLoopSynapse) Fire(ctx context.Context, session *Session, input string) (string, error) {
	resp, err := t.fireLoop(ctx, session, input, nil)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

// FireWithDetails runs the tool loop and returns the full response with call records.
func (t *ToolLoopSynapse) FireWithDetails(ctx context.Context, session *Session, input string) (*ToolLoopResponse, error) {
	return t.fireLoop(ctx, session, input, nil)
}

// FireStream runs the tool loop with streaming, delivering typed events per call.
// If the provider implements ToolStreamingProvider, each call streams via the callback.
func (t *ToolLoopSynapse) FireStream(ctx context.Context, session *Session, input string, callback StreamEventCallback) (string, error) {
	resp, err := t.fireLoop(ctx, session, input, callback)
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (t *ToolLoopSynapse) fireLoop(ctx context.Context, session *Session, input string, streamCallback StreamEventCallback) (*ToolLoopResponse, error) {
	requestID := uuid.New().String()

	capitan.Info(ctx, RequestStarted,
		RequestIDKey.Field(requestID),
		SynapseTypeKey.Field("toolloop"),
		ProviderKey.Field(t.providerName),
		InputKey.Field(input),
		TemperatureKey.Field(float64(t.temperature)),
	)

	tools := t.handler.ListTools()

	// Build initial messages: session history + user input
	sessionMessages := session.Messages()
	messages := make([]Message, len(sessionMessages), len(sessionMessages)+1)
	copy(messages, sessionMessages)
	messages = append(messages, Message{
		Role:    RoleUser,
		Content: input,
	})

	var allRecords []ToolCallRecord
	var aggregatedUsage TokenUsage
	var finalContent string
	completed := false
	iterationCount := 0

	for iteration := 1; iteration <= t.maxIterations; iteration++ {
		iterationCount = iteration

		req := &SynapseRequest{
			Messages:            messages,
			Temperature:         t.temperature,
			Tools:               tools,
			StreamEventCallback: streamCallback,
			SessionID:           session.ID(),
			RequestID:           requestID,
			SynapseType:         "toolloop",
			ProviderName:        t.providerName,
		}

		processed, err := t.pipeline.Process(ctx, req)
		if err != nil {
			capitan.Error(ctx, RequestFailed,
				RequestIDKey.Field(requestID),
				SynapseTypeKey.Field("toolloop"),
				ProviderKey.Field(t.providerName),
				ErrorKey.Field(err.Error()),
			)
			return nil, err
		}

		// Aggregate usage
		if processed.Usage != nil {
			aggregatedUsage.Prompt += processed.Usage.Prompt
			aggregatedUsage.Completion += processed.Usage.Completion
			aggregatedUsage.Total += processed.Usage.Total
		}

		capitan.Info(ctx, ToolLoopIteration,
			RequestIDKey.Field(requestID),
			IterationKey.Field(iteration),
			ResponseFinishReasonKey.Field(processed.StopReason),
		)

		// Not a tool call — done
		if processed.StopReason != StopReasonToolUse {
			finalContent = processed.Response
			completed = true
			break
		}

		// Append assistant message with tool calls
		messages = append(messages, Message{
			Role:      RoleAssistant,
			Content:   processed.Response,
			ToolCalls: processed.ResponseCalls,
		})

		// Execute each tool call
		for _, tc := range processed.ResponseCalls {
			capitan.Info(ctx, ToolLoopDispatch,
				RequestIDKey.Field(requestID),
				IterationKey.Field(iteration),
				ToolNameKey.Field(tc.Name),
				ToolCallIDKey.Field(tc.ID),
			)

			output, execErr := t.handler.Execute(ctx, tc)
			if execErr != nil {
				capitan.Error(ctx, RequestFailed,
					RequestIDKey.Field(requestID),
					SynapseTypeKey.Field("toolloop"),
					ProviderKey.Field(t.providerName),
					ErrorKey.Field(execErr.Error()),
					ToolNameKey.Field(tc.Name),
				)
				return nil, fmt.Errorf("tool %s dispatch failed: %w", tc.Name, execErr)
			}

			allRecords = append(allRecords, ToolCallRecord{
				ID:        tc.ID,
				Name:      tc.Name,
				Input:     tc.Input,
				Output:    output,
				Iteration: iteration,
			})

			messages = append(messages, Message{
				Role:       RoleTool,
				Content:    output,
				ToolCallID: tc.ID,
			})
		}
	}

	// Update session transactionally with full conversation
	session.SetMessages(messages)
	session.SetUsage(&aggregatedUsage)

	capitan.Info(ctx, RequestCompleted,
		RequestIDKey.Field(requestID),
		SynapseTypeKey.Field("toolloop"),
		ProviderKey.Field(t.providerName),
		OutputKey.Field(finalContent),
	)

	return &ToolLoopResponse{
		Content:    finalContent,
		Completed:  completed,
		Iterations: iterationCount,
		Calls:      allRecords,
		Usage:      aggregatedUsage,
	}, nil
}
