# Langchaingo Langfuse 适配器

[English](./README.md) | [中文](./README_ZH.md)

Langfuse 可观测性集成，专为 [langchaingo](https://github.com/tmc/langchaingo)（LangChain 的 Go 实现）设计。

![GitHub stars](https://img.shields.io/github/stars/your-org/langchaingo-langfuse)
![Go Version](https://img.shields.io/github/go-mod/go-version/your-org/langchaingo-langfuse)
![License](https://img.shields.io/github/license/your-org/langchaingo-langfuse)

## 功能特性

- ✅ **Trace 管理** - 创建和管理 Langfuse traces
- ✅ **LLM 调用追踪** - 自动追踪 LLM 调用的完整输入/输出
- ✅ **Token 使用量追踪** - 追踪 prompt/completion/total tokens
- ✅ **Tool 调用追踪** - 追踪工具调用
- ✅ **流式输出支持** - 完整支持流式输出和追踪
- ✅ **错误追踪** - 自动捕获和记录错误
- ✅ **元数据支持** - 为 traces 附加自定义元数据

## 安装

```bash
go get github.com/your-org/langchaingo-langfuse
```

## 快速开始

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

    // 1. 创建适配器
    adapter, err := langfuseAdapter.New(&langfuseAdapter.Config{
        PublicKey:  os.Getenv("LANGFUSE_PUBLIC_KEY"),
        SecretKey:  os.Getenv("LANGFUSE_SECRET_KEY"),
        Host:       os.Getenv("LANGFUSE_HOST"), // 例如: "https://cloud.langfuse.com"
    })
    if err != nil {
        panic(err)
    }

    // 2. 创建 trace
    trace, err := adapter.CreateTrace(ctx, "my-chat-session", map[string]interface{}{
        "user_id": "user123",
    })
    if err != nil {
        panic(err)
    }
    fmt.Printf("Trace ID: %s\n", trace.ID)

    // 3. 创建 LLM
    llm, err := openai.New()
    if err != nil {
        panic(err)
    }

    // 4. 包装 LLM（添加追踪）
    tracedLLM := adapter.WrapLLM(llm, "gpt-4")

    // 5. 生成 - 自动追踪！
    resp, err := tracedLLM.Generate(ctx, []llms.Message{
        llms.HumanChatMessage{Content: "法国的首都是什么？"},
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Choices[0].Content)
}
```

## 高级用法

### 流式输出 + 追踪

```go
tracedLLM := adapter.WrapLLM(llm, "gpt-4")

stream, errChan := tracedLLM.Stream(ctx, []llms.Message{
    llms.HumanChatMessage{Content: "数到5"},
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

### Tool 调用追踪

```go
// 手动记录工具调用
err := adapter.RecordToolCall(ctx, 
    "weather_tool",
    map[string]interface{}{"city": "巴黎"},
    map[string]interface{}{"温度": 22, "天气": "晴"},
    nil, // 无错误
)
```

### 自定义 Generation 名称

```go
// 为更好的组织命名 generation
tracedLLM := adapter.WrapLLMWithName(llm, "gpt-4", "my-chat-completion")
```

## 环境变量

| 变量 | 描述 | 必需 |
|------|------|------|
| `LANGFUSE_PUBLIC_KEY` | Langfuse 公钥 | 是 |
| `LANGFUSE_SECRET_KEY` | Langfuse 私钥 | 是 |
| `LANGFUSE_HOST` | Langfuse 主机（默认: https://cloud.langfuse.com） | 否 |

## API 参考

### `New(config *Config) (*Adapter, error)`

创建新的 Langfuse 适配器。

### `CreateTrace(ctx context.Context, name string, metadata map[string]interface{}) (*TraceInfo, error)`

创建新的 trace。

### `WrapLLM(llm llms.Model, model string) *TracedLLM`

使用 Langfuse 追踪包装 LLM。

### `RecordToolCall(ctx context.Context, name string, args, result map[string]interface{}, err error) error`

将工具调用记录到 Langfuse。

## 与 langchaingo 集成

此适配器与 langchaingo 无缝集成：

- 支持所有 langchaingo LLM 提供商（OpenAI、Anthropic、Ollama 等）
- 兼容 langchaingo chains 和 agents
- 支持所有 CallOptions（temperature、max_tokens 等）

## 许可证

Apache License 2.0 - 详见 [LICENSE](./LICENSE)
