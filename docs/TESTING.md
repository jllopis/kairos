# Testing Framework

Kairos provides a testing framework (`pkg/testing`) for writing comprehensive tests for AI agents, flows, and tool integrations.

## Overview

```go
import ktesting "github.com/jllopis/kairos/pkg/testing"

// Define a scenario
scenario := ktesting.NewScenario("greeting test").
    WithInput("Hello").
    ExpectNoError().
    ExpectOutput(ktesting.Contains("Hello"))

// Run against an agent
result := scenario.Run(t, agent)
result.Assert(t, scenario)
```

## Components

### ScenarioProvider

A mock LLM provider with scripted responses for predictable testing:

```go
provider := ktesting.NewScenarioProvider().
    AddResponse("First response").
    AddToolCallResponse(toolCall).
    AddResponse("Final response").
    WithDefaultError(errors.New("no more responses"))
```

**Methods:**
| Method | Description |
|--------|-------------|
| `AddResponse(content)` | Queue a text response |
| `AddToolCallResponse(calls...)` | Queue tool calls |
| `AddErrorResponse(err)` | Queue an error |
| `AddScriptedResponse(resp)` | Queue a full response object |
| `WithDefaultError(err)` | Set error when queue empty |
| `WithChatFunc(fn)` | Custom response handler |
| `Requests()` | Get all captured requests |
| `LastRequest()` | Get most recent request |
| `CallCount()` | Number of calls made |
| `Reset()` | Clear state |

### Scenario

Declarative test case definition:

```go
scenario := ktesting.NewScenario("test name").
    WithDescription("Tests the greeting flow").
    WithInput("Hello, agent!").
    WithContext(ctx).
    WithTimeout(30 * time.Second).
    WithSetup(func() error { /* setup code */ return nil }).
    WithTeardown(func() error { /* cleanup */ return nil }).
    ExpectNoError().
    ExpectOutput(ktesting.Contains("Hello")).
    ExpectToolCall("search").
    ExpectEvent(core.EventAgentTaskStarted).
    ExpectMaxDuration(5 * time.Second)
```

**Expectations:**
| Method | Description |
|--------|-------------|
| `ExpectNoError()` | No error returned |
| `ExpectError(matcher)` | Error matches pattern |
| `ExpectOutput(matcher)` | Output matches pattern |
| `ExpectToolCall(name)` | Tool was called |
| `ExpectNoToolCalls()` | No tools called |
| `ExpectEvent(type)` | Event was emitted |
| `ExpectMinDuration(d)` | Took at least d |
| `ExpectMaxDuration(d)` | Completed within d |

### String Matchers

Flexible string matching:

```go
ktesting.Contains("substring")    // Contains
ktesting.Equals("exact match")    // Exact equality
ktesting.HasPrefix("starts with") // Prefix
ktesting.HasSuffix("ends with")   // Suffix
ktesting.Regex(`pattern`)         // Regular expression
```

### Tool Builders

Construct tool calls and definitions:

```go
// Tool call (from LLM)
call := ktesting.NewToolCall("get_weather").
    WithID("call_123").
    WithArg("city", "London").
    WithArg("unit", "celsius").
    Build()

// Tool definition (for LLM)
def := ktesting.NewToolDefinition("search").
    WithDescription("Search the web").
    WithParameter("query", "string", "Search query", true).
    WithParameter("limit", "integer", "Max results", false).
    Build()
```

### Assertions

Fluent assertion helpers:

```go
a := ktesting.NewAssertions(t)

// Basic assertions
a.AssertEqual(expected, actual, "message")
a.AssertContains(str, substr, "message")
a.AssertNoError(err, "message")
a.AssertError(err, "message")

// Request assertions
a.AssertRequest(req).
    HasModel("gpt-4").
    HasMessageCount(3).
    HasToolCount(2).
    HasSystemMessage("You are").
    HasUserMessage("Hello").
    HasTool("search")

// Response assertions
a.AssertResponse(resp).
    HasContent("result").
    HasToolCalls().
    HasToolCallCount(1).
    HasToolCallNamed("search")

// Scenario result assertions
a.AssertScenarioResult(result).
    Succeeded().
    OutputContains("hello")
```

### Event Collector

Collect and verify events:

```go
collector := ktesting.NewEventCollector()

// In agent setup
agent.WithEventListener(collector.Collect)

// After running
collector.Count()                          // Number of events
collector.Events()                         // All events
collector.EventTypes()                     // Event types
collector.HasEvent(core.EventAgentThinking) // Check for event
collector.Reset()                          // Clear events
```

## Patterns

### Testing Multi-Turn Conversations

```go
func TestMultiTurn(t *testing.T) {
    provider := ktesting.NewScenarioProvider().
        AddResponse("What would you like to know?").
        AddToolCallResponse(
            ktesting.NewToolCall("search").WithArg("q", "weather").Build(),
        ).
        AddResponse("The weather is sunny!")

    agent := NewAgent(provider)

    // Turn 1
    resp1, _ := agent.Run(ctx, "Tell me about the weather")
    assert.Contains(t, resp1, "What")

    // Turn 2 (with tool result)
    resp2, _ := agent.Run(ctx, "weather: sunny, 72°F")
    assert.Contains(t, resp2, "sunny")
}
```

### Testing Error Handling

```go
func TestErrorRecovery(t *testing.T) {
    provider := ktesting.NewScenarioProvider().
        AddErrorResponse(errors.New("rate limited")).
        AddResponse("Recovered!")

    agent := NewAgent(provider)

    scenario := ktesting.NewScenario("error recovery").
        WithInput("test").
        ExpectNoError().
        ExpectOutput(ktesting.Contains("Recovered"))

    result := scenario.Run(t, agent)
    result.Assert(t, scenario)
}
```

### Testing Tool Calls

```go
func TestToolExecution(t *testing.T) {
    searchCall := ktesting.NewToolCall("search").
        WithID("call_1").
        WithArg("query", "Go programming").
        Build()

    provider := ktesting.NewScenarioProvider().
        AddToolCallResponse(searchCall).
        AddResponse("Found 10 results about Go.")

    // Verify tool call was requested
    resp1, _ := provider.Chat(ctx, llm.ChatRequest{})
    
    a := ktesting.NewAssertions(t)
    a.AssertResponse(resp1).
        HasToolCalls().
        HasToolCallNamed("search")

    // Verify arguments
    args := ktesting.AssertToolCallArgs(t, resp1.ToolCalls[0], "search")
    a.AssertEqual("Go programming", args["query"], "query arg")
}
```

### Testing with Setup/Teardown

```go
func TestWithResources(t *testing.T) {
    var tempFile string

    scenario := ktesting.NewScenario("file operations").
        WithSetup(func() error {
            f, err := os.CreateTemp("", "test")
            if err != nil {
                return err
            }
            tempFile = f.Name()
            return f.Close()
        }).
        WithTeardown(func() error {
            return os.Remove(tempFile)
        }).
        WithInput("Process the file").
        ExpectNoError()

    result := scenario.Run(t, agent)
    result.Assert(t, scenario)
}
```

## Best Practices

### 1. Use Descriptive Scenario Names
```go
// Good
ktesting.NewScenario("user asks about weather with invalid city")

// Bad
ktesting.NewScenario("test1")
```

### 2. Set Appropriate Timeouts
```go
// Short for simple tests
scenario.WithTimeout(5 * time.Second)

// Longer for complex flows
scenario.WithTimeout(30 * time.Second)
```

### 3. Capture and Verify Requests
```go
// Always verify the request was formed correctly
a.AssertRequest(provider.LastRequest()).
    HasModel(expectedModel).
    HasSystemMessage(expectedPrompt)
```

### 4. Test Edge Cases
```go
tests := []struct {
    name    string
    input   string
    wantErr bool
}{
    {"empty input", "", true},
    {"very long input", strings.Repeat("x", 10000), false},
    {"special characters", "Hello\n\t\"'", false},
    {"unicode", "こんにちは", false},
}
```

### 5. Use Table-Driven Tests
```go
func TestMatchers(t *testing.T) {
    tests := []struct {
        name    string
        matcher ktesting.StringMatcher
        input   string
        want    bool
    }{
        {"contains match", ktesting.Contains("world"), "hello world", true},
        {"contains miss", ktesting.Contains("foo"), "hello world", false},
    }
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            if got := tc.matcher.Match(tc.input); got != tc.want {
                t.Errorf("Match() = %v, want %v", got, tc.want)
            }
        })
    }
}
```

## See Also

- [Example 15: Testing](../examples/15-testing/) - Working examples
- [pkg/llm/mock.go](../pkg/llm/mock.go) - Basic mock provider
- [pkg/llm/mock_scripted.go](../pkg/llm/mock_scripted.go) - Scripted provider
