# Langchaingo Langfuse Adapter

[English](./README.md) | [中文](./README_ZH.md)

Langfuse observability integration for [langchaingo](https://github.com/tmc/langchaingo) - the Go implementation of LangChain.

![GitHub stars](https://img.shields.io/github/stars/1ncludeSteven/langchaingo-langfuse)
![Go Version](https://img.shields.io/github/go-mod/go-version/1ncludeSteven/langchaingo-langfuse)
![License](https://img.shields.io/github/license/1ncludeSteven/langchaingo-langfuse)

## Features

- ✅ **Trace Management** - Create and manage Langfuse traces
- ✅ **LLM Call Tracing** - Automatically trace LLM calls with full input/output
- ✅ **Token Usage Tracking** - Track prompt/completion/total tokens
- ✅ **Tool Call Tracing** - Trace tool invocations
- ✅ **Streaming Support** - Full streaming support with tracing
- ✅ **Error Tracking** - Automatically capture and log errors
- ✅ **Metadata Support** - Attach custom metadata to traces

## Installation

```bash
go get github.com/1ncludeSteven/langchaingo-langfuse
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "os"

    langfuseAdapter "github.com/your-org/langchaingo-langfuse"
    "github.com/tmc/langchaingo/llms"
    "github.com/tmc/langchaingo/llms/openai"
)

func main() {
    ctx := context.Background()

    // 1. Create adapter
    adapter, err := langfuseAdapter.New(&langfuseAdapter.Config{
        PublicKey:  os.Getenv("LANGFUSE_PUBLIC_KEY"),
        SecretKey:  os.Getenv("LANGFUSE_SECRET_KEY"),
        Host:       os.Getenv("LANGFUSE_HOST"), // e.g., "https://cloud.langfuse.com"
    })
    if err != nil {
        panic(err)
    }

    // 2. Create a trace
    trace, err := adapter.CreateTrace(ctx, "my-chat-session", map[string]interface{}{
        "user_id": "user123",
    })
    if err != nil {
        panic(err)
    }
    fmt.Printf("Trace ID: %s\n", trace.ID)

    // 3. Create LLM
    llm, err := openai.New()
    if err != nil {
        panic(err)
    }

    // 4. Wrap LLM with tracing
    tracedLLM := adapter.WrapLLM(llm, "gpt-4")

    // 5. Generate - automatically traced!
    resp, err := tracedLLM.Generate(ctx, []llms.Message{
        llms.HumanChatMessage{Content: "What is the capital of France?"},
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Choices[0].Content)
}
```

## Advanced Usage

### Streaming with Tracing

```go
tracedLLM := adapter.WrapLLM(llm, "gpt-4")

stream, errChan := tracedLLM.Stream(ctx, []llms.Message{
    llms.HumanChatMessage{Content: "Count to 5"},
})

go func() {
    for {
        select {
        case content, ok := <-stream:
            if !ok {
                return
            }
            fmt.Print(content.Choices[0].Content)
        case err := <-errChan:
            if err != nil {
                fmt.Printf("Error: %v\n", err)
            }
        }
    }
}()
```

### Tool Call Tracing

```go
// Record tool call manually
err := adapter.RecordToolCall(ctx, 
    "weather_tool",
    map[string]interface{}{"city": "Paris"},
    map[string]interface{}{"temp": 22, "condition": "sunny"},
    nil, // no error
)
```

### Custom Generation Names

```go
// Name your generations for better organization
tracedLLM := adapter.WrapLLMWithName(llm, "gpt-4", "my-chat-completion")
```

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `LANGFUSE_PUBLIC_KEY` | Langfuse public key | Yes |
| `LANGFUSE_SECRET_KEY` | Langfuse secret key | Yes |
| `LANGFUSE_HOST` | Langfuse host (default: https://cloud.langfuse.com) | No |

## API Reference

### `New(config *Config) (*Adapter, error)`

Creates a new Langfuse adapter.

### `CreateTrace(ctx context.Context, name string, metadata map[string]interface{}) (*TraceInfo, error)`

Creates a new trace.

### `WrapLLM(llm llms.Model, model string) *TracedLLM`

Wraps an LLM with Langfuse tracing.

### `RecordToolCall(ctx context.Context, name string, args, result map[string]interface{}, err error) error`

Records a tool call to Langfuse.

## Integration with langchaingo

This adapter integrates seamlessly with langchaingo:

- Works with all langchaingo LLM providers (OpenAI, Anthropic, Ollama, etc.)
- Compatible with langchaingo chains and agents
- Supports all CallOptions (temperature, max_tokens, etc.)

## License

Apache License 2.0 - see [LICENSE](./LICENSE) for details.
