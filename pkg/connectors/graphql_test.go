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

// Mock GraphQL schema for testing
var mockGraphQLSchema = &GraphQLSchema{
	QueryType:    &GraphQLType{Name: "Query"},
	MutationType: &GraphQLType{Name: "Mutation"},
	Types: []GraphQLType{
		{
			Kind: "OBJECT",
			Name: "Query",
			Fields: []GraphQLField{
				{
					Name:        "user",
					Description: "Get a user by ID",
					Args: []GraphQLArg{
						{
							Name:        "id",
							Description: "User ID",
							Type:        GraphQLTypeRef{Kind: "NON_NULL", OfType: &GraphQLTypeRef{Kind: "SCALAR", Name: "ID"}},
						},
					},
					Type: GraphQLTypeRef{Kind: "OBJECT", Name: "User"},
				},
				{
					Name:        "users",
					Description: "List all users",
					Args: []GraphQLArg{
						{
							Name: "limit",
							Type: GraphQLTypeRef{Kind: "SCALAR", Name: "Int"},
						},
						{
							Name: "offset",
							Type: GraphQLTypeRef{Kind: "SCALAR", Name: "Int"},
						},
					},
					Type: GraphQLTypeRef{Kind: "LIST", OfType: &GraphQLTypeRef{Kind: "OBJECT", Name: "User"}},
				},
				{
					Name:        "search",
					Description: "Search users by name",
					Args: []GraphQLArg{
						{
							Name: "query",
							Type: GraphQLTypeRef{Kind: "NON_NULL", OfType: &GraphQLTypeRef{Kind: "SCALAR", Name: "String"}},
						},
						{
							Name: "active",
							Type: GraphQLTypeRef{Kind: "SCALAR", Name: "Boolean"},
						},
					},
					Type: GraphQLTypeRef{Kind: "LIST", OfType: &GraphQLTypeRef{Kind: "OBJECT", Name: "User"}},
				},
			},
		},
		{
			Kind: "OBJECT",
			Name: "Mutation",
			Fields: []GraphQLField{
				{
					Name:        "createUser",
					Description: "Create a new user",
					Args: []GraphQLArg{
						{
							Name: "name",
							Type: GraphQLTypeRef{Kind: "NON_NULL", OfType: &GraphQLTypeRef{Kind: "SCALAR", Name: "String"}},
						},
						{
							Name: "email",
							Type: GraphQLTypeRef{Kind: "NON_NULL", OfType: &GraphQLTypeRef{Kind: "SCALAR", Name: "String"}},
						},
						{
							Name: "age",
							Type: GraphQLTypeRef{Kind: "SCALAR", Name: "Int"},
						},
					},
					Type: GraphQLTypeRef{Kind: "OBJECT", Name: "User"},
				},
				{
					Name:        "deleteUser",
					Description: "Delete a user",
					Args: []GraphQLArg{
						{
							Name: "id",
							Type: GraphQLTypeRef{Kind: "NON_NULL", OfType: &GraphQLTypeRef{Kind: "SCALAR", Name: "ID"}},
						},
					},
					Type: GraphQLTypeRef{Kind: "SCALAR", Name: "Boolean"},
				},
			},
		},
		{
			Kind: "OBJECT",
			Name: "User",
			Fields: []GraphQLField{
				{Name: "id", Type: GraphQLTypeRef{Kind: "SCALAR", Name: "ID"}},
				{Name: "name", Type: GraphQLTypeRef{Kind: "SCALAR", Name: "String"}},
				{Name: "email", Type: GraphQLTypeRef{Kind: "SCALAR", Name: "String"}},
			},
		},
	},
}

func TestGraphQLConnectorFromSchema(t *testing.T) {
	c := NewGraphQLConnectorFromSchema("https://api.example.com/graphql", mockGraphQLSchema)

	if c.Endpoint() != "https://api.example.com/graphql" {
		t.Errorf("Expected endpoint https://api.example.com/graphql, got %s", c.Endpoint())
	}

	if c.Schema() == nil {
		t.Fatal("Expected non-nil schema")
	}
}

func TestGraphQLToolGeneration(t *testing.T) {
	c := NewGraphQLConnectorFromSchema("https://api.example.com/graphql", mockGraphQLSchema)

	tools := c.Tools()
	if len(tools) != 5 { // 3 queries + 2 mutations
		t.Errorf("Expected 5 tools, got %d", len(tools))
	}

	// Check tool names
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Function.Name] = true
	}

	expectedTools := []string{"user", "users", "search", "createUser", "deleteUser"}
	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("Expected tool %s not found", name)
		}
	}
}

func TestGraphQLToolParameters(t *testing.T) {
	c := NewGraphQLConnectorFromSchema("https://api.example.com/graphql", mockGraphQLSchema)

	tools := c.Tools()

	// Find the "user" tool
	var userTool *struct {
		params   map[string]interface{}
		required []string
	}
	for _, tool := range tools {
		if tool.Function.Name == "user" {
			params := tool.Function.Parameters.(map[string]interface{})
			userTool = &struct {
				params   map[string]interface{}
				required []string
			}{
				params: params,
			}
			if req, ok := params["required"].([]string); ok {
				userTool.required = req
			}
			break
		}
	}

	if userTool == nil {
		t.Fatal("user tool not found")
	}

	// Check properties
	props := userTool.params["properties"].(map[string]interface{})
	if _, ok := props["id"]; !ok {
		t.Error("Expected 'id' parameter in user tool")
	}

	// Check required
	required := userTool.params["required"].([]string)
	found := false
	for _, r := range required {
		if r == "id" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'id' to be required")
	}
}

func TestGraphQLToolPrefix(t *testing.T) {
	c := NewGraphQLConnectorFromSchema("https://api.example.com/graphql", mockGraphQLSchema,
		WithGraphQLToolPrefix("gql"))

	tools := c.Tools()

	// All tools should have the prefix
	for _, tool := range tools {
		if tool.Function.Name[:4] != "gql_" {
			t.Errorf("Expected tool name to start with 'gql_', got %s", tool.Function.Name)
		}
	}
}

func TestGraphQLIntrospection(t *testing.T) {
	// Create a mock server that responds to introspection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var req struct {
			Query string `json:"query"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		// Return mock schema
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"__schema": mockGraphQLSchema,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	c, err := NewGraphQLConnector(server.URL)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}

	if c.Schema() == nil {
		t.Fatal("Expected non-nil schema after introspection")
	}
}

func TestGraphQLExecute(t *testing.T) {
	// Create a mock GraphQL server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string `json:"query"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		// Return mock data
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"user": map[string]interface{}{
					"__typename": "User",
					"id":         "123",
					"name":       "John Doe",
					"email":      "john@example.com",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	c := NewGraphQLConnectorFromSchema(server.URL, mockGraphQLSchema)

	ctx := context.Background()
	result, err := c.Execute(ctx, "user", map[string]interface{}{
		"id": "123",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check result
	data, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	user, ok := data["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected user in result")
	}

	if user["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %v", user["name"])
	}
}

func TestGraphQLExecuteMutation(t *testing.T) {
	// Create a mock GraphQL server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query string `json:"query"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		// Verify it's a mutation
		if req.Query[:8] != "mutation" {
			t.Errorf("Expected mutation query, got: %s", req.Query)
		}

		// Return mock data
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"createUser": map[string]interface{}{
					"__typename": "User",
					"id":         "456",
					"name":       "Jane Doe",
					"email":      "jane@example.com",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	c := NewGraphQLConnectorFromSchema(server.URL, mockGraphQLSchema)

	ctx := context.Background()
	result, err := c.Execute(ctx, "createUser", map[string]interface{}{
		"name":  "Jane Doe",
		"email": "jane@example.com",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	data := result.(map[string]interface{})
	user := data["createUser"].(map[string]interface{})
	if user["name"] != "Jane Doe" {
		t.Errorf("Expected name 'Jane Doe', got %v", user["name"])
	}
}

func TestGraphQLAuthentication(t *testing.T) {
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"__schema": mockGraphQLSchema,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	_, err := NewGraphQLConnector(server.URL, WithGraphQLBearerToken("test-token"))
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}

	if receivedAuth != "Bearer test-token" {
		t.Errorf("Expected 'Bearer test-token', got '%s'", receivedAuth)
	}
}

func TestGraphQLAPIKey(t *testing.T) {
	var receivedKey string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("X-API-Key")

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"__schema": mockGraphQLSchema,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	_, err := NewGraphQLConnector(server.URL, WithGraphQLAPIKey("my-api-key", "X-API-Key"))
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}

	if receivedKey != "my-api-key" {
		t.Errorf("Expected 'my-api-key', got '%s'", receivedKey)
	}
}

func TestGraphQLTypeMapping(t *testing.T) {
	c := NewGraphQLConnectorFromSchema("https://api.example.com/graphql", mockGraphQLSchema)

	// Test scalar mapping
	tests := []struct {
		ref      GraphQLTypeRef
		expected string
	}{
		{GraphQLTypeRef{Kind: "SCALAR", Name: "Int"}, "integer"},
		{GraphQLTypeRef{Kind: "SCALAR", Name: "Float"}, "number"},
		{GraphQLTypeRef{Kind: "SCALAR", Name: "Boolean"}, "boolean"},
		{GraphQLTypeRef{Kind: "SCALAR", Name: "String"}, "string"},
		{GraphQLTypeRef{Kind: "SCALAR", Name: "ID"}, "string"},
		{GraphQLTypeRef{Kind: "SCALAR", Name: "CustomScalar"}, "string"},
	}

	for _, tt := range tests {
		schema := c.typeRefToJSONSchema(tt.ref)
		if schema["type"] != tt.expected {
			t.Errorf("For %s, expected type %s, got %s", tt.ref.Name, tt.expected, schema["type"])
		}
	}
}

func TestGraphQLListType(t *testing.T) {
	c := NewGraphQLConnectorFromSchema("https://api.example.com/graphql", mockGraphQLSchema)

	listRef := GraphQLTypeRef{
		Kind:   "LIST",
		OfType: &GraphQLTypeRef{Kind: "SCALAR", Name: "String"},
	}

	schema := c.typeRefToJSONSchema(listRef)
	if schema["type"] != "array" {
		t.Errorf("Expected array type, got %s", schema["type"])
	}

	items := schema["items"].(map[string]interface{})
	if items["type"] != "string" {
		t.Errorf("Expected string items, got %s", items["type"])
	}
}

func TestGraphQLErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"errors": []map[string]interface{}{
				{"message": "Field 'invalid' not found"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	c := NewGraphQLConnectorFromSchema(server.URL, mockGraphQLSchema)

	_, err := c.Execute(context.Background(), "user", map[string]interface{}{"id": "1"})
	if err == nil {
		t.Error("Expected error for GraphQL error response")
	}
}

func TestFormatValue(t *testing.T) {
	c := NewGraphQLConnectorFromSchema("https://api.example.com/graphql", mockGraphQLSchema)

	tests := []struct {
		input    interface{}
		expected string
	}{
		{"hello", `"hello"`},
		{`has "quotes"`, `"has \"quotes\""`},
		{true, "true"},
		{false, "false"},
		{42, "42"},
		{3.14, "3.14"},
		{[]interface{}{"a", "b"}, `["a", "b"]`},
	}

	for _, tt := range tests {
		result := c.formatValue(tt.input)
		if result != tt.expected {
			t.Errorf("formatValue(%v) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}
