# 03 - Memory Agent

Agente con memoria semántica (vector store) para recordar información entre interacciones.

## Qué aprenderás

- Configurar memoria vectorial con Qdrant
- Usar embeddings de Ollama para búsqueda semántica
- Almacenar y recuperar información contextual
- El agente "recuerda" conversaciones anteriores

## Requisitos

```bash
# Ollama para embeddings
ollama serve
ollama pull nomic-embed-text

# Qdrant para vector store (opcional, usa in-memory si no está)
docker run -p 6333:6333 qdrant/qdrant
```

## Ejecutar

```bash
go run .
```

## Código clave

```go
// Embedder para convertir texto en vectores
embedder := memory.NewOllamaEmbedder("http://localhost:11434", "nomic-embed-text")

// Vector store (Qdrant o in-memory)
store := memory.NewQdrantStore("http://localhost:6333")

// Memoria semántica
mem, _ := memory.NewVectorMemory(ctx, store, embedder, "conversations")

// Agente con memoria
ag, _ := agent.New("memory-agent", llmProvider,
    agent.WithMemory(mem),
)

// El agente ahora puede recordar y buscar información
```

## Cómo funciona

1. Usuario dice algo → se guarda en memoria con embedding
2. Nueva pregunta → búsqueda semántica de contexto relevante
3. Contexto se inyecta al prompt del LLM
4. Respuesta considera información histórica

## Siguiente paso

→ [04-skills-agent](../04-skills-agent/) para definir skills/tools
