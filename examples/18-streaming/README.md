# Example 18: Streaming

Este ejemplo demuestra cómo usar streaming con los proveedores LLM para recibir respuestas en tiempo real, token a token.

## Beneficios del Streaming

- **Mejor UX**: El usuario ve la respuesta mientras se genera
- **Menor latencia percibida**: No hay que esperar a la respuesta completa
- **Cancelación temprana**: Puedes cancelar si la respuesta no es útil

## Interface StreamingProvider

```go
// StreamingProvider extiende Provider con capacidades de streaming
type StreamingProvider interface {
    Provider
    ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
}

// StreamChunk representa un fragmento de la respuesta
type StreamChunk struct {
    Content   string      // Texto delta
    ToolCalls []ToolCall  // Tool calls (acumulados)
    Done      bool        // Indica chunk final
    Usage     *Usage      // Uso de tokens (en chunk final)
    Error     error       // Error durante streaming
}
```

## Uso básico

```go
// Verificar que el provider soporta streaming
streamingProvider, ok := provider.(llm.StreamingProvider)
if !ok {
    log.Fatal("Provider does not support streaming")
}

// Iniciar streaming
chunks, err := streamingProvider.ChatStream(ctx, req)
if err != nil {
    log.Fatal(err)
}

// Procesar chunks
for chunk := range chunks {
    if chunk.Error != nil {
        log.Fatal(chunk.Error)
    }
    
    // Imprimir contenido en tiempo real
    fmt.Print(chunk.Content)
    
    if chunk.Done {
        // Procesar resultado final
        if chunk.Usage != nil {
            fmt.Printf("\nTokens: %d\n", chunk.Usage.TotalTokens)
        }
    }
}
```

## Ejecutar el ejemplo

```bash
cd examples/18-streaming

# Con OpenAI
export OPENAI_API_KEY="sk-..."
go run . -provider openai

# Con Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."
go run . -provider anthropic

# Con Gemini
export GOOGLE_API_KEY="..."
go run . -provider gemini
```

## Salida esperada

```
Provider: openai
Prompt: Escribe un poema corto (4 versos) sobre la programación en Go.
---
Streaming response:

En canales fluye el código veloz,
goroutines danzan sin cesar,
simplicidad es su mayor voz,
en Go aprendemos a programar.

---
✓ Streaming completed
  Total chunks:        42
  Time to first chunk: 234ms
  Total time:          1.2s
  Content length:      156 chars
  Token usage:
    Prompt:     18
    Completion: 45
    Total:      63
```

## Providers con soporte de streaming

| Provider | Streaming | Notas |
|----------|-----------|-------|
| OpenAI | ✅ | SSE nativo |
| Anthropic | ✅ | SSE con eventos tipados |
| Gemini | ✅ | iter.Seq2 |
| Qwen | 🚧 | En desarrollo |
| Ollama | 🚧 | En desarrollo |

## Manejo de errores

```go
for chunk := range chunks {
    // Siempre verificar errores primero
    if chunk.Error != nil {
        if errors.Is(chunk.Error, context.Canceled) {
            fmt.Println("Stream cancelled by user")
        } else {
            fmt.Printf("Stream error: %v\n", chunk.Error)
        }
        break
    }
    
    fmt.Print(chunk.Content)
}
```

## Cancelación

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

chunks, _ := provider.ChatStream(ctx, req)

for chunk := range chunks {
    fmt.Print(chunk.Content)
    
    // Cancelar si encontramos algo
    if strings.Contains(chunk.Content, "stop") {
        cancel()
        break
    }
}
```
