// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package main implements the Kairos CLI.
package main

import (
	"fmt"
	"os"

	"github.com/jllopis/kairos/pkg/errors"
)

// CLIError wraps KairosError with CLI-specific formatting and hints.
type CLIError struct {
	*errors.KairosError
	Hint string
}

// NewCLIError creates a new CLI error.
func NewCLIError(ke *errors.KairosError, hint string) *CLIError {
	return &CLIError{
		KairosError: ke,
		Hint:        hint,
	}
}

// Error returns the formatted error message with hints.
func (e *CLIError) Error() string {
	if e.KairosError == nil {
		return "unknown error"
	}

	msg := e.KairosError.Error()
	if e.Hint != "" {
		msg += "\n  Hint: " + e.Hint
	}
	return msg
}

// PrintError prints the error with appropriate formatting.
func (e *CLIError) PrintError(json bool) {
	if json {
		fmt.Fprintf(os.Stderr, `{"error":{"code":"%s","message":"%s","hint":"%s"}}%s`,
			e.KairosError.Code,
			e.KairosError.Message,
			e.Hint,
			"\n")
		return
	}

	fmt.Fprintf(os.Stderr, "Error [%s]: %s\n", e.KairosError.Code, e.KairosError.Message)
	if e.Hint != "" {
		fmt.Fprintf(os.Stderr, "  Hint: %s\n", e.Hint)
	}
}

// WrapConnectionError wraps a connection error with CLI hints.
func WrapConnectionError(err error, addr string) *CLIError {
	ke := errors.New(errors.CodeInternal, "connection failed", err).
		WithContext("address", addr).
		WithRecoverable(true)
	return NewCLIError(ke, fmt.Sprintf("check if the server is running at %s", addr))
}

// WrapTimeoutError wraps a timeout error with CLI hints.
func WrapTimeoutError(err error, operation string) *CLIError {
	ke := errors.New(errors.CodeTimeout, operation+" timed out", err).
		WithContext("operation", operation).
		WithRecoverable(true)
	return NewCLIError(ke, "try increasing timeout with --timeout flag or check server health")
}

// NewNotFoundError creates a not found error with CLI hints.
func NewNotFoundError(resource, name string) *CLIError {
	ke := errors.New(errors.CodeNotFound, fmt.Sprintf("%s '%s' not found", resource, name), nil).
		WithContext("resource", resource).
		WithContext("name", name).
		WithRecoverable(false)
	return NewCLIError(ke, fmt.Sprintf("check that the %s exists and you have access", resource))
}

// NewInvalidArgumentError creates an invalid argument error with CLI hints.
func NewInvalidArgumentError(arg, reason string) *CLIError {
	ke := errors.New(errors.CodeInvalidInput, fmt.Sprintf("invalid argument: %s", reason), nil).
		WithContext("argument", arg).
		WithContext("reason", reason).
		WithRecoverable(false)
	return NewCLIError(ke, "run 'kairos help' for usage information")
}

// NewConfigError creates a configuration error with CLI hints.
func NewConfigError(err error, configPath string) *CLIError {
	ke := errors.New(errors.CodeInvalidInput, "configuration error", err).
		WithContext("config_path", configPath).
		WithRecoverable(false)

	hint := "check your configuration file syntax"
	if configPath != "" {
		hint = fmt.Sprintf("check %s for syntax errors", configPath)
	}
	return NewCLIError(ke, hint)
}

// NewUnauthorizedError creates an unauthorized error with CLI hints.
func NewUnauthorizedError(reason string) *CLIError {
	ke := errors.New(errors.CodeUnauthorized, "unauthorized", nil).
		WithContext("reason", reason).
		WithRecoverable(false)
	return NewCLIError(ke, "check your credentials or API key")
}

// NewServerError creates a server error with CLI hints.
func NewServerError(err error, operation string) *CLIError {
	ke := errors.New(errors.CodeInternal, "server error", err).
		WithContext("operation", operation).
		WithRecoverable(true)
	return NewCLIError(ke, "this may be a transient error; try again later")
}

// PrintSimpleError prints a simple error message (for non-KairosError cases).
func PrintSimpleError(err error, json bool) {
	if json {
		fmt.Fprintf(os.Stderr, `{"error":{"code":"UNKNOWN","message":"%s"}}%s`, err.Error(), "\n")
		return
	}
	fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
}

// FormatErrorCode returns a user-friendly name for error codes.
func FormatErrorCode(code errors.ErrorCode) string {
	switch code {
	case errors.CodeInternal:
		return "Internal Error"
	case errors.CodeInvalidInput:
		return "Invalid Input"
	case errors.CodeNotFound:
		return "Not Found"
	case errors.CodeUnauthorized:
		return "Unauthorized"
	case errors.CodeTimeout:
		return "Timeout"
	case errors.CodeRateLimit:
		return "Rate Limited"
	case errors.CodeToolFailure:
		return "Tool Failure"
	case errors.CodeLLMError:
		return "LLM Error"
	case errors.CodeMemoryError:
		return "Memory Error"
	case errors.CodeContextLost:
		return "Context Lost"
	default:
		return string(code)
	}
}
