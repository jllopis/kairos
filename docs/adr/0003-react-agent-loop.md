# 3. Agent Loop Pattern and Tooling

Date: 2026-01-10

## Status

Accepted

## Context

The Kairos agent framework requires a mechanism to handle complex, multi-step tasks that involve reasoning and interacting with external environments. A simple "Input -> LLM -> Output" (single-turn) model is insufficient for agents that need to:

1. Determine if external information is needed.
2. Execute tools to retrieve that information.
3. Evaluate the tool output.
4. Decide efficiently on the next step or final answer.

Furthermore, testing such multi-turn interactions is challenging. Standard mock providers that return a static string are inadequate for testing loops where the agent expects different responses at different stages (e.g., first a tool call request, then a final answer after the tool result).

## Decision

### 1. ReAct Pattern (Reasoning + Acting)

We will implement the **ReAct** pattern for the main Agent Loop.

- **Why**: ReAct combines Chain-of-Thought reasoning with Action execution. It allows the model to "think" out loud about what it needs to do ("Thought"), select an "Action", and then process the "Observation".
- **Structure**: The loop consists of `Thought -> Action -> Observation -> Thought -> ... -> Final Answer`.
- **Constraint**: We enforce a `maxIterations` limit to prevent infinite loops if the model gets stuck.

### 2. Tool Abstraction

Tools will be injected into the Agent via a standard `WithTools` option.

- **Interface**: The `core.Tool` interface is used, ensuring compatibility with the MCP (Model Context Protocol) implementation decided in ADR-0002.
- **Discovery**: Tools are presented to the LLM in the system prompt with their names and descriptions.

### 3. Testing Strategy: ScriptedMockProvider

To reliably test the ReAct loop without flakiness or dependence on real LLM APIs, we introduce a `ScriptedMockProvider`.

- **Mechanism**: A FIFO queue of responses.
- **Usage**:
  - **Turn 1**: Mock returns a "Thought + Action" response.
  - **Turn 2**: Mock returns a "Final Answer" response (simulating the LLM reacting to the tool output).
- **Benefit**: This ensures deterministic testing of the loop logic (parsing, tool execution, history management) completely decoupled from model intelligence.

## Consequences

### Positive

- **capabilities**: Agents can now solve problems requiring external data or side effects.
- **Observability**: The "Thought" process provides visibility into the agent's decision-making.
- **Testability**: The `ScriptedMockProvider` pattern establishes a standard for testing complex agent flows in Kairos.

### Negative

- **Latency**: Multiple turns increase the time to final answer and token costs.
- **Complexity**: The prompt engineering required to strictly enforce the "Thought/Action" format can be fragile with smaller or less capable models.
