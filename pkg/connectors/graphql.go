// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
)

// GraphQLConnector generates tools from a GraphQL schema via introspection.
type GraphQLConnector struct {
	endpoint   string
	client     *http.Client
	schema     *GraphQLSchema
	headers    map[string]string
	toolPrefix string
}

// GraphQLSchema represents the introspected GraphQL schema.
type GraphQLSchema struct {
	QueryType        *GraphQLType  `json:"queryType"`
	MutationType     *GraphQLType  `json:"mutationType"`
	SubscriptionType *GraphQLType  `json:"subscriptionType"`
	Types            []GraphQLType `json:"types"`
}

// GraphQLType represents a GraphQL type.
type GraphQLType struct {
	Kind        string             `json:"kind"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Fields      []GraphQLField     `json:"fields"`
	InputFields []GraphQLField     `json:"inputFields"`
	EnumValues  []GraphQLEnumValue `json:"enumValues"`
	OfType      *GraphQLType       `json:"ofType"`
}

// GraphQLField represents a field in a GraphQL type.
type GraphQLField struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Args        []GraphQLArg   `json:"args"`
	Type        GraphQLTypeRef `json:"type"`
}

// GraphQLArg represents an argument to a GraphQL field.
type GraphQLArg struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Type         GraphQLTypeRef `json:"type"`
	DefaultValue interface{}    `json:"defaultValue"`
}

// GraphQLTypeRef represents a reference to a GraphQL type.
type GraphQLTypeRef struct {
	Kind   string          `json:"kind"`
	Name   string          `json:"name"`
	OfType *GraphQLTypeRef `json:"ofType"`
}

// GraphQLEnumValue represents an enum value.
type GraphQLEnumValue struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GraphQLOption configures the GraphQLConnector.
type GraphQLOption func(*GraphQLConnector)

// WithGraphQLHeader adds a custom header to requests.
func WithGraphQLHeader(key, value string) GraphQLOption {
	return func(c *GraphQLConnector) {
		c.headers[key] = value
	}
}

// WithGraphQLBearerToken adds a Bearer token for authentication.
func WithGraphQLBearerToken(token string) GraphQLOption {
	return func(c *GraphQLConnector) {
		c.headers["Authorization"] = "Bearer " + token
	}
}

// WithGraphQLAPIKey adds an API key header.
func WithGraphQLAPIKey(key, headerName string) GraphQLOption {
	return func(c *GraphQLConnector) {
		c.headers[headerName] = key
	}
}

// WithGraphQLHTTPClient sets a custom HTTP client.
func WithGraphQLHTTPClient(client *http.Client) GraphQLOption {
	return func(c *GraphQLConnector) {
		c.client = client
	}
}

// WithGraphQLToolPrefix adds a prefix to generated tool names.
func WithGraphQLToolPrefix(prefix string) GraphQLOption {
	return func(c *GraphQLConnector) {
		c.toolPrefix = prefix
	}
}

// NewGraphQLConnector creates a GraphQL connector from an endpoint.
// It performs introspection to discover the schema.
func NewGraphQLConnector(endpoint string, opts ...GraphQLOption) (*GraphQLConnector, error) {
	c := &GraphQLConnector{
		endpoint: endpoint,
		client:   http.DefaultClient,
		headers:  make(map[string]string),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Perform introspection
	if err := c.introspect(); err != nil {
		return nil, fmt.Errorf("introspection failed: %w", err)
	}

	return c, nil
}

// NewGraphQLConnectorFromSchema creates a connector from an already-parsed schema.
// Useful for testing or when you have the schema locally.
func NewGraphQLConnectorFromSchema(endpoint string, schema *GraphQLSchema, opts ...GraphQLOption) *GraphQLConnector {
	c := &GraphQLConnector{
		endpoint: endpoint,
		client:   http.DefaultClient,
		headers:  make(map[string]string),
		schema:   schema,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// introspectionQuery is the standard GraphQL introspection query.
const introspectionQuery = `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      kind
      name
      description
      fields(includeDeprecated: false) {
        name
        description
        args {
          name
          description
          type {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
                ofType {
                  kind
                  name
                }
              }
            }
          }
          defaultValue
        }
        type {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
              }
            }
          }
        }
      }
      inputFields {
        name
        description
        type {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
            }
          }
        }
        defaultValue
      }
      enumValues(includeDeprecated: false) {
        name
        description
      }
    }
  }
}
`

// introspect fetches the GraphQL schema via introspection.
func (c *GraphQLConnector) introspect() error {
	reqBody := map[string]interface{}{
		"query": introspectionQuery,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("introspection returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data struct {
			Schema GraphQLSchema `json:"__schema"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("introspection error: %s", result.Errors[0].Message)
	}

	c.schema = &result.Data.Schema
	return nil
}

// Tools generates core tools from the GraphQL schema queries and mutations.
func (c *GraphQLConnector) Tools() []core.Tool {
	return coreToolsFromDefinitions(c.toolDefinitions(), c)
}

func (c *GraphQLConnector) toolDefinitions() []llm.Tool {
	if c.schema == nil {
		return nil
	}

	var tools []llm.Tool

	// Get the Query and Mutation type definitions
	typeMap := make(map[string]*GraphQLType)
	for i := range c.schema.Types {
		t := &c.schema.Types[i]
		typeMap[t.Name] = t
	}

	// Generate tools for queries
	if c.schema.QueryType != nil {
		if queryType, ok := typeMap[c.schema.QueryType.Name]; ok {
			for _, field := range queryType.Fields {
				if tool := c.fieldToTool(field, "query"); tool != nil {
					tools = append(tools, *tool)
				}
			}
		}
	}

	// Generate tools for mutations
	if c.schema.MutationType != nil {
		if mutationType, ok := typeMap[c.schema.MutationType.Name]; ok {
			for _, field := range mutationType.Fields {
				if tool := c.fieldToTool(field, "mutation"); tool != nil {
					tools = append(tools, *tool)
				}
			}
		}
	}

	return tools
}

// fieldToTool converts a GraphQL field to an llm.Tool.
func (c *GraphQLConnector) fieldToTool(field GraphQLField, opType string) *llm.Tool {
	// Skip internal fields
	if strings.HasPrefix(field.Name, "__") {
		return nil
	}

	name := field.Name
	if c.toolPrefix != "" {
		name = c.toolPrefix + "_" + name
	}

	description := field.Description
	if description == "" {
		description = fmt.Sprintf("GraphQL %s: %s", opType, field.Name)
	}

	// Build parameters from args
	properties := make(map[string]interface{})
	var required []string

	for _, arg := range field.Args {
		paramSchema := c.typeRefToJSONSchema(arg.Type)
		if arg.Description != "" {
			paramSchema["description"] = arg.Description
		}
		properties[arg.Name] = paramSchema

		// Check if required (non-null without default)
		if c.isNonNull(arg.Type) && arg.DefaultValue == nil {
			required = append(required, arg.Name)
		}
	}

	parameters := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		parameters["required"] = required
	}

	return &llm.Tool{
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionDef{
			Name:        name,
			Description: description,
			Parameters:  parameters,
		},
	}
}

// typeRefToJSONSchema converts a GraphQL type reference to JSON Schema.
func (c *GraphQLConnector) typeRefToJSONSchema(ref GraphQLTypeRef) map[string]interface{} {
	switch ref.Kind {
	case "NON_NULL":
		if ref.OfType != nil {
			return c.typeRefToJSONSchema(*ref.OfType)
		}
		return map[string]interface{}{"type": "string"}

	case "LIST":
		itemSchema := map[string]interface{}{"type": "string"}
		if ref.OfType != nil {
			itemSchema = c.typeRefToJSONSchema(*ref.OfType)
		}
		return map[string]interface{}{
			"type":  "array",
			"items": itemSchema,
		}

	case "SCALAR":
		return c.scalarToJSONSchema(ref.Name)

	case "ENUM":
		return map[string]interface{}{
			"type": "string",
			// Could add enum values here if we had them
		}

	case "INPUT_OBJECT":
		return map[string]interface{}{
			"type": "object",
		}

	default:
		return map[string]interface{}{"type": "string"}
	}
}

// scalarToJSONSchema maps GraphQL scalars to JSON Schema types.
func (c *GraphQLConnector) scalarToJSONSchema(name string) map[string]interface{} {
	switch name {
	case "Int":
		return map[string]interface{}{"type": "integer"}
	case "Float":
		return map[string]interface{}{"type": "number"}
	case "Boolean":
		return map[string]interface{}{"type": "boolean"}
	case "ID", "String":
		return map[string]interface{}{"type": "string"}
	default:
		// Custom scalars default to string
		return map[string]interface{}{"type": "string"}
	}
}

// isNonNull checks if a type reference is non-null.
func (c *GraphQLConnector) isNonNull(ref GraphQLTypeRef) bool {
	return ref.Kind == "NON_NULL"
}

// Execute runs a GraphQL query or mutation.
func (c *GraphQLConnector) Execute(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	// Remove prefix if present
	fieldName := toolName
	if c.toolPrefix != "" && strings.HasPrefix(toolName, c.toolPrefix+"_") {
		fieldName = strings.TrimPrefix(toolName, c.toolPrefix+"_")
	}

	// Determine if this is a query or mutation
	opType := c.getOperationType(fieldName)
	if opType == "" {
		return nil, fmt.Errorf("unknown operation: %s", toolName)
	}

	// Build the GraphQL query
	query := c.buildQuery(fieldName, args, opType)

	// Execute
	return c.executeQuery(ctx, query, args)
}

// getOperationType determines if a field is a query or mutation.
func (c *GraphQLConnector) getOperationType(fieldName string) string {
	if c.schema == nil {
		return ""
	}

	typeMap := make(map[string]*GraphQLType)
	for i := range c.schema.Types {
		t := &c.schema.Types[i]
		typeMap[t.Name] = t
	}

	// Check queries
	if c.schema.QueryType != nil {
		if queryType, ok := typeMap[c.schema.QueryType.Name]; ok {
			for _, field := range queryType.Fields {
				if field.Name == fieldName {
					return "query"
				}
			}
		}
	}

	// Check mutations
	if c.schema.MutationType != nil {
		if mutationType, ok := typeMap[c.schema.MutationType.Name]; ok {
			for _, field := range mutationType.Fields {
				if field.Name == fieldName {
					return "mutation"
				}
			}
		}
	}

	return ""
}

// buildQuery constructs a GraphQL query string.
func (c *GraphQLConnector) buildQuery(fieldName string, args map[string]interface{}, opType string) string {
	var argStr string
	if len(args) > 0 {
		var argParts []string
		for k, v := range args {
			argParts = append(argParts, fmt.Sprintf("%s: %s", k, c.formatValue(v)))
		}
		argStr = "(" + strings.Join(argParts, ", ") + ")"
	}

	// Simple query with __typename to ensure we get something back
	return fmt.Sprintf(`%s { %s%s { __typename } }`, opType, fieldName, argStr)
}

// formatValue formats a Go value as a GraphQL value.
func (c *GraphQLConnector) formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		// Escape and quote strings
		escaped := strings.ReplaceAll(val, `"`, `\"`)
		return fmt.Sprintf(`"%s"`, escaped)
	case bool:
		return fmt.Sprintf("%t", val)
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", val)
	case []interface{}:
		var parts []string
		for _, item := range val {
			parts = append(parts, c.formatValue(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]interface{}:
		var parts []string
		for k, item := range val {
			parts = append(parts, fmt.Sprintf("%s: %s", k, c.formatValue(item)))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		// Try JSON for unknown types
		b, _ := json.Marshal(v)
		return string(b)
	}
}

// executeQuery sends a GraphQL query to the endpoint.
func (c *GraphQLConnector) executeQuery(ctx context.Context, query string, variables map[string]interface{}) (interface{}, error) {
	reqBody := map[string]interface{}{
		"query": query,
	}
	if len(variables) > 0 {
		// Note: We're embedding args in query string, not using variables
		// This is simpler but less efficient for complex queries
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GraphQL request failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data   interface{} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", result.Errors[0].Message)
	}

	return result.Data, nil
}

// Schema returns the introspected schema.
func (c *GraphQLConnector) Schema() *GraphQLSchema {
	return c.schema
}

// Endpoint returns the GraphQL endpoint URL.
func (c *GraphQLConnector) Endpoint() string {
	return c.endpoint
}
