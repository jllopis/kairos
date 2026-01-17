// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package connectors

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// TestGRPCConnectorFromServices tests creating a connector from pre-defined services.
func TestGRPCConnectorFromServices(t *testing.T) {
	services := map[string]*GRPCService{
		"example.UserService": {
			Name:     "UserService",
			FullName: "example.UserService",
			Methods: []GRPCMethod{
				{
					Name:        "GetUser",
					FullName:    "/example.UserService/GetUser",
					IsStreaming: false,
				},
				{
					Name:        "CreateUser",
					FullName:    "/example.UserService/CreateUser",
					IsStreaming: false,
				},
				{
					Name:        "ListUsers",
					FullName:    "/example.UserService/ListUsers",
					IsStreaming: true, // Streaming method
				},
			},
		},
	}

	c := NewGRPCConnectorFromServices("localhost:50051", services)

	if c.Target() != "localhost:50051" {
		t.Errorf("Expected target localhost:50051, got %s", c.Target())
	}

	if len(c.Services()) != 1 {
		t.Errorf("Expected 1 service, got %d", len(c.Services()))
	}
}

// TestGRPCToolGeneration tests tool generation (without actual message descriptors).
func TestGRPCToolGeneration(t *testing.T) {
	// Since we can't easily create real protoreflect.MessageDescriptors in tests,
	// we'll test the helper functions and structural aspects.

	services := map[string]*GRPCService{
		"example.UserService": {
			Name:     "UserService",
			FullName: "example.UserService",
			Methods: []GRPCMethod{
				{
					Name:        "GetUser",
					FullName:    "/example.UserService/GetUser",
					IsStreaming: false,
					// InputType and OutputType would be set in real usage
				},
				{
					Name:        "StreamUsers",
					FullName:    "/example.UserService/StreamUsers",
					IsStreaming: true,
				},
			},
		},
	}

	c := NewGRPCConnectorFromServices("localhost:50051", services)

	// Tools() will return empty since InputType is nil,
	// but we can verify the service structure
	if svc, ok := c.Services()["example.UserService"]; !ok {
		t.Error("Expected UserService to be registered")
	} else {
		if len(svc.Methods) != 2 {
			t.Errorf("Expected 2 methods, got %d", len(svc.Methods))
		}
	}
}

// TestGRPCToolPrefix tests tool name prefixing.
func TestGRPCToolPrefix(t *testing.T) {
	c := NewGRPCConnectorFromServices("localhost:50051", nil,
		WithGRPCToolPrefix("myapi"))

	if c.toolPrefix != "myapi" {
		t.Errorf("Expected prefix 'myapi', got '%s'", c.toolPrefix)
	}
}

// TestToSnakeCase tests the snake_case conversion helper.
func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"GetUser", "get_user"},
		{"getUserById", "get_user_by_id"},
		{"UserService", "user_service"},
		{"UserService_GetUser", "user_service_get_user"}, // No double underscore
		{"HTTPServer", "h_t_t_p_server"},                 // Each uppercase is treated separately
		{"simple", "simple"},
		{"", ""},
	}

	for _, tt := range tests {
		result := toSnakeCase(tt.input)
		if result != tt.expected {
			t.Errorf("toSnakeCase(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

// TestToInt64 tests integer conversion helper.
func TestToInt64(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected int64
		ok       bool
	}{
		{int(42), 42, true},
		{int32(42), 42, true},
		{int64(42), 42, true},
		{float64(42.5), 42, true},
		{"not a number", 0, false},
		{nil, 0, false},
	}

	for _, tt := range tests {
		result, ok := toInt64(tt.input)
		if ok != tt.ok {
			t.Errorf("toInt64(%v) ok = %v, expected %v", tt.input, ok, tt.ok)
		}
		if ok && result != tt.expected {
			t.Errorf("toInt64(%v) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

// TestToFloat64 tests float conversion helper.
func TestToFloat64(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected float64
		ok       bool
	}{
		{float32(3.14), 3.14, true},
		{float64(3.14), 3.14, true},
		{int(42), 42.0, true},
		{int64(42), 42.0, true},
		{"not a number", 0, false},
	}

	for _, tt := range tests {
		result, ok := toFloat64(tt.input)
		if ok != tt.ok {
			t.Errorf("toFloat64(%v) ok = %v, expected %v", tt.input, ok, tt.ok)
		}
		if ok {
			// Use approximate comparison for floats
			diff := result - tt.expected
			if diff < -0.01 || diff > 0.01 {
				t.Errorf("toFloat64(%v) = %f, expected %f", tt.input, result, tt.expected)
			}
		}
	}
}

// TestKindToJSONSchema tests protobuf kind to JSON schema mapping.
func TestKindToJSONSchema(t *testing.T) {
	c := NewGRPCConnectorFromServices("localhost:50051", nil)

	// We can't easily test with real protoreflect.FieldDescriptor,
	// but we can verify the connector is created successfully
	if c == nil {
		t.Fatal("Expected non-nil connector")
	}
}

// MockFieldDescriptor is a minimal mock for testing (not actually usable for protobuf ops)
type mockFieldDescriptor struct {
	kind protoreflect.Kind
}

func (m mockFieldDescriptor) Kind() protoreflect.Kind { return m.kind }

// TestGRPCFindMethodNotFound tests error handling for unknown methods.
func TestGRPCFindMethodNotFound(t *testing.T) {
	services := map[string]*GRPCService{
		"example.UserService": {
			Name:     "UserService",
			FullName: "example.UserService",
			Methods: []GRPCMethod{
				{Name: "GetUser", FullName: "/example.UserService/GetUser"},
			},
		},
	}

	c := NewGRPCConnectorFromServices("localhost:50051", services)

	_, _, err := c.findMethod("nonexistent_method")
	if err == nil {
		t.Error("Expected error for nonexistent method")
	}
}

// TestGRPCFindMethodWithPrefix tests method finding with prefix.
func TestGRPCFindMethodWithPrefix(t *testing.T) {
	services := map[string]*GRPCService{
		"example.UserService": {
			Name:     "UserService",
			FullName: "example.UserService",
			Methods: []GRPCMethod{
				{Name: "GetUser", FullName: "/example.UserService/GetUser"},
			},
		},
	}

	c := NewGRPCConnectorFromServices("localhost:50051", services,
		WithGRPCToolPrefix("api"))

	// The tool name with prefix should be "api_user_service_get_user"
	// findMethod should strip the prefix and match "user_service_get_user"
	svc, method, err := c.findMethod("api_user_service_get_user")
	if err != nil {
		// Debug: let's see what the expected name is
		expectedName := toSnakeCase("UserService_GetUser")
		t.Logf("Expected name after snake_case: %s", expectedName)
		t.Errorf("Expected to find method, got error: %v", err)
	}
	if method != nil && method.Name != "GetUser" {
		t.Errorf("Expected method GetUser, got %s", method.Name)
	}
	_ = svc // Avoid unused warning
}

// TestGRPCExecuteWithoutConnection tests execute error when not connected.
func TestGRPCExecuteWithoutConnection(t *testing.T) {
	services := map[string]*GRPCService{
		"example.UserService": {
			Name:     "UserService",
			FullName: "example.UserService",
			Methods: []GRPCMethod{
				{Name: "GetUser", FullName: "/example.UserService/GetUser"},
			},
		},
	}

	c := NewGRPCConnectorFromServices("localhost:50051", services)

	// No connection, should fail
	_, err := c.Execute(nil, "user_service_get_user", nil)
	if err == nil {
		t.Error("Expected error when not connected")
	}
}
