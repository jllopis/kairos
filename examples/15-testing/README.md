# Example 15: Testing Framework

This example demonstrates the Kairos testing framework for writing tests for AI agents.

## Running the Demo

```bash
go run ./examples/15-testing/
```

## Running the Tests

```bash
go test ./examples/15-testing/... -v
```

## Features Demonstrated

### 1. Scenario-Based Testing

Define test scenarios with a fluent API:

```go
scenario := ktesting.NewScenario("basic greeting").
    WithInput("Hello").
    WithTimeout(5 * time.Second).
    ExpectNoError().
    ExpectOutput(ktesting.Contains("assistant"))

result := scenario.Run(t, agent)
result.Assert(t, scenario)
```

### 2. Scripted Provider

Queue responses for multi-turn conversations:

```go
provider := ktesting.NewScenarioProvider().
    AddResponse("Hello! How can I help?").
    AddToolCallResponse(toolCall).
    AddResponse("Task complete!")
```

### 3. Tool Call Builders

Build tool calls and definitions easily:

```go
// Build a tool call
toolCall := ktesting.NewToolCall("search").
    WithID("call_123").
    WithArg("query", "weather").
    Build()

// Build a tool definition
toolDef := ktesting.NewToolDefinition("calculator").
    WithDescription("Perform calculations").
    WithParameter("expression", "string", "Math expression", true).
    Build()
```

### 4. Request Capture

Capture and validate LLM requests:

```go
_, _ = agent.Run(ctx, "Hello")

a.AssertRequest(provider.LastRequest()).
    HasModel("gpt-4").
    HasMessageCount(2).
    HasUserMessage("Hello")
```

### 5. Response Assertions

Validate LLM responses:

```go
a.AssertResponse(resp).
    HasContent("weather").
    HasToolCalls().
    HasToolCallNamed("get_weather")
```

### 6. String Matchers

Various ways to match strings:

```go
ktesting.Contains("world")     // Contains substring
ktesting.Equals("hello")       // Exact match
ktesting.HasPrefix("hello")    // Starts with
ktesting.HasSuffix("world")    // Ends with
ktesting.Regex(`\d+`)          // Regular expression
```

### 7. Event Collection

Track events during agent execution:

```go
collector := ktesting.NewEventCollector()
// Connect to agent...

// Later, verify events
if !collector.HasEvent(core.EventAgentTaskStarted) {
    t.Error("expected task.started event")
}
```

## Full Test Example

See `example_test.go` for complete test implementations.

## Best Practices

1. **Use descriptive scenario names** - Makes test failures easy to understand
2. **Set appropriate timeouts** - Prevent tests from hanging
3. **Capture requests** - Verify the agent sends correct data to the LLM
4. **Test edge cases** - Include error scenarios, timeouts, tool failures
5. **Use setup/teardown** - Clean up resources properly

## See Also

- [Testing Documentation](../../docs/TESTING.md)
- [pkg/testing](../../pkg/testing/) - Full API reference
