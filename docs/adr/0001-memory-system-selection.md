# 1. Selection of Memory System Components

Date: 2026-01-10

## Status

Accepted

## Context

The Kairos framework requires a memory system to enable agents to store and retrieve information relevant to their current context. This "long-term memory" allows agents to maintain continuity across interactions and make more informed decisions based on past data.

We needed to select:

1. A **Vector Database** to store semantic embeddings of text.
2. An **Embedding Provider** to convert text into vector representations.

## Decision

We have decided to use:

1. **Qdrant** as the Vector Database.
2. **Ollama** as the Embedding Provider (specifically with the `nomic-embed-text` model).

## Consequences

### Positive

- **Qdrant**:
  - High performance and scalability (written in Rust).
  - Excellent Go client (`github.com/qdrant/go-client`) with gRPC support.
  - Easy to run locally via Docker for development.
  - Rich filtering capabilities (payload filtering).
- **Ollama**:
  - Allows for a fully local, privacy-preserving stack.
  - Easy integration for users already using Ollama for LLM inference.
  - `nomic-embed-text` provides high-quality embeddings with a manageable dimension size (768).

### Negative

- **Infrastructure Dependency**: Users must run a separate Qdrant service (e.g., via Docker), increasing the "getting started" friction compared to a purely in-memory or embedded solution (like SQLite vss or simple file storage).
- **Resource Usage**: Running both an LLM (Ollama) and a Vector DB (Qdrant) locally can be resource-intensive on smaller machines.

### Risks

- If the user does not have Qdrant running, the memory system will fail to initialize. We have mitigated this by adding a "Graceful Fallback" where the agent logs a warning and continues without memory capabilities.

## Alternatives Considered

- **In-Memory Vector Store**: Simpler for small scale, but doesn't persist across restarts and scales poorly.
- **PostgreSQL (pgvector)**: Good if Postgres is already present, but adds a heavy dependency if it's not.
- **OpenAI Embeddings**: Higher quality, but requires an API key and internet connection, breaking the "local-first" goal of some Kairos users.
