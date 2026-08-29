package zyn

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/zoobz-io/capitan"
	"github.com/zoobz-io/pipz"
)

// Identity for the LLM terminal processor.
var terminalID = pipz.NewIdentity("zyn:terminal", "LLM provider terminal")

// Service provides type-safe LLM interactions for a specific response type T.
// It wraps a pipz pipeline and handles JSON parsing of responses.
// T must implement Validator to ensure response validation.
type Service[T Validator] struct {
	pipeline           pipz.Chainable[*SynapseRequest]
	synapseType        string
	providerName       string
	defaultTemperature float32
}

// NewService creates a new Service with the given pipeline, synapse type, provider, and default temperature.
// The default temperature is used when no temperature is specified in Execute calls.
func NewService[T Validator](pipeline pipz.Chainable[*SynapseRequest], synapseType string, provider Provider, defaultTemperature float32) *Service[T] {
	return &Service[T]{
		pipeline:           pipeline,
		synapseType:        synapseType,
		providerName:       provider.Name(),
		defaultTemperature: defaultTemperature,
	}
}

// NewTerminal creates a terminal processor that calls the provider with session messages.
// This is the common terminal processor used by all synapse types.
func NewTerminal(provider Provider) pipz.Chainable[*SynapseRequest] {
	return pipz.Apply(terminalID, func(ctx context.Context, req *SynapseRequest) (*SynapseRequest, error) {
		// Build messages array from session + new prompt
		messages := make([]Message, len(req.Messages)+1)
		copy(messages, req.Messages)

		// Add new user message with rendered prompt
		promptStr := req.Prompt.Render()
		messages[len(messages)-1] = Message{
			Role:    RoleUser,
			Content: promptStr,
		}

		// Route to the appropriate provider method based on request capabilities.
		// Priority: tools+streaming → tools → text streaming → standard call.
		var resp *ProviderResponse
		var err error
		if len(req.Tools) > 0 && req.StreamEventCallback != nil {
			tsp, ok := provider.(ToolStreamingProvider)
			if !ok {
				return req, fmt.Errorf("provider %s does not support streaming with tools", provider.Name())
			}
			resp, err = tsp.StreamWithTools(ctx, messages, req.Temperature, req.Tools, req.StreamEventCallback)
		} else if len(req.Tools) > 0 {
			tp, ok := provider.(ToolProvider)
			if !ok {
				return req, fmt.Errorf("provider %s does not support tools", provider.Name())
			}
			resp, err = tp.CallWithTools(ctx, messages, req.Temperature, req.Tools)
		} else if sp, ok := provider.(StreamingProvider); ok && req.StreamCallback != nil {
			resp, err = sp.Stream(ctx, messages, req.Temperature, req.StreamCallback)
		} else {
			resp, err = provider.Call(ctx, messages, req.Temperature)
		}
		if err != nil {
			return req, err
		}
		req.Response = resp.Content
		req.Usage = &resp.Usage
		return req, nil
	})
}

// Identity for the raw terminal processor.
var rawTerminalID = pipz.NewIdentity("zyn:raw-terminal", "LLM provider raw terminal")

// NewRawTerminal creates a terminal processor that passes messages through as-is.
// Unlike NewTerminal, it does not append a rendered Prompt as a user message.
// It also populates StopReason and ResponseCalls on the request for loop consumers.
// Used by ToolLoopSynapse where messages are built by the loop, not by a Prompt.
func NewRawTerminal(provider Provider) pipz.Chainable[*SynapseRequest] {
	return pipz.Apply(rawTerminalID, func(ctx context.Context, req *SynapseRequest) (*SynapseRequest, error) {
		// Use req.Messages as-is — no prompt appending
		messages := req.Messages

		// Same routing logic as NewTerminal
		var resp *ProviderResponse
		var err error
		if len(req.Tools) > 0 && req.StreamEventCallback != nil {
			tsp, ok := provider.(ToolStreamingProvider)
			if !ok {
				return req, fmt.Errorf("provider %s does not support streaming with tools", provider.Name())
			}
			resp, err = tsp.StreamWithTools(ctx, messages, req.Temperature, req.Tools, req.StreamEventCallback)
		} else if len(req.Tools) > 0 {
			tp, ok := provider.(ToolProvider)
			if !ok {
				return req, fmt.Errorf("provider %s does not support tools", provider.Name())
			}
			resp, err = tp.CallWithTools(ctx, messages, req.Temperature, req.Tools)
		} else if sp, ok := provider.(StreamingProvider); ok && req.StreamCallback != nil {
			resp, err = sp.Stream(ctx, messages, req.Temperature, req.StreamCallback)
		} else {
			resp, err = provider.Call(ctx, messages, req.Temperature)
		}
		if err != nil {
			return req, err
		}

		req.Response = resp.Content
		req.Usage = &resp.Usage
		req.StopReason = resp.StopReason
		req.ResponseCalls = resp.ToolCalls
		return req, nil
	})
}

// GetPipeline returns the internal pipeline for composition.
// This is used by WithFallback to combine pipelines.
func (s *Service[T]) GetPipeline() pipz.Chainable[*SynapseRequest] {
	return s.pipeline
}

// Execute processes a prompt through the pipeline and returns a typed response.
// It creates a SynapseRequest with session context, runs it through the pipeline,
// parses the result, and updates the session with the conversation.
//
// Temperature resolution: if the provided temperature is 0 or TemperatureUnset,
// the service's default temperature is used instead.
//
// The session is only updated after a successful response, ensuring that
// retries from pipz don't corrupt the session state.
func (s *Service[T]) Execute(ctx context.Context, session *Session, prompt *Prompt, temperature float32) (T, error) {
	return s.execute(ctx, session, prompt, temperature, nil)
}

// StreamExecute processes a prompt through the pipeline with streaming support.
// The callback receives text chunks as they arrive from the provider.
// After streaming completes, the full response is parsed, validated, and the session is updated
// identically to Execute.
//
// If the provider does not implement StreamingProvider, the callback is not invoked and
// execution falls back to the standard non-streaming path transparently.
//
// If retry options are applied to the pipeline, the callback may be invoked multiple times
// across retries. Each retry starts streaming from the beginning.
func (s *Service[T]) StreamExecute(ctx context.Context, session *Session, prompt *Prompt, temperature float32, callback StreamCallback) (T, error) {
	return s.execute(ctx, session, prompt, temperature, callback)
}

func (s *Service[T]) execute(ctx context.Context, session *Session, prompt *Prompt, temperature float32, callback StreamCallback) (T, error) {
	var result T

	// Resolve temperature: use default if unset or zero
	if temperature == TemperatureUnset || temperature == 0 {
		temperature = s.defaultTemperature
	}

	// Validate prompt
	if err := prompt.Validate(); err != nil {
		return result, fmt.Errorf("invalid prompt: %w", err)
	}

	// Generate unique request ID
	requestID := uuid.New().String()

	// Get current messages from session
	sessionMessages := session.Messages()

	// Create request with session context
	request := &SynapseRequest{
		Prompt:         prompt,
		Temperature:    temperature,
		StreamCallback: callback,
		Messages:       sessionMessages,
		SessionID:      session.ID(),
		RequestID:      requestID,
		SynapseType:    s.synapseType,
		ProviderName:   s.providerName,
	}

	// Emit request.started hook
	capitan.Info(ctx, RequestStarted,
		RequestIDKey.Field(requestID),
		SynapseTypeKey.Field(s.synapseType),
		ProviderKey.Field(s.providerName),
		PromptTaskKey.Field(prompt.Task),
		InputKey.Field(prompt.Input),
		TemperatureKey.Field(float64(temperature)),
	)

	// Process through pipeline
	processed, err := s.pipeline.Process(ctx, request)
	if err != nil {
		// Emit request.failed hook
		capitan.Error(ctx, RequestFailed,
			RequestIDKey.Field(requestID),
			SynapseTypeKey.Field(s.synapseType),
			ProviderKey.Field(s.providerName),
			PromptTaskKey.Field(prompt.Task),
			ErrorKey.Field(err.Error()),
		)
		return result, err
	}

	// Parse response to type T
	if processed.Response == "" {
		return result, fmt.Errorf("no response from provider")
	}

	if parseErr := json.Unmarshal([]byte(processed.Response), &result); parseErr != nil {
		// Emit response.failed hook
		capitan.Error(ctx, ResponseParseFailed,
			RequestIDKey.Field(requestID),
			SynapseTypeKey.Field(s.synapseType),
			ProviderKey.Field(s.providerName),
			PromptTaskKey.Field(prompt.Task),
			ResponseKey.Field(processed.Response),
			ErrorKey.Field(parseErr.Error()),
			ErrorTypeKey.Field("parse_error"),
		)
		return result, fmt.Errorf("failed to parse response: %w", parseErr)
	}

	// Validate response (T is constrained to Validator)
	if validationErr := result.Validate(); validationErr != nil {
		// Emit response.failed hook
		capitan.Error(ctx, ResponseParseFailed,
			RequestIDKey.Field(requestID),
			SynapseTypeKey.Field(s.synapseType),
			ProviderKey.Field(s.providerName),
			PromptTaskKey.Field(prompt.Task),
			ResponseKey.Field(processed.Response),
			ErrorKey.Field(validationErr.Error()),
			ErrorTypeKey.Field("validation_error"),
		)
		return result, fmt.Errorf("invalid response: %w", validationErr)
	}

	// Success - update session with conversation and usage
	// This is transactional: only happens after successful parsing and validation
	promptStr := prompt.Render()
	session.Append(RoleUser, promptStr)
	session.Append(RoleAssistant, processed.Response)
	session.SetUsage(processed.Usage)

	// Marshal result to JSON for output field
	outputJSON, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		// This should never fail since we already unmarshaled successfully
		outputJSON = []byte("{}")
	}

	// Emit request.completed hook
	capitan.Info(ctx, RequestCompleted,
		RequestIDKey.Field(requestID),
		SynapseTypeKey.Field(s.synapseType),
		ProviderKey.Field(s.providerName),
		PromptTaskKey.Field(prompt.Task),
		InputKey.Field(prompt.Input),
		OutputKey.Field(string(outputJSON)),
		ResponseKey.Field(processed.Response),
	)

	return result, nil
}
