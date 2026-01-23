// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testOpenAPISpec = `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
servers:
  - url: https://api.example.com
paths:
  /users:
    get:
      operationId: listUsers
      summary: List all users
      parameters:
        - name: limit
          in: query
          description: Maximum number of users to return
          required: false
          schema:
            type: integer
            default: 10
      responses:
        "200":
          description: A list of users
    post:
      operationId: createUser
      summary: Create a new user
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                  description: User's name
                email:
                  type: string
                  description: User's email
              required:
                - name
                - email
      responses:
        "201":
          description: User created
  /users/{id}:
    get:
      operationId: getUser
      summary: Get a user by ID
      parameters:
        - name: id
          in: path
          description: User ID
          required: true
          schema:
            type: string
      responses:
        "200":
          description: A user
    delete:
      operationId: deleteUser
      summary: Delete a user
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "204":
          description: User deleted
`

func TestNewFromBytes(t *testing.T) {
	connector, err := NewFromBytes([]byte(testOpenAPISpec))
	if err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	if connector.spec.Info.Title != "Test API" {
		t.Errorf("expected title 'Test API', got %s", connector.spec.Info.Title)
	}

	tools := connector.Tools()
	if len(tools) != 4 {
		t.Errorf("expected 4 tools, got %d", len(tools))
	}
}

func TestToolGeneration(t *testing.T) {
	connector, err := NewFromBytes([]byte(testOpenAPISpec))
	if err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	tools := connector.Tools()

	// Check tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.ToolDefinition().Function.Name] = true
	}

	expectedNames := []string{"listUsers", "createUser", "getUser", "deleteUser"}
	for _, name := range expectedNames {
		if !toolNames[name] {
			t.Errorf("expected tool %s not found", name)
		}
	}
}

func TestToolParameters(t *testing.T) {
	connector, err := NewFromBytes([]byte(testOpenAPISpec))
	if err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	// Find getUser tool
	var getUserTool *struct {
		params map[string]interface{}
	}
	for _, tool := range connector.Tools() {
		def := tool.ToolDefinition()
		if def.Function.Name == "getUser" {
			params, ok := def.Function.Parameters.(map[string]interface{})
			if ok {
				getUserTool = &struct{ params map[string]interface{} }{params}
			}
			break
		}
	}

	if getUserTool == nil {
		t.Fatal("getUser tool not found")
	}

	// Check required parameters
	required, ok := getUserTool.params["required"].([]string)
	if !ok || len(required) == 0 {
		t.Error("expected required parameters for getUser")
	}
}

func TestExecuteWithMockServer(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/users" && r.Method == "GET":
			json.NewEncoder(w).Encode([]map[string]string{
				{"id": "1", "name": "Alice"},
				{"id": "2", "name": "Bob"},
			})
		case r.URL.Path == "/users/1" && r.Method == "GET":
			json.NewEncoder(w).Encode(map[string]string{"id": "1", "name": "Alice"})
		case r.URL.Path == "/users" && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": "3", "name": "Charlie"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	connector, err := NewFromBytes([]byte(testOpenAPISpec), WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	ctx := context.Background()

	// Test listUsers
	result, err := connector.Execute(ctx, "listUsers", map[string]interface{}{"limit": 10})
	if err != nil {
		t.Fatalf("listUsers failed: %v", err)
	}
	if resStr, ok := result.(string); !ok || resStr == "" {
		t.Error("expected non-empty result from listUsers")
	}

	// Test getUser
	result, err = connector.Execute(ctx, "getUser", map[string]interface{}{"id": "1"})
	if err != nil {
		t.Fatalf("getUser failed: %v", err)
	}
	if resStr, ok := result.(string); !ok || resStr == "" {
		t.Error("expected non-empty result from getUser")
	}

	// Test createUser
	result, err = connector.Execute(ctx, "createUser", map[string]interface{}{
		"name":  "Charlie",
		"email": "charlie@example.com",
	})
	if err != nil {
		t.Fatalf("createUser failed: %v", err)
	}
	if resStr, ok := result.(string); !ok || resStr == "" {
		t.Error("expected non-empty result from createUser")
	}
}

func TestAuthenticationOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check API key
		if r.Header.Get("X-API-Key") == "test-key" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "authenticated"}`))
			return
		}
		// Check Bearer token
		if r.Header.Get("Authorization") == "Bearer test-token" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "authenticated"}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	// Test API key auth
	t.Run("APIKey", func(t *testing.T) {
		connector, _ := NewFromBytes([]byte(testOpenAPISpec),
			WithBaseURL(server.URL),
			WithAPIKey("test-key", "X-API-Key"),
		)

		ctx := context.Background()
		result, err := connector.Execute(ctx, "listUsers", nil)
		if err != nil {
			t.Fatalf("request with API key failed: %v", err)
		}
		if resStr, ok := result.(string); !ok || resStr == "" {
			t.Error("expected non-empty result")
		}
	})

	// Test Bearer token auth
	t.Run("Bearer", func(t *testing.T) {
		connector, _ := NewFromBytes([]byte(testOpenAPISpec),
			WithBaseURL(server.URL),
			WithBearerToken("test-token"),
		)

		ctx := context.Background()
		result, err := connector.Execute(ctx, "listUsers", nil)
		if err != nil {
			t.Fatalf("request with Bearer token failed: %v", err)
		}
		if resStr, ok := result.(string); !ok || resStr == "" {
			t.Error("expected non-empty result")
		}
	})
}

func TestExecuteJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"id": "1", "name": "Alice"})
	}))
	defer server.Close()

	connector, _ := NewFromBytes([]byte(testOpenAPISpec), WithBaseURL(server.URL))

	ctx := context.Background()
	result, err := connector.ExecuteJSON(ctx, "getUser", `{"id": "1"}`)
	if err != nil {
		t.Fatalf("ExecuteJSON failed: %v", err)
	}
	if resStr, ok := result.(string); !ok || resStr == "" {
		t.Error("expected non-empty result")
	}
}

func TestJSONSpec(t *testing.T) {
	jsonSpec := `{
		"openapi": "3.0.0",
		"info": {"title": "JSON API", "version": "1.0.0"},
		"paths": {
			"/ping": {
				"get": {
					"operationId": "ping",
					"summary": "Health check"
				}
			}
		}
	}`

	connector, err := NewFromBytes([]byte(jsonSpec))
	if err != nil {
		t.Fatalf("failed to parse JSON spec: %v", err)
	}

	if connector.spec.Info.Title != "JSON API" {
		t.Errorf("expected title 'JSON API', got %s", connector.spec.Info.Title)
	}
}
