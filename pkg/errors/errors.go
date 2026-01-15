// SPDX-License-Identifier: Apache-2.0
// Package errors provides typed error handling with rich context for Kairos.
// See docs/ERROR_HANDLING.md for strategy and examples.
package errors

import (
	"encoding/json"
	"fmt"
)

// ErrorCode classifies Kairos errors for monitoring and recovery.
type ErrorCode string

const (
	// CodeInternal indicates an internal system error.
	CodeInternal ErrorCode = "INTERNAL_ERROR"

	// CodeInvalidInput indicates the input was invalid.
	CodeInvalidInput ErrorCode = "INVALID_INPUT"

	// CodeToolFailure indicates a tool execution failed.
	CodeToolFailure ErrorCode = "TOOL_FAILURE"

	// CodeContextLost indicates context was lost (e.g., span ended).
	CodeContextLost ErrorCode = "CONTEXT_LOST"

	// CodeTimeout indicates an operation exceeded its time limit.
	CodeTimeout ErrorCode = "TIMEOUT"

	// CodeRateLimit indicates rate limiting was triggered.
	CodeRateLimit ErrorCode = "RATE_LIMITED"

	// CodeNotFound indicates a resource was not found.
	CodeNotFound ErrorCode = "NOT_FOUND"

	// CodeUnauthorized indicates authorization failed.
	CodeUnauthorized ErrorCode = "UNAUTHORIZED"

	// CodeMemoryError indicates a memory system error.
	CodeMemoryError ErrorCode = "MEMORY_ERROR"

	// CodeLLMError indicates an LLM provider error.
	CodeLLMError ErrorCode = "LLM_ERROR"
)

// KairosError is a typed error with rich context for observability.
// It implements the error interface and can be unwrapped with errors.As().
type KairosError struct {
	Code        ErrorCode
	Message     string
	Err         error
	Context     map[string]interface{}
	Attributes  map[string]string
	Recoverable bool
	StatusCode  int // For A2A/gRPC responses
}

// Error implements the error interface.
func (e *KairosError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap implements errors.Unwrap for error chain traversal.
func (e *KairosError) Unwrap() error {
	return e.Err
}

// MarshalJSON implements json.Marshaler for structured logging.
func (e *KairosError) MarshalJSON() ([]byte, error) {
	type Alias KairosError
	return json.Marshal(&struct {
		Message   string `json:"message"`
		Code      string `json:"code"`
		Err       string `json:"error,omitempty"`
		Recoverable bool `json:"recoverable"`
		*Alias
	}{
		Message:     e.Error(),
		Code:        string(e.Code),
		Err:         fmt.Sprintf("%v", e.Err),
		Recoverable: e.Recoverable,
		Alias:       (*Alias)(e),
	})
}

// New creates a new KairosError with the given code, message, and cause.
func New(code ErrorCode, msg string, cause error) *KairosError {
	return &KairosError{
		Code:       code,
		Message:    msg,
		Err:        cause,
		Context:    make(map[string]interface{}),
		Attributes: make(map[string]string),
		StatusCode: codeToStatusCode(code),
	}
}

// WithContext adds a key-value pair to the error context.
// Returns the error for method chaining.
func (e *KairosError) WithContext(key string, value interface{}) *KairosError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithAttribute adds a string attribute for OTEL traces.
// Returns the error for method chaining.
func (e *KairosError) WithAttribute(key, value string) *KairosError {
	if e.Attributes == nil {
		e.Attributes = make(map[string]string)
	}
	e.Attributes[key] = value
	return e
}

// WithRecoverable sets whether the error can be recovered from.
// Returns the error for method chaining.
func (e *KairosError) WithRecoverable(recoverable bool) *KairosError {
	e.Recoverable = recoverable
	return e
}

// AsKairosError attempts to convert an error to a KairosError.
// Returns the error as KairosError if it is one, or wraps it otherwise.
func AsKairosError(err error) *KairosError {
	if err == nil {
		return nil
	}
	if ke, ok := err.(*KairosError); ok {
		return ke
	}
	// Wrap unknown error as internal
	return New(CodeInternal, "wrapped error", err)
}

// RecoverableString returns "true" or "false" as a string for observability.
func (e *KairosError) RecoverableString() string {
	if e.Recoverable {
		return "true"
	}
	return "false"
}

// codeToStatusCode maps error codes to gRPC/HTTP status codes.
func codeToStatusCode(code ErrorCode) int {
	switch code {
	case CodeNotFound:
		return 404 // NOT_FOUND
	case CodeUnauthorized:
		return 401 // UNAUTHENTICATED
	case CodeInvalidInput:
		return 400 // INVALID_ARGUMENT
	case CodeTimeout:
		return 408 // DEADLINE_EXCEEDED
	case CodeRateLimit:
		return 429 // RESOURCE_EXHAUSTED
	default:
		return 500 // INTERNAL
	}
}
