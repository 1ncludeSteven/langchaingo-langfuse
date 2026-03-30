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
	"testing"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// Example demonstrates basic usage
func Example() {
	ctx := context.Background()

	// Create adapter
	adapter, err := New(ctx, &Config{
		PublicKey:    os.Getenv("LANGFUSE_PUBLIC_KEY"),
		SecretKey:    os.Getenv("LANGFUSE_SECRET_KEY"),
		Host:         os.Getenv("LANGFUSE_HOST"),
		ServiceName:  "my-chat-app",
	})
	if err != nil {
		fmt.Printf("Error creating adapter: %v\n", err)
		return
	}

	// Create a trace
	trace, err := adapter.CreateTrace(ctx, "chat-session", map[string]interface{}{
		"user_id": "user123",
		"session": "abc-456",
	})
	if err != nil {
		fmt.Printf("Error creating trace: %v\n", err)
		return
	}
	fmt.Printf("Created trace: %s\n", trace.ID)

	// Create OpenAI LLM
	llm, err := openai.New()
	if err != nil {
		fmt.Printf("Error creating LLM: %v\n", err)
		return
	}

	// Wrap LLM with tracing
	tracedLLM := adapter.WrapLLM(llm, "gpt-4")

	// Generate with tracing
	resp, err := tracedLLM.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, "What is the capital of France?"),
	})
	if err != nil {
		fmt.Printf("Error generating: %v\n", err)
		return
	}

	fmt.Printf("Response: %s\n", resp.Choices[0].Content)
}

// Example_withTools demonstrates tool usage
func Example_withTools() {
	ctx := context.Background()

	adapter, _ := New(ctx, &Config{
		PublicKey: os.Getenv("LANGFUSE_PUBLIC_KEY"),
		SecretKey: os.Getenv("LANGFUSE_SECRET_KEY"),
		Host:      os.Getenv("LANGFUSE_HOST"),
	})

	// Create trace
	adapter.CreateTrace(ctx, "tool-session", nil)

	// Simulate tool call
	err := adapter.RecordToolCall(ctx, "weather_tool", 
		map[string]interface{}{"city": "Paris"},
		map[string]interface{}{"temp": 20, "condition": "sunny"},
		nil,
	)
	if err != nil {
		fmt.Printf("Error recording tool call: %v\n", err)
	}
}

// TestNewAdapter tests adapter creation
func TestNewAdapter(t *testing.T) {
	// This test requires LANGFUSE_PUBLIC_KEY and LANGFUSE_SECRET_KEY
	publicKey := os.Getenv("LANGFUSE_PUBLIC_KEY")
	secretKey := os.Getenv("LANGFUSE_SECRET_KEY")

	if publicKey == "" || secretKey == "" {
		t.Skip("Skipping test: LANGFUSE credentials not set")
	}

	adapter, err := New(context.Background(), &Config{
		PublicKey: publicKey,
		SecretKey: secretKey,
		Host:      os.Getenv("LANGFUSE_HOST"),
	})

	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	if adapter == nil {
		t.Fatal("Adapter is nil")
	}
}

// TestCreateTrace tests trace creation
func TestCreateTrace(t *testing.T) {
	publicKey := os.Getenv("LANGFUSE_PUBLIC_KEY")
	secretKey := os.Getenv("LANGFUSE_SECRET_KEY")

	if publicKey == "" || secretKey == "" {
		t.Skip("Skipping test: LANGFUSE credentials not set")
	}

	adapter, _ := New(context.Background(), &Config{
		PublicKey: publicKey,
		SecretKey: secretKey,
	})

	ctx := context.Background()
	trace, err := adapter.CreateTrace(ctx, "test-trace", map[string]interface{}{
		"test": "true",
	})

	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	if trace == nil || trace.ID == "" {
		t.Fatal("Trace ID is empty")
	}

	t.Logf("Created trace: %s", trace.ID)
}

// TestGenerateWithTracing tests LLM generation with tracing
func TestGenerateWithTracing(t *testing.T) {
	publicKey := os.Getenv("LANGFUSE_PUBLIC_KEY")
	secretKey := os.Getenv("LANGFUSE_SECRET_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")

	if publicKey == "" || secretKey == "" || openaiKey == "" {
		t.Skip("Skipping test: credentials not set")
	}

	adapter, _ := New(context.Background(), &Config{
		PublicKey: publicKey,
		SecretKey: secretKey,
	})

	llm, _ := openai.New()
	tracedLLM := adapter.WrapLLM(llm, "gpt-4o-mini")

	ctx := context.Background()

	// Create trace
	adapter.CreateTrace(ctx, "test-generation", nil)

	// Generate
	resp, err := tracedLLM.GenerateFromSinglePrompt(ctx, "Say 'hello' in one word")
	if err != nil {
		t.Fatalf("Generation failed: %v", err)
	}

	if resp == "" {
		t.Fatal("Empty response")
	}

	t.Logf("Generated: %s", resp)
}

// BenchmarkLLMGenerate benchmarks LLM generation with tracing
func BenchmarkLLMGenerate(b *testing.B) {
	publicKey := os.Getenv("LANGFUSE_PUBLIC_KEY")
	secretKey := os.Getenv("LANGFUSE_SECRET_KEY")

	if publicKey == "" || secretKey == "" {
		b.Skip("Skipping benchmark: LANGFUSE credentials not set")
	}

	adapter, _ := New(context.Background(), &Config{
		PublicKey: publicKey,
		SecretKey: secretKey,
	})

	llm, _ := openai.New()
	tracedLLM := adapter.WrapLLM(llm, "gpt-4o-mini")

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.CreateTrace(ctx, "bench-trace", nil)
		tracedLLM.GenerateFromSinglePrompt(ctx, "Say hello")
	}
}
