# Memoria

Kairos define una abstracción de memoria con almacenamiento vectorial y
embeddings. El objetivo es recuperar contexto relevante antes de responder y
almacenar interacciones después.

## Componentes

La memoria se apoya en dos piezas: un VectorStore y un Embedder. El manager
orquesta la escritura y lectura y puede degradar a modo sin memoria si el
backend no está disponible.

## Integración en el agente

El agente consulta memoria antes de generar respuesta y guarda hechos
relevantes al finalizar la respuesta.

Ejemplo de uso esperado:

- Usuario: "Mi color favorito es azul."
- El agente guarda ese hecho en memoria.
- En la siguiente interacción, el agente lo recupera para responder con
  contexto.

Ejemplo de código (vector store + embedder):

```go
store, err := qdrant.New("localhost:6334")
if err != nil {
  // handle error
}

embedder := ollama.NewEmbedder("http://localhost:11434", "nomic-embed-text")
mem, err := memory.NewVectorMemory(context.Background(), store, embedder, "kairos")
if err != nil {
  // handle error
}
if err := mem.Initialize(context.Background()); err != nil {
  // handle error
}

_ = mem.Store(context.Background(), "Mi color favorito es azul.")
matches, _ := mem.Retrieve(context.Background(), "color favorito")
fmt.Println(matches)
```
