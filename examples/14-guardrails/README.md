# Example 14: Security Guardrails

This example demonstrates Kairos security guardrails for protecting AI agents from:
- Prompt injection attacks
- PII (Personally Identifiable Information) leakage
- Dangerous/inappropriate content

## Running the Example

```bash
go run ./examples/14-guardrails/
```

## Features Demonstrated

### Prompt Injection Detection
Detects common prompt injection techniques:
- Instruction override attempts ("ignore previous instructions")
- Role manipulation ("you are now...")
- System prompt extraction ("what are your instructions")
- Jailbreak attempts (DAN mode, developer mode)

### Content Filtering
Blocks requests for dangerous content:
- Weapons/explosives instructions
- Malware creation
- Self-harm content
- Illegal activities

### PII Filtering
Masks sensitive data in outputs:
- Email addresses → `[EMAIL]`
- Phone numbers → `[PHONE]`
- SSN → `[SSN]`
- Credit cards → `[CREDIT_CARD]`
- IP addresses → `[IP_ADDRESS]`

## Usage in Your Agent

```go
import "github.com/jllopis/kairos/pkg/guardrails"

// Create guardrails
g := guardrails.New(
    guardrails.WithPromptInjectionDetector(),
    guardrails.WithContentFilter(
        guardrails.ContentCategoryDangerous,
        guardrails.ContentCategoryMalware,
    ),
    guardrails.WithPIIFilter(guardrails.PIIFilterMask),
)

// Check input before sending to LLM
result := g.CheckInput(ctx, userInput)
if result.Blocked {
    return fmt.Errorf("blocked: %s", result.Reason)
}

// Filter output before returning to user
output := g.FilterOutput(ctx, llmResponse)
return output.Content
```

## Configuration Options

### Prompt Injection Detector
```go
guardrails.WithPromptInjectionDetector(
    guardrails.WithStrictMode(true),       // Block on any match
    guardrails.WithInjectionThreshold(0.5), // Confidence threshold
    guardrails.WithInjectionPatterns([]string{`custom pattern`}),
)
```

### PII Filter Modes
```go
// Mask: Replace with placeholder
guardrails.PIIFilterMask   // john@example.com → [EMAIL]

// Redact: Remove entirely
guardrails.PIIFilterRedact // john@example.com → 

// Hash: Replace with hash for correlation
guardrails.PIIFilterHash   // john@example.com → [EMAIL_a1b2c3d4]
```

### Selective PII Types
```go
guardrails.WithPIIFilter(
    guardrails.PIIFilterMask,
    guardrails.WithPIITypes(
        guardrails.PIITypeEmail,
        guardrails.PIITypePhone,
    ),
)
```

## Fail-Safe Behavior

By default, guardrails use fail-closed behavior:
- If context is cancelled, input is blocked
- If an error occurs, input is blocked

To change to fail-open:
```go
g := guardrails.New(
    guardrails.WithFailOpen(true),
    // ... other options
)
```

## Adding Custom Checkers

```go
type MyChecker struct{}

func (c *MyChecker) ID() string { return "my-checker" }

func (c *MyChecker) CheckInput(ctx context.Context, input string) guardrails.CheckResult {
    // Your logic here
    return guardrails.CheckResult{Blocked: false}
}

g.AddInputChecker(&MyChecker{})
```
