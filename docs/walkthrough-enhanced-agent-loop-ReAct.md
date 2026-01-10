# Walkthrough: Enhanced Agent Loop (ReAct)

I have successfully enhanced the `pkg/agent` package to support a **ReAct (Reasoning + Acting)** loop, enabling agents to use tools and reason in a multi-turn fashion.

## Changes

### 1. Agent Logic (ReAct Loop)

The `Agent.Run` method now implements a multi-turn loop:

- **Thought**: The agent reasons about whether it needs to use a tool.
- **Action**: The agent selects a tool and provides input.
- **Observation**: The agent receives the output from the tool.
- **Final Answer**: The agent provides the final response to the user.

### 2. Tool Support

- Added `WithTools([]core.Tool)` option to inject tools into the agent.
- Added `Tools()` Accessor.

### 3. Testing Infrastructure

- Created `pkg/llm/mock_scripted.go`: A `ScriptedMockProvider` that returns a pre-defined sequence of responses. This is crucial for testing multi-turn conversations where the LLM's output must change based on the conversation state (e.g. suggesting a tool call first, then providing a final answer after seeing the tool output).

## Verification Results

### Automated Tests

Ran `go test ./pkg/agent/...` and all tests passed.

```
=== RUN   TestAgent_ReActLoop
--- PASS: TestAgent_ReActLoop (0.00s)
=== RUN   TestAgent_SingleTurn
--- PASS: TestAgent_SingleTurn (0.00s)
PASS
ok      github.com/jllopis/kairos/pkg/agent     0.388s
```

### Validated Scenario

The `TestAgent_ReActLoop` test verifies the following flow:

1. **User**: "What is 10 + 5?"
2. **Agent (LLM)**: "Thought: need math. Action: Calculator. Action Input: 10 + 5"
3. **System**: Calls `Calculator` tool with "10 + 5" -> Returns "Result from Calculator with 10 + 5"
4. **Agent (LLM)**: "Final Answer: 15"
5. **System**: Returns "15" to the caller.
