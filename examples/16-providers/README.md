# Example 16: LLM Providers

Este ejemplo demuestra cómo usar los diferentes proveedores de LLM con Kairos, verificando la autenticación y mostrando el consumo de tokens.

## Requisitos

Configura las variables de entorno para los proveedores que quieras probar:

```bash
# OpenAI
export OPENAI_API_KEY="sk-..."

# Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."

# Google Gemini
export GOOGLE_API_KEY="..."
# o
export GEMINI_API_KEY="..."

# Alibaba Qwen (DashScope)
export DASHSCOPE_API_KEY="..."
```

## Uso

```bash
cd examples/16-providers

# Probar OpenAI
go run . -provider openai

# Probar Anthropic
go run . -provider anthropic

# Probar Gemini
go run . -provider gemini

# Probar Qwen
go run . -provider qwen

# Probar todos los providers configurados
go run . -provider all
```

## Salida esperada

```
=== OpenAI Provider ===
✓ API key found
Model: gpt-5-mini
Prompt: Responde en una sola línea: ¿Cuál es la capital de España?
---
✓ Response: La capital de España es Madrid.
---
Token Usage:
  Prompt tokens:     18
  Completion tokens: 9
  Total tokens:      27
  Latency:           1.234s
```

## Modelos por defecto

| Provider | Modelo | Características |
|----------|--------|-----------------|
| OpenAI | `gpt-5-mini` | Mejor relación calidad/precio |
| Anthropic | `claude-haiku-4-20250514` | $1/$5 por MTok |
| Gemini | `gemini-3-flash-preview` | Gratis en free tier |
| Qwen | `qwen-turbo` | Más económico de DashScope |

## Usar un modelo diferente

```go
// OpenAI con GPT-5.2
provider := openai.New(openai.WithModel("gpt-5.2"))

// Anthropic con Sonnet
provider := anthropic.New(anthropic.WithModel("claude-sonnet-4.5"))

// Gemini con Pro
provider, _ := gemini.New(ctx, gemini.WithModel("gemini-3-pro-preview"))

// Qwen con Max
provider := qwen.New(apiKey, qwen.WithModel("qwen-max"))
```

## Verificar costes

Consulta la documentación de precios de cada proveedor:
- [OpenAI Pricing](https://platform.openai.com/docs/pricing)
- [Anthropic Pricing](https://www.anthropic.com/pricing)
- [Gemini Pricing](https://ai.google.dev/gemini-api/docs/pricing)
- [DashScope Pricing](https://www.alibabacloud.com/help/en/model-studio/pricing)
