// Package langchaingo provides Langfuse observability integration for langchaingo.
//
// # Usage
//
//	import "your/module/langchaingo-langfuse"
//
//	// Create adapter
//	adapter, err := langchaingo.New(ctx, &langfuse.Config{
//	    PublicKey:  os.Getenv("LANGFUSE_PUBLIC_KEY"),
//	    SecretKey:  os.Getenv("LANGFUSE_SECRET_KEY"),
//	    Host:       os.Getenv("LANGFUSE_HOST"), // e.g., "https://cloud.langfuse.com"
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create traced LLM
//	llm, err := openai.New()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Wrap LLM with tracing
//	tracedLLM := adapter.WrapLLM(llm, "chat-gpt-4")
//
//	// Use as normal
//	resp, err := tracedLLM.GenerateContent(ctx, []llms.MessageContent{
//	    llms.TextParts(llms.ChatMessageTypeHuman, "Hello!"),
//	})
package langchaingo

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/henomis/langfuse-go"
	langfuseModel "github.com/henomis/langfuse-go/model"
	"github.com/tmc/langchaingo/llms"
)

// ========== Types ==========

// Config Langfuse configuration
type Config struct {
	PublicKey     string
	SecretKey     string
	Host          string
	Debug         bool
	ServiceName   string
	FlushInterval time.Duration
}

// Adapter Langfuse adapter for langchaingo
type Adapter struct {
	client      *langfuse.Langfuse
	serviceName string
	traceID    string
	mu         sync.RWMutex
}

// TraceInfo Trace information
type TraceInfo struct {
	ID        string
	Name      string
	StartTime time.Time
	Metadata  map[string]interface{}
}

// GenerationInfo Generation tracking info
type GenerationInfo struct {
	ID        string
	TraceID   string
	ParentID  string
	Name      string
	StartTime time.Time
	Model     string
	Input     map[string]interface{}
	Output    map[string]interface{}
	Usage     *langfuseModel.Usage
	Status    string
	Error     error
}

// ========== Constructor ==========

// New creates a new Langfuse adapter
func New(ctx context.Context, config *Config) (*Adapter, error) {
	if config == nil {
		config = &Config{}
	}

	// Load from environment if not provided
	if config.PublicKey == "" {
		config.PublicKey = os.Getenv("LANGFUSE_PUBLIC_KEY")
	}
	if config.SecretKey == "" {
		config.SecretKey = os.Getenv("LANGFUSE_SECRET_KEY")
	}
	if config.Host == "" {
		config.Host = os.Getenv("LANGFUSE_HOST")
		if config.Host == "" {
			config.Host = "https://cloud.langfuse.com"
		}
	}
	if config.ServiceName == "" {
		config.ServiceName = "langchaingo-app"
	}
	if config.FlushInterval == 0 {
		config.FlushInterval = 500 * time.Millisecond
	}

	// Set environment variables for langfuse-go
	os.Setenv("LANGFUSE_PUBLIC_KEY", config.PublicKey)
	os.Setenv("LANGFUSE_SECRET_KEY", config.SecretKey)
	os.Setenv("LANGFUSE_HOST", config.Host)

	client := langfuse.New(ctx)

	if config.FlushInterval > 0 {
		client = client.WithFlushInterval(config.FlushInterval)
	}

	return &Adapter{
		client:      client,
		serviceName: config.ServiceName,
	}, nil
}

// NewWithDefaults creates adapter with default configuration from environment
func NewWithDefaults(ctx context.Context) (*Adapter, error) {
	return New(ctx, nil)
}

// ========== Trace Management ==========

// CreateTrace creates a new trace
func (a *Adapter) CreateTrace(ctx context.Context, name string, metadata map[string]interface{}) (*TraceInfo, error) {
	traceID := uuid.New().String()
	startTime := time.Now().UTC()

	trace := &langfuseModel.Trace{
		ID:        traceID,
		Name:      name,
		Timestamp: &startTime,
		Metadata:  metadata,
	}

	resp, err := a.client.Trace(trace)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace: %w", err)
	}

	// Update adapter's current trace
	a.mu.Lock()
	a.traceID = resp.ID
	a.mu.Unlock()

	return &TraceInfo{
		ID:        resp.ID,
		Name:      name,
		StartTime: startTime,
		Metadata:  metadata,
	}, nil
}

// GetTraceID returns current trace ID
func (a *Adapter) GetTraceID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.traceID
}

// SetTraceID sets current trace ID
func (a *Adapter) SetTraceID(traceID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.traceID = traceID
}

// ========== LLM Wrapping ==========

// TracedLLM wraps an LLM with Langfuse tracing
type TracedLLM struct {
	llm     llms.Model
	adapter *Adapter
	model   string
	name    string
}

// WrapLLM wraps an LLM with Langfuse tracing
// The model parameter is the model name (e.g., "gpt-4")
func (a *Adapter) WrapLLM(llm llms.Model, model string) *TracedLLM {
	return &TracedLLM{
		llm:     llm,
		adapter: a,
		model:   model,
		name:    "llm_generate",
	}
}

// WrapLLMWithName wraps an LLM with a custom name
func (a *Adapter) WrapLLMWithName(llm llms.Model, model, name string) *TracedLLM {
	return &TracedLLM{
		llm:     llm,
		adapter: a,
		model:   model,
		name:    name,
	}
}

// GenerateContent generates content with tracing (implements llms.Model interface)
func (t *TracedLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error) {
	genInfo := t.startGeneration(ctx, messages, opts)

	resp, err := t.llm.GenerateContent(ctx, messages, opts...)

	t.endGeneration(ctx, genInfo, resp, err)

	return resp, err
}

// Call is a simplified interface (implements llms.Model interface)
func (t *TracedLLM) Call(ctx context.Context, prompt string, opts ...llms.CallOption) (string, error) {
	msg := llms.MessageContent{
		Role:  llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{llms.TextContent{Text: prompt}},
	}

	resp, err := t.GenerateContent(ctx, []llms.MessageContent{msg}, opts...)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return resp.Choices[0].Content, nil
}

// GenerateFromSinglePrompt generates content from a single prompt with tracing
func (t *TracedLLM) GenerateFromSinglePrompt(ctx context.Context, prompt string, opts ...llms.CallOption) (string, error) {
	msg := llms.MessageContent{
		Role:  llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{llms.TextContent{Text: prompt}},
	}

	resp, err := t.GenerateContent(ctx, []llms.MessageContent{msg}, opts...)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return resp.Choices[0].Content, nil
}

// Stream generates content with streaming and tracing
func (t *TracedLLM) Stream(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (<-chan llms.ContentResponse, <-chan error) {
	contentChan := make(chan llms.ContentResponse, 1)
	errChan := make(chan error, 1)

	genInfo := t.startGeneration(ctx, messages, opts)

	// Get original response (non-streaming for now)
	resp, err := t.llm.GenerateContent(ctx, messages, opts...)

	// Handle the response
	go func() {
		defer close(contentChan)
		defer close(errChan)

		if err != nil {
			t.endGeneration(ctx, genInfo, nil, err)
			errChan <- err
			return
		}

		t.endGeneration(ctx, genInfo, resp, nil)
		contentChan <- *resp
	}()

	return contentChan, errChan
}

// ========== Helper Methods ==========

func (t *TracedLLM) startGeneration(ctx context.Context, messages []llms.MessageContent, opts []llms.CallOption) *GenerationInfo {
	genID := uuid.New().String()
	startTime := time.Now().UTC()

	// Get options for model info
	callOpts := &llms.CallOptions{}
	for _, opt := range opts {
		opt(callOpts)
	}

	genInfo := &GenerationInfo{
		ID:        genID,
		TraceID:   t.adapter.GetTraceID(),
		Name:      t.name,
		StartTime: startTime,
		Model:     t.extractModel(callOpts),
		Input: map[string]interface{}{
			"messages": messageContentsToMap(messages),
			"options":  callOptsToMap(callOpts),
		},
	}

	// Create generation in Langfuse (async via observer)
	generation := &langfuseModel.Generation{
		ID:        genID,
		TraceID:   genInfo.TraceID,
		Name:      genInfo.Name,
		StartTime: &startTime,
		Model:     genInfo.Model,
		Input: map[string]interface{}{
			"messages": messageContentsToMap(messages),
		},
		ModelParameters: callOptsToMap(callOpts),
	}

	// Create generation (async)
	t.adapter.client.Generation(generation, nil)

	return genInfo
}

func (t *TracedLLM) endGeneration(ctx context.Context, genInfo *GenerationInfo, resp *llms.ContentResponse, err error) {
	endTime := time.Now().UTC()

	statusMessage := ""
	if err != nil {
		statusMessage = err.Error()
	}

	outputMap := map[string]interface{}{}
	usage := langfuseModel.Usage{}

	if resp != nil && len(resp.Choices) > 0 {
		outputMap["content"] = contentResponseToMap(resp)
	}

	// Create completion generation
	completionGen := &langfuseModel.Generation{
		ID:                  genInfo.ID + "-complete",
		TraceID:             genInfo.TraceID,
		ParentObservationID: genInfo.ID,
		Name:                genInfo.Name + "_complete",
		StartTime:           &genInfo.StartTime,
		EndTime:             &endTime,
		Output:              outputMap,
		Usage:               usage,
		StatusMessage:       statusMessage,
	}

	if err != nil {
		completionGen.Level = langfuseModel.ObservationLevelError
	}

	t.adapter.client.Generation(completionGen, nil)
}

func (t *TracedLLM) extractModel(opts *llms.CallOptions) string {
	if opts == nil {
		return t.model
	}
	if opts.Model != "" {
		return opts.Model
	}
	return t.model
}

// ========== Tool Wrapping ==========

// RecordToolCall records a tool call to Langfuse
func (a *Adapter) RecordToolCall(ctx context.Context, name string, args map[string]interface{}, result interface{}, callErr error) error {
	spanID := uuid.New().String()
	startTime := time.Now().UTC()
	endTime := time.Now().UTC()

	level := langfuseModel.ObservationLevelDefault
	statusMessage := ""
	metadata := map[string]interface{}{}

	if callErr != nil {
		level = langfuseModel.ObservationLevelError
		statusMessage = callErr.Error()
		metadata["error"] = callErr.Error()
	}

	span := &langfuseModel.Generation{
		ID:            spanID,
		TraceID:       a.GetTraceID(),
		Name:          "tool_" + name,
		StartTime:     &startTime,
		EndTime:       &endTime,
		Input:         args,
		Output:        map[string]interface{}{"result": result},
		StatusMessage: statusMessage,
		Level:         level,
		Metadata:      metadata,
	}

	a.client.Generation(span, nil)
	return nil
}

// ========== Utility Functions ==========

func messageContentsToMap(messages []llms.MessageContent) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		parts := make([]map[string]interface{}, len(msg.Parts))
		for j, part := range msg.Parts {
			if tc, ok := part.(llms.TextContent); ok {
				parts[j] = map[string]interface{}{
					"type": "text",
					"text": tc.Text,
				}
			} else if ic, ok := part.(llms.ImageURLContent); ok {
				parts[j] = map[string]interface{}{
					"type": "image_url",
					"url":  ic.URL,
				}
			}
		}
		result[i] = map[string]interface{}{
			"role":  msg.Role,
			"parts": parts,
		}
	}
	return result
}

func callOptsToMap(opts *llms.CallOptions) map[string]interface{} {
	if opts == nil {
		return nil
	}

	result := map[string]interface{}{}

	if opts.Model != "" {
		result["model"] = opts.Model
	}
	if opts.Temperature != 0 {
		result["temperature"] = opts.Temperature
	}
	if opts.MaxTokens != 0 {
		result["max_tokens"] = opts.MaxTokens
	}
	if opts.TopP != 0 {
		result["top_p"] = opts.TopP
	}
	if opts.FrequencyPenalty != 0 {
		result["frequency_penalty"] = opts.FrequencyPenalty
	}
	if opts.PresencePenalty != 0 {
		result["presence_penalty"] = opts.PresencePenalty
	}
	if len(opts.StopWords) > 0 {
		result["stop"] = opts.StopWords
	}

	return result
}

func contentResponseToMap(resp *llms.ContentResponse) []map[string]interface{} {
	if resp == nil {
		return nil
	}

	result := make([]map[string]interface{}, len(resp.Choices))
	for i, choice := range resp.Choices {
		result[i] = map[string]interface{}{
			"content": choice.Content,
		}
		
		if choice.StopReason != "" {
			result[i]["stop_reason"] = choice.StopReason
		}
	}

	return result
}

// ========== Simple LLM Wrapper (for backward compatibility) ==========

// LLM is a simple LLM wrapper that adds Langfuse tracing
type LLM struct {
	llm     llms.Model
	adapter *Adapter
	name    string
}

// NewLLM creates a new traced LLM
func (a *Adapter) NewLLM(llm llms.Model, model string) *LLM {
	return &LLM{
		llm:     llm,
		adapter: a,
		name:    model,
	}
}

// Call implements the llms.Model interface
func (l *LLM) Call(ctx context.Context, prompt string, opts ...llms.CallOption) (string, error) {
	return llms.GenerateFromSinglePrompt(ctx, l.llm, prompt, opts...)
}

// GenerateContent implements the llms.Model interface
func (l *LLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error) {
	traced := l.adapter.WrapLLM(l.llm, l.name)
	return traced.GenerateContent(ctx, messages, opts...)
}
