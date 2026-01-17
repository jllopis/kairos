// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package connectors provides declarative connectors for converting API specs to tools.
package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jllopis/kairos/pkg/llm"
	"gopkg.in/yaml.v3"
)

// OpenAPISpec represents a parsed OpenAPI 3.x specification.
type OpenAPISpec struct {
	OpenAPI string                `json:"openapi" yaml:"openapi"`
	Info    OpenAPIInfo           `json:"info" yaml:"info"`
	Servers []OpenAPIServer       `json:"servers" yaml:"servers"`
	Paths   map[string]PathItem   `json:"paths" yaml:"paths"`
}

// OpenAPIInfo contains API metadata.
type OpenAPIInfo struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version" yaml:"version"`
}

// OpenAPIServer represents a server endpoint.
type OpenAPIServer struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description" yaml:"description"`
}

// PathItem represents operations on a path.
type PathItem struct {
	Get     *Operation `json:"get" yaml:"get"`
	Post    *Operation `json:"post" yaml:"post"`
	Put     *Operation `json:"put" yaml:"put"`
	Delete  *Operation `json:"delete" yaml:"delete"`
	Patch   *Operation `json:"patch" yaml:"patch"`
}

// Operation represents an API operation.
type Operation struct {
	OperationID string              `json:"operationId" yaml:"operationId"`
	Summary     string              `json:"summary" yaml:"summary"`
	Description string              `json:"description" yaml:"description"`
	Parameters  []Parameter         `json:"parameters" yaml:"parameters"`
	RequestBody *RequestBody        `json:"requestBody" yaml:"requestBody"`
	Responses   map[string]Response `json:"responses" yaml:"responses"`
	Tags        []string            `json:"tags" yaml:"tags"`
}

// Parameter represents an operation parameter.
type Parameter struct {
	Name        string      `json:"name" yaml:"name"`
	In          string      `json:"in" yaml:"in"` // query, path, header, cookie
	Description string      `json:"description" yaml:"description"`
	Required    bool        `json:"required" yaml:"required"`
	Schema      *Schema     `json:"schema" yaml:"schema"`
}

// RequestBody represents a request body.
type RequestBody struct {
	Description string               `json:"description" yaml:"description"`
	Required    bool                 `json:"required" yaml:"required"`
	Content     map[string]MediaType `json:"content" yaml:"content"`
}

// MediaType represents content type details.
type MediaType struct {
	Schema *Schema `json:"schema" yaml:"schema"`
}

// Response represents an API response.
type Response struct {
	Description string               `json:"description" yaml:"description"`
	Content     map[string]MediaType `json:"content" yaml:"content"`
}

// Schema represents a JSON Schema.
type Schema struct {
	Type        string             `json:"type" yaml:"type"`
	Description string             `json:"description" yaml:"description"`
	Properties  map[string]*Schema `json:"properties" yaml:"properties"`
	Items       *Schema            `json:"items" yaml:"items"`
	Required    []string           `json:"required" yaml:"required"`
	Enum        []interface{}      `json:"enum" yaml:"enum"`
	Default     interface{}        `json:"default" yaml:"default"`
	Format      string             `json:"format" yaml:"format"`
}

// OpenAPIConnector converts OpenAPI specs to Kairos tools.
type OpenAPIConnector struct {
	spec       *OpenAPISpec
	baseURL    string
	auth       AuthConfig
	httpClient *http.Client
	tools      []llm.Tool
	handlers   map[string]ToolHandler
}

// AuthConfig defines authentication options.
type AuthConfig struct {
	Type   AuthType
	APIKey string
	Header string // Header name for API key
	Token  string // Bearer token
	User   string // Basic auth user
	Pass   string // Basic auth password
}

// AuthType defines authentication types.
type AuthType int

const (
	AuthNone AuthType = iota
	AuthAPIKey
	AuthBearer
	AuthBasic
)

// ToolHandler executes a tool call against the API.
type ToolHandler func(ctx context.Context, args map[string]interface{}) (string, error)

// Option configures the OpenAPIConnector.
type Option func(*OpenAPIConnector)

// WithBaseURL overrides the base URL from the spec.
func WithBaseURL(url string) Option {
	return func(c *OpenAPIConnector) {
		c.baseURL = url
	}
}

// WithAPIKey sets API key authentication.
func WithAPIKey(key, header string) Option {
	return func(c *OpenAPIConnector) {
		c.auth = AuthConfig{
			Type:   AuthAPIKey,
			APIKey: key,
			Header: header,
		}
	}
}

// WithBearerToken sets Bearer token authentication.
func WithBearerToken(token string) Option {
	return func(c *OpenAPIConnector) {
		c.auth = AuthConfig{
			Type:  AuthBearer,
			Token: token,
		}
	}
}

// WithBasicAuth sets Basic authentication.
func WithBasicAuth(user, pass string) Option {
	return func(c *OpenAPIConnector) {
		c.auth = AuthConfig{
			Type: AuthBasic,
			User: user,
			Pass: pass,
		}
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *OpenAPIConnector) {
		c.httpClient = client
	}
}

// NewFromFile creates an OpenAPIConnector from a file path.
func NewFromFile(path string, opts ...Option) (*OpenAPIConnector, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return NewFromBytes(data, opts...)
}

// NewFromURL creates an OpenAPIConnector from a URL.
func NewFromURL(specURL string, opts ...Option) (*OpenAPIConnector, error) {
	resp, err := http.Get(specURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spec: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return NewFromBytes(data, opts...)
}

// NewFromBytes creates an OpenAPIConnector from raw bytes.
func NewFromBytes(data []byte, opts ...Option) (*OpenAPIConnector, error) {
	var spec OpenAPISpec

	// Try JSON first, then YAML
	if err := json.Unmarshal(data, &spec); err != nil {
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse spec (tried JSON and YAML): %w", err)
		}
	}

	c := &OpenAPIConnector{
		spec:       &spec,
		httpClient: http.DefaultClient,
		handlers:   make(map[string]ToolHandler),
	}

	// Set base URL from spec if available
	if len(spec.Servers) > 0 {
		c.baseURL = spec.Servers[0].URL
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	// Generate tools from spec
	if err := c.generateTools(); err != nil {
		return nil, fmt.Errorf("failed to generate tools: %w", err)
	}

	return c, nil
}

// Tools returns the generated LLM tools.
func (c *OpenAPIConnector) Tools() []llm.Tool {
	return c.tools
}

// Execute runs a tool by name with the given arguments.
func (c *OpenAPIConnector) Execute(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	handler, ok := c.handlers[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return handler(ctx, args)
}

// ExecuteJSON runs a tool with JSON string arguments.
func (c *OpenAPIConnector) ExecuteJSON(ctx context.Context, name, argsJSON string) (string, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid JSON arguments: %w", err)
	}
	return c.Execute(ctx, name, args)
}

// generateTools creates tools from the OpenAPI spec.
func (c *OpenAPIConnector) generateTools() error {
	for path, pathItem := range c.spec.Paths {
		if pathItem.Get != nil {
			c.addOperation(path, "GET", pathItem.Get)
		}
		if pathItem.Post != nil {
			c.addOperation(path, "POST", pathItem.Post)
		}
		if pathItem.Put != nil {
			c.addOperation(path, "PUT", pathItem.Put)
		}
		if pathItem.Delete != nil {
			c.addOperation(path, "DELETE", pathItem.Delete)
		}
		if pathItem.Patch != nil {
			c.addOperation(path, "PATCH", pathItem.Patch)
		}
	}
	return nil
}

// addOperation adds a single operation as a tool.
func (c *OpenAPIConnector) addOperation(path, method string, op *Operation) {
	// Generate tool name
	name := op.OperationID
	if name == "" {
		name = fmt.Sprintf("%s_%s", strings.ToLower(method), strings.ReplaceAll(path, "/", "_"))
		name = strings.Trim(name, "_")
	}

	// Generate description
	desc := op.Summary
	if desc == "" {
		desc = op.Description
	}
	if desc == "" {
		desc = fmt.Sprintf("%s %s", method, path)
	}

	// Build parameters schema
	properties := make(map[string]interface{})
	required := []string{}

	for _, param := range op.Parameters {
		propSchema := c.paramToSchema(param)
		properties[param.Name] = propSchema
		if param.Required {
			required = append(required, param.Name)
		}
	}

	// Handle request body for POST/PUT/PATCH
	if op.RequestBody != nil {
		if content, ok := op.RequestBody.Content["application/json"]; ok && content.Schema != nil {
			if content.Schema.Properties != nil {
				for propName, propSchema := range content.Schema.Properties {
					properties[propName] = c.schemaToMap(propSchema)
				}
				required = append(required, content.Schema.Required...)
			} else {
				// Simple body parameter
				properties["body"] = c.schemaToMap(content.Schema)
				if op.RequestBody.Required {
					required = append(required, "body")
				}
			}
		}
	}

	// Create the tool
	tool := llm.Tool{
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionDef{
			Name:        name,
			Description: desc,
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": properties,
				"required":   required,
			},
		},
	}
	c.tools = append(c.tools, tool)

	// Create the handler
	c.handlers[name] = c.createHandler(path, method, op)
}

// paramToSchema converts a parameter to a JSON Schema map.
func (c *OpenAPIConnector) paramToSchema(param Parameter) map[string]interface{} {
	schema := map[string]interface{}{
		"type":        "string",
		"description": param.Description,
	}

	if param.Schema != nil {
		if param.Schema.Type != "" {
			schema["type"] = param.Schema.Type
		}
		if len(param.Schema.Enum) > 0 {
			schema["enum"] = param.Schema.Enum
		}
		if param.Schema.Default != nil {
			schema["default"] = param.Schema.Default
		}
	}

	return schema
}

// schemaToMap converts a Schema to a JSON Schema map.
func (c *OpenAPIConnector) schemaToMap(schema *Schema) map[string]interface{} {
	if schema == nil {
		return map[string]interface{}{"type": "string"}
	}

	result := map[string]interface{}{}

	if schema.Type != "" {
		result["type"] = schema.Type
	}
	if schema.Description != "" {
		result["description"] = schema.Description
	}
	if len(schema.Enum) > 0 {
		result["enum"] = schema.Enum
	}
	if schema.Default != nil {
		result["default"] = schema.Default
	}

	if schema.Properties != nil {
		props := make(map[string]interface{})
		for name, prop := range schema.Properties {
			props[name] = c.schemaToMap(prop)
		}
		result["properties"] = props
	}

	if schema.Items != nil {
		result["items"] = c.schemaToMap(schema.Items)
	}

	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}

	return result
}

// createHandler creates an HTTP handler for an operation.
func (c *OpenAPIConnector) createHandler(path, method string, op *Operation) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (string, error) {
		// Build URL with path parameters
		finalPath := path
		queryParams := url.Values{}
		headers := http.Header{}
		var bodyData []byte

		for _, param := range op.Parameters {
			value, ok := args[param.Name]
			if !ok {
				continue
			}
			strValue := fmt.Sprintf("%v", value)

			switch param.In {
			case "path":
				finalPath = strings.ReplaceAll(finalPath, "{"+param.Name+"}", strValue)
			case "query":
				queryParams.Set(param.Name, strValue)
			case "header":
				headers.Set(param.Name, strValue)
			}
		}

		// Handle body
		if op.RequestBody != nil {
			// Check if there's a 'body' argument or extract body fields
			if body, ok := args["body"]; ok {
				bodyData, _ = json.Marshal(body)
			} else {
				// Extract body fields from args
				bodyArgs := make(map[string]interface{})
				for key, value := range args {
					isParam := false
					for _, param := range op.Parameters {
						if param.Name == key {
							isParam = true
							break
						}
					}
					if !isParam {
						bodyArgs[key] = value
					}
				}
				if len(bodyArgs) > 0 {
					bodyData, _ = json.Marshal(bodyArgs)
				}
			}
		}

		// Build final URL
		finalURL := c.baseURL + finalPath
		if len(queryParams) > 0 {
			finalURL += "?" + queryParams.Encode()
		}

		// Create request
		var bodyReader io.Reader
		if bodyData != nil {
			bodyReader = strings.NewReader(string(bodyData))
		}

		req, err := http.NewRequestWithContext(ctx, method, finalURL, bodyReader)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		for key, values := range headers {
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}
		if bodyData != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		// Apply authentication
		c.applyAuth(req)

		// Execute request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response: %w", err)
		}

		// Check for errors
		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
		}

		return string(respBody), nil
	}
}

// applyAuth applies authentication to a request.
func (c *OpenAPIConnector) applyAuth(req *http.Request) {
	switch c.auth.Type {
	case AuthAPIKey:
		header := c.auth.Header
		if header == "" {
			header = "X-API-Key"
		}
		req.Header.Set(header, c.auth.APIKey)
	case AuthBearer:
		req.Header.Set("Authorization", "Bearer "+c.auth.Token)
	case AuthBasic:
		req.SetBasicAuth(c.auth.User, c.auth.Pass)
	}
}
