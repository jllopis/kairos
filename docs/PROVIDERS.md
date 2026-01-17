# LLM Providers

Kairos proporciona soporte para múltiples proveedores de LLM a través de módulos Go independientes. Esto permite importar solo los proveedores que necesites, evitando dependencias innecesarias.

## Arquitectura

```
kairos/
├── pkg/llm/              # Core - interfaces y tipos
│   └── provider.go       # llm.Provider interface
└── providers/            # Módulos independientes
    ├── openai/           # OpenAI API (GPT-4o, etc.)
    ├── anthropic/        # Anthropic (Claude)
    ├── qwen/             # Alibaba Cloud (Qwen)
    └── gemini/           # Google (Gemini)
```

## Instalación

Instala solo los providers que necesites:

```bash
# Core de Kairos
go get github.com/jllopis/kairos

# Providers individuales
go get github.com/jllopis/kairos/providers/openai
go get github.com/jllopis/kairos/providers/anthropic
go get github.com/jllopis/kairos/providers/qwen
go get github.com/jllopis/kairos/providers/gemini
```

## Provider: OpenAI

Soporta GPT-4o, GPT-4-turbo, GPT-3.5-turbo y modelos compatibles.

```go
import "github.com/jllopis/kairos/providers/openai"

// Usando variable de entorno OPENAI_API_KEY
provider := openai.New()

// Con API key explícita
provider := openai.NewWithAPIKey("sk-...")

// Con opciones
provider := openai.New(
    openai.WithModel("gpt-4-turbo"),
    openai.WithBaseURL("https://api.openai.azure.com/"), // Azure OpenAI
)
```

### Variables de entorno

- `OPENAI_API_KEY`: API key de OpenAI

### Modelos soportados

| Modelo | Descripción |
|--------|-------------|
| `gpt-5-mini` | **Modelo por defecto** - Mejor relación calidad/precio |
| `gpt-5` | GPT-5, alta capacidad |
| `gpt-5.1` | GPT-5.1, mejoras incrementales |
| `gpt-5.1-mini` | GPT-5.1 compacto |
| `gpt-5.2` | GPT-5.2, último modelo estable |
| `o1` | Modelo de razonamiento avanzado |
| `o1-mini` | Modelo de razonamiento compacto |
| `o3` | Razonamiento de última generación |
| `o3-mini` | Razonamiento compacto |
| `gpt-4o` | GPT-4 multimodal (legacy, más caro) |
| `gpt-4-turbo` | GPT-4 optimizado (legacy) |

## Provider: Anthropic

Soporta Claude 4 (Opus, Sonnet, Haiku) y Claude 3.5.

```go
import "github.com/jllopis/kairos/providers/anthropic"

// Usando variable de entorno ANTHROPIC_API_KEY
provider := anthropic.New()

// Con API key explícita
provider := anthropic.NewWithAPIKey("sk-ant-...")

// Con opciones
provider := anthropic.New(
    anthropic.WithModel("claude-opus-4-20250514"),
    anthropic.WithMaxTokens(8192),
)
```

### Variables de entorno

- `ANTHROPIC_API_KEY`: API key de Anthropic

### Modelos soportados

| Modelo | Descripción |
|--------|-------------|
| `claude-haiku-4-20250514` | **Modelo por defecto** - Mejor relación calidad/precio ($1/$5 MTok) |
| `claude-haiku-4.5` | Haiku 4.5, más rápido y económico |
| `claude-sonnet-4.5` | Balanceado, alta calidad |
| `claude-sonnet-4-20250514` | Sonnet 4, balanceado |
| `claude-opus-4.5` | Máxima capacidad, agentes |
| `claude-opus-4-20250514` | Opus 4 (legacy) |

### Notas sobre Anthropic

- Los mensajes de sistema se envían como `system` prompt separado
- Los tool results se envían como mensajes de usuario con bloques `tool_result`
- Max tokens es obligatorio (default: 4096)

## Provider: Qwen

Soporta modelos Qwen de Alibaba Cloud a través de DashScope API (compatible con OpenAI).

```go
import "github.com/jllopis/kairos/providers/qwen"

// API key es requerida
provider := qwen.New("your-dashscope-api-key")

// Con opciones
provider := qwen.New("api-key",
    qwen.WithModel("qwen-max"),
    qwen.WithBaseURL("https://custom.endpoint.com/v1"),
)
```

### Variables de entorno

- Configura tu API key de DashScope en el código o variables de entorno personalizadas

### Modelos soportados

| Modelo | Descripción |
|--------|-------------|
| `qwen-turbo` | **Modelo por defecto** - Más económico |
| `qwen-plus` | Balanceado |
| `qwen-max` | Máxima capacidad |
| `qwen-vl-plus` | Multimodal (visión) |

### Endpoint

Por defecto usa `https://dashscope.aliyuncs.com/compatible-mode/v1`

## Provider: Gemini

Soporta modelos Gemini de Google.

```go
import "github.com/jllopis/kairos/providers/gemini"

// Usando variable de entorno GOOGLE_API_KEY o GEMINI_API_KEY
provider, err := gemini.New(ctx)

// Con API key explícita
provider, err := gemini.NewWithAPIKey(ctx, "your-api-key")

// Con opciones
provider, err := gemini.New(ctx,
    gemini.WithModel("gemini-1.5-pro"),
)
```

### Variables de entorno

- `GOOGLE_API_KEY` o `GEMINI_API_KEY`: API key de Google AI

### Modelos soportados

| Modelo | Descripción |
|--------|-------------|
| `gemini-3-flash-preview` | **Modelo por defecto** - Mejor relación calidad/precio, gratis en free tier |
| `gemini-3-pro-preview` | Alta capacidad, multimodal avanzado |
| `gemini-2.5-flash` | Balanceado, estable |
| `gemini-2.5-pro` | Alta capacidad, estable |
| `gemini-2.0-flash` | Versión anterior (legacy) |
| `gemini-1.5-pro` | Legacy, contexto largo |

### Notas sobre Gemini

- Los mensajes de sistema se envían como `systemInstruction`
- Las function calls usan el formato nativo de Gemini
- El provider no requiere `Close()` explícito

## Ejemplo completo

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/jllopis/kairos/pkg/agent"
    "github.com/jllopis/kairos/pkg/llm"
    "github.com/jllopis/kairos/providers/openai"
)

func main() {
    ctx := context.Background()

    // Crear provider
    provider := openai.New(openai.WithModel("gpt-4o"))

    // Crear agente
    a := agent.New(
        agent.WithProvider(provider),
        agent.WithSystemPrompt("Eres un asistente útil."),
    )

    // Ejecutar
    resp, err := a.Run(ctx, "¿Cuál es la capital de Francia?")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp)
}
```

## Uso con Tools/Function Calling

Todos los providers soportan el sistema unificado de tools:

```go
tool := llm.Tool{
    Type: llm.ToolTypeFunction,
    Function: llm.FunctionDef{
        Name:        "get_weather",
        Description: "Obtiene el clima de una ciudad",
        Parameters: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "city": map[string]interface{}{
                    "type":        "string",
                    "description": "Nombre de la ciudad",
                },
            },
            "required": []string{"city"},
        },
    },
}

// Añadir al request
req := llm.ChatRequest{
    Model:    "gpt-4o",
    Messages: messages,
    Tools:    []llm.Tool{tool},
}

resp, err := provider.Chat(ctx, req)
if len(resp.ToolCalls) > 0 {
    // Procesar tool calls
}
```

## Comparativa de Providers

| Feature | OpenAI | Anthropic | Qwen | Gemini | Ollama |
|---------|--------|-----------|------|--------|--------|
| Function calling | ✅ | ✅ | ✅ | ✅ | ✅ |
| Streaming | ✅ | ✅ | ✅ | ✅ | ✅ |
| Vision/Images | ✅ | ✅ | ✅ | ✅ | ✅ |
| API key env var | `OPENAI_API_KEY` | `ANTHROPIC_API_KEY` | Manual | `GOOGLE_API_KEY` | N/A (local) |
| Custom base URL | ✅ | ✅ | ✅ | ❌ | ✅ |

✅ = Soportado | ❌ = No disponible

## Streaming

Todos los providers implementan `llm.StreamingProvider`:

```go
type StreamingProvider interface {
    Provider
    ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
}
```

### Uso

```go
import "github.com/jllopis/kairos/providers/openai"

provider := openai.New()

// Streaming
stream, err := provider.ChatStream(ctx, llm.ChatRequest{
    Model:    "gpt-5-mini",
    Messages: []llm.Message{{Role: llm.RoleUser, Content: "Hola"}},
})
if err != nil {
    log.Fatal(err)
}

for chunk := range stream {
    if chunk.Error != nil {
        log.Fatal(chunk.Error)
    }
    fmt.Print(chunk.Content) // Imprime en tiempo real
    if chunk.Done {
        fmt.Printf("\nTokens: %d\n", chunk.Usage.TotalTokens)
    }
}
```

Ver `examples/18-streaming/` para un ejemplo completo.

## Crear un Provider personalizado

Implementa la interfaz `llm.Provider`:

```go
type Provider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}
```

Opcionalmente, implementa `llm.StreamingProvider` para soporte de streaming.

Ver [ARCHITECTURE.md](ARCHITECTURE.md#arquitectura-de-llm-providers) para más detalles.
