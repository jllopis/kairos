# Playbook 06 - Memory

Goal: add conversation memory and optional vector memory.

## Why this step?

A travel concierge that forgets your name or your preferences every time you speak is not very useful. Memory allows SkyGuide to maintain context over multiple turns and provide a personalized experience.

## SkyGuide Narrative

SkyGuide is becoming more human-like. It can now remember that you prefer window seats or that you are traveling with a pet. This context is maintained throughout the "session".

## Incremental reuse

- Add `internal/memory` for conversation/vector memory wiring.

## What to implement

- Conversation memory with `memory.NewInMemoryConversation`.
- Use `core.WithSessionID` to keep multi-turn history.
- Configure a truncation strategy (window or token).
- (Optional) vector memory using Qdrant + embedder:
  - `memory.NewVectorMemory`
  - `qdrant.New` + `ollama.NewEmbedder`
- Attach with `agent.WithConversationMemory` and `agent.WithMemory`.
- Reuse provider/config wiring from step 02 via shared helpers.

## Suggested checks

- Multiple turns reuse prior context.
- Conversation history can be printed at the end.

## Manual tests

- "My favorite color is blue."
- "What is my favorite color?"

## Expected behavior

- SkyGuide correctly identifies "blue" in the second response, showing it utilized conversation memory.

## Checklist

- [ ] Session ID stays stable across turns.
- [ ] Truncation strategy is configured.
- [ ] Conversation history is retrievable.

## References

- [03-memory-agent](file:///Users/jllopis/src/kairos/examples/03-memory-agent)
- [20-conversation-memory](file:///Users/jllopis/src/kairos/examples/20-conversation-memory)
- `pkg/memory`
