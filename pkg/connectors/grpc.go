// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package connectors provides declarative connectors for converting API specs to tools.
package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// GRPCConnector generates tools from a gRPC server via reflection.
type GRPCConnector struct {
	target     string
	conn       *grpc.ClientConn
	services   map[string]*GRPCService
	opts       []grpc.DialOption
	toolPrefix string
}

// GRPCService represents a gRPC service discovered via reflection.
type GRPCService struct {
	Name        string
	FullName    string
	Methods     []GRPCMethod
	FileDesc    protoreflect.FileDescriptor
	ServiceDesc protoreflect.ServiceDescriptor
}

// GRPCMethod represents a method in a gRPC service.
type GRPCMethod struct {
	Name        string
	FullName    string
	InputType   protoreflect.MessageDescriptor
	OutputType  protoreflect.MessageDescriptor
	IsStreaming bool
}

// GRPCOption configures the GRPCConnector.
type GRPCOption func(*GRPCConnector)

// WithGRPCDialOptions adds custom gRPC dial options.
func WithGRPCDialOptions(opts ...grpc.DialOption) GRPCOption {
	return func(c *GRPCConnector) {
		c.opts = append(c.opts, opts...)
	}
}

// WithGRPCInsecure uses insecure connection (for development).
func WithGRPCInsecure() GRPCOption {
	return func(c *GRPCConnector) {
		c.opts = append(c.opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
}

// WithGRPCToolPrefix adds a prefix to generated tool names.
func WithGRPCToolPrefix(prefix string) GRPCOption {
	return func(c *GRPCConnector) {
		c.toolPrefix = prefix
	}
}

// NewGRPCConnector creates a gRPC connector using server reflection.
func NewGRPCConnector(target string, opts ...GRPCOption) (*GRPCConnector, error) {
	c := &GRPCConnector{
		target:   target,
		services: make(map[string]*GRPCService),
		opts:     []grpc.DialOption{},
	}

	for _, opt := range opts {
		opt(c)
	}

	// Default to insecure if no transport credentials set
	if len(c.opts) == 0 {
		c.opts = append(c.opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Connect to the server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, target, c.opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", target, err)
	}
	c.conn = conn

	// Perform reflection to discover services
	if err := c.reflect(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("reflection failed: %w", err)
	}

	return c, nil
}

// NewGRPCConnectorFromServices creates a connector from pre-defined services.
// Useful for testing or when reflection is not available.
func NewGRPCConnectorFromServices(target string, services map[string]*GRPCService, opts ...GRPCOption) *GRPCConnector {
	c := &GRPCConnector{
		target:   target,
		services: services,
		opts:     []grpc.DialOption{},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// reflect discovers services using gRPC server reflection.
func (c *GRPCConnector) reflect(ctx context.Context) error {
	client := grpc_reflection_v1alpha.NewServerReflectionClient(c.conn)

	stream, err := client.ServerReflectionInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to create reflection stream: %w", err)
	}
	defer stream.CloseSend()

	// List all services
	if err := stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_ListServices{
			ListServices: "",
		},
	}); err != nil {
		return fmt.Errorf("failed to send list services request: %w", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive list services response: %w", err)
	}

	listResp := resp.GetListServicesResponse()
	if listResp == nil {
		return fmt.Errorf("unexpected response type")
	}

	// For each service, get its file descriptor
	for _, svc := range listResp.GetService() {
		serviceName := svc.GetName()

		// Skip reflection service itself
		if strings.HasPrefix(serviceName, "grpc.reflection") {
			continue
		}

		// Get file descriptor for this service
		if err := stream.Send(&grpc_reflection_v1alpha.ServerReflectionRequest{
			MessageRequest: &grpc_reflection_v1alpha.ServerReflectionRequest_FileContainingSymbol{
				FileContainingSymbol: serviceName,
			},
		}); err != nil {
			continue
		}

		resp, err := stream.Recv()
		if err != nil {
			continue
		}

		fdResp := resp.GetFileDescriptorResponse()
		if fdResp == nil {
			continue
		}

		// Parse file descriptors
		if err := c.parseFileDescriptors(serviceName, fdResp.GetFileDescriptorProto()); err != nil {
			continue
		}
	}

	return nil
}

// parseFileDescriptors parses the file descriptor protos and extracts service info.
func (c *GRPCConnector) parseFileDescriptors(serviceName string, fdProtos [][]byte) error {
	// Build a file descriptor set
	var files []*descriptorpb.FileDescriptorProto
	for _, fdBytes := range fdProtos {
		var fd descriptorpb.FileDescriptorProto
		if err := proto.Unmarshal(fdBytes, &fd); err != nil {
			return err
		}
		files = append(files, &fd)
	}

	// Create a resolver for dependencies
	resolver := &protoregistry.Files{}

	// Register all files
	for _, fdProto := range files {
		fd, err := protodesc.NewFile(fdProto, resolver)
		if err != nil {
			// Try to continue with other files
			continue
		}
		resolver.RegisterFile(fd)
	}

	// Find our service
	desc, err := resolver.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return err
	}

	serviceDesc, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return fmt.Errorf("not a service descriptor")
	}

	// Extract methods
	svc := &GRPCService{
		Name:        string(serviceDesc.Name()),
		FullName:    serviceName,
		ServiceDesc: serviceDesc,
	}

	methods := serviceDesc.Methods()
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)
		svc.Methods = append(svc.Methods, GRPCMethod{
			Name:        string(method.Name()),
			FullName:    fmt.Sprintf("/%s/%s", serviceName, method.Name()),
			InputType:   method.Input(),
			OutputType:  method.Output(),
			IsStreaming: method.IsStreamingClient() || method.IsStreamingServer(),
		})
	}

	c.services[serviceName] = svc
	return nil
}

// Tools generates core tools from discovered gRPC services.
func (c *GRPCConnector) Tools() []core.Tool {
	return coreToolsFromDefinitions(c.toolDefinitions(), c)
}

func (c *GRPCConnector) toolDefinitions() []llm.Tool {
	var tools []llm.Tool

	for _, svc := range c.services {
		for _, method := range svc.Methods {
			// Skip streaming methods (not supported for simple tool calls)
			if method.IsStreaming {
				continue
			}

			tool := c.methodToTool(svc, method)
			if tool != nil {
				tools = append(tools, *tool)
			}
		}
	}

	return tools
}

// methodToTool converts a gRPC method to an llm.Tool.
func (c *GRPCConnector) methodToTool(svc *GRPCService, method GRPCMethod) *llm.Tool {
	name := fmt.Sprintf("%s_%s", svc.Name, method.Name)
	if c.toolPrefix != "" {
		name = c.toolPrefix + "_" + name
	}

	// Convert to snake_case for consistency
	name = toSnakeCase(name)

	description := fmt.Sprintf("gRPC method %s.%s", svc.Name, method.Name)

	// Build parameters from input message type
	parameters := c.messageToJSONSchema(method.InputType)

	return &llm.Tool{
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionDef{
			Name:        name,
			Description: description,
			Parameters:  parameters,
		},
	}
}

// messageToJSONSchema converts a protobuf message descriptor to JSON Schema.
func (c *GRPCConnector) messageToJSONSchema(msg protoreflect.MessageDescriptor) map[string]interface{} {
	properties := make(map[string]interface{})
	var required []string

	fields := msg.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		fieldName := string(field.JSONName())
		if fieldName == "" {
			fieldName = string(field.Name())
		}

		schema := c.fieldToJSONSchema(field)
		properties[fieldName] = schema

		// In proto3, all fields are optional by default
		// We only mark as required if it has presence (proto2 or explicit optional)
		if field.Cardinality() == protoreflect.Required {
			required = append(required, fieldName)
		}
	}

	result := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		result["required"] = required
	}

	return result
}

// fieldToJSONSchema converts a protobuf field to JSON Schema.
func (c *GRPCConnector) fieldToJSONSchema(field protoreflect.FieldDescriptor) map[string]interface{} {
	// Handle repeated fields (arrays)
	if field.IsList() {
		itemSchema := c.kindToJSONSchema(field)
		return map[string]interface{}{
			"type":  "array",
			"items": itemSchema,
		}
	}

	// Handle maps
	if field.IsMap() {
		valueField := field.MapValue()
		return map[string]interface{}{
			"type":                 "object",
			"additionalProperties": c.kindToJSONSchema(valueField),
		}
	}

	return c.kindToJSONSchema(field)
}

// kindToJSONSchema maps protobuf kinds to JSON Schema types.
func (c *GRPCConnector) kindToJSONSchema(field protoreflect.FieldDescriptor) map[string]interface{} {
	switch field.Kind() {
	case protoreflect.BoolKind:
		return map[string]interface{}{"type": "boolean"}

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return map[string]interface{}{"type": "integer", "format": "int32"}

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return map[string]interface{}{"type": "integer", "format": "int64"}

	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return map[string]interface{}{"type": "integer", "format": "uint32"}

	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return map[string]interface{}{"type": "integer", "format": "uint64"}

	case protoreflect.FloatKind:
		return map[string]interface{}{"type": "number", "format": "float"}

	case protoreflect.DoubleKind:
		return map[string]interface{}{"type": "number", "format": "double"}

	case protoreflect.StringKind:
		return map[string]interface{}{"type": "string"}

	case protoreflect.BytesKind:
		return map[string]interface{}{"type": "string", "format": "byte"}

	case protoreflect.EnumKind:
		enum := field.Enum()
		values := enum.Values()
		enumValues := make([]string, 0, values.Len())
		for i := 0; i < values.Len(); i++ {
			enumValues = append(enumValues, string(values.Get(i).Name()))
		}
		return map[string]interface{}{
			"type": "string",
			"enum": enumValues,
		}

	case protoreflect.MessageKind:
		// Nested message - recursively convert
		return c.messageToJSONSchema(field.Message())

	default:
		return map[string]interface{}{"type": "string"}
	}
}

// Execute calls a gRPC method with the given arguments.
func (c *GRPCConnector) Execute(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	// Find the service and method
	_, method, err := c.findMethod(toolName)
	if err != nil {
		return nil, err
	}

	if c.conn == nil {
		return nil, fmt.Errorf("not connected to gRPC server")
	}

	// Create the input message dynamically
	inputMsg := dynamicpb.NewMessage(method.InputType)

	// Populate fields from args
	if err := c.populateMessage(inputMsg, args); err != nil {
		return nil, fmt.Errorf("failed to populate input message: %w", err)
	}

	// Make the gRPC call
	outputMsg := dynamicpb.NewMessage(method.OutputType)
	err = c.conn.Invoke(ctx, method.FullName, inputMsg, outputMsg)
	if err != nil {
		return nil, fmt.Errorf("gRPC call failed: %w", err)
	}

	// Convert output to map
	return c.messageToMap(outputMsg), nil
}

// findMethod finds the service and method for a tool name.
func (c *GRPCConnector) findMethod(toolName string) (*GRPCService, *GRPCMethod, error) {
	// Remove prefix if present
	name := toolName
	if c.toolPrefix != "" && strings.HasPrefix(toolName, c.toolPrefix+"_") {
		name = strings.TrimPrefix(toolName, c.toolPrefix+"_")
	}

	// Tool names are in format "service_method" (snake_case)
	for _, svc := range c.services {
		for i := range svc.Methods {
			method := &svc.Methods[i]
			expectedName := toSnakeCase(fmt.Sprintf("%s_%s", svc.Name, method.Name))
			if name == expectedName {
				return svc, method, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("method not found: %s", toolName)
}

// populateMessage populates a dynamic message from a map of arguments.
func (c *GRPCConnector) populateMessage(msg *dynamicpb.Message, args map[string]interface{}) error {
	if args == nil {
		return nil
	}

	desc := msg.Descriptor()
	fields := desc.Fields()

	for key, value := range args {
		// Find the field by JSON name or regular name
		var field protoreflect.FieldDescriptor
		for i := 0; i < fields.Len(); i++ {
			f := fields.Get(i)
			if string(f.JSONName()) == key || string(f.Name()) == key {
				field = f
				break
			}
		}

		if field == nil {
			continue // Unknown field, skip
		}

		protoValue, err := c.toProtoValue(field, value)
		if err != nil {
			return fmt.Errorf("field %s: %w", key, err)
		}

		msg.Set(field, protoValue)
	}

	return nil
}

// toProtoValue converts a Go value to a protoreflect.Value.
func (c *GRPCConnector) toProtoValue(field protoreflect.FieldDescriptor, value interface{}) (protoreflect.Value, error) {
	if value == nil {
		return protoreflect.Value{}, nil
	}

	switch field.Kind() {
	case protoreflect.BoolKind:
		if b, ok := value.(bool); ok {
			return protoreflect.ValueOfBool(b), nil
		}

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		if n, ok := toInt64(value); ok {
			return protoreflect.ValueOfInt32(int32(n)), nil
		}

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		if n, ok := toInt64(value); ok {
			return protoreflect.ValueOfInt64(n), nil
		}

	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		if n, ok := toUint64(value); ok {
			return protoreflect.ValueOfUint32(uint32(n)), nil
		}

	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		if n, ok := toUint64(value); ok {
			return protoreflect.ValueOfUint64(n), nil
		}

	case protoreflect.FloatKind:
		if f, ok := toFloat64(value); ok {
			return protoreflect.ValueOfFloat32(float32(f)), nil
		}

	case protoreflect.DoubleKind:
		if f, ok := toFloat64(value); ok {
			return protoreflect.ValueOfFloat64(f), nil
		}

	case protoreflect.StringKind:
		if s, ok := value.(string); ok {
			return protoreflect.ValueOfString(s), nil
		}

	case protoreflect.BytesKind:
		if s, ok := value.(string); ok {
			return protoreflect.ValueOfBytes([]byte(s)), nil
		}

	case protoreflect.EnumKind:
		if s, ok := value.(string); ok {
			enum := field.Enum()
			v := enum.Values().ByName(protoreflect.Name(s))
			if v != nil {
				return protoreflect.ValueOfEnum(v.Number()), nil
			}
		}

	case protoreflect.MessageKind:
		if m, ok := value.(map[string]interface{}); ok {
			nestedMsg := dynamicpb.NewMessage(field.Message())
			if err := c.populateMessage(nestedMsg, m); err != nil {
				return protoreflect.Value{}, err
			}
			return protoreflect.ValueOfMessage(nestedMsg), nil
		}
	}

	return protoreflect.Value{}, fmt.Errorf("cannot convert %T to %s", value, field.Kind())
}

// messageToMap converts a dynamic message to a map.
func (c *GRPCConnector) messageToMap(msg *dynamicpb.Message) map[string]interface{} {
	result := make(map[string]interface{})

	msg.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		key := string(field.JSONName())
		if key == "" {
			key = string(field.Name())
		}
		result[key] = c.protoValueToGo(field, value)
		return true
	})

	return result
}

// protoValueToGo converts a protoreflect.Value to a Go value.
func (c *GRPCConnector) protoValueToGo(field protoreflect.FieldDescriptor, value protoreflect.Value) interface{} {
	if field.IsList() {
		list := value.List()
		result := make([]interface{}, list.Len())
		for i := 0; i < list.Len(); i++ {
			result[i] = c.scalarToGo(field, list.Get(i))
		}
		return result
	}

	if field.IsMap() {
		m := value.Map()
		result := make(map[string]interface{})
		m.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
			keyStr := fmt.Sprintf("%v", k.Interface())
			result[keyStr] = c.scalarToGo(field.MapValue(), v)
			return true
		})
		return result
	}

	return c.scalarToGo(field, value)
}

// scalarToGo converts a scalar protoreflect.Value to a Go value.
func (c *GRPCConnector) scalarToGo(field protoreflect.FieldDescriptor, value protoreflect.Value) interface{} {
	switch field.Kind() {
	case protoreflect.MessageKind:
		if msg, ok := value.Interface().(*dynamicpb.Message); ok {
			return c.messageToMap(msg)
		}
		return value.Interface()
	case protoreflect.EnumKind:
		return string(field.Enum().Values().ByNumber(value.Enum()).Name())
	default:
		return value.Interface()
	}
}

// Close closes the gRPC connection.
func (c *GRPCConnector) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Services returns the discovered services.
func (c *GRPCConnector) Services() map[string]*GRPCService {
	return c.services
}

// Target returns the gRPC target address.
func (c *GRPCConnector) Target() string {
	return c.target
}

// Helper functions

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				// Don't add underscore if previous char is already underscore
				prev := s[i-1]
				if prev != '_' {
					result.WriteRune('_')
				}
			}
			result.WriteRune(r + 32) // lowercase
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	}
	return 0, false
}

func toUint64(v interface{}) (uint64, bool) {
	switch n := v.(type) {
	case uint:
		return uint64(n), true
	case uint32:
		return uint64(n), true
	case uint64:
		return n, true
	case float64:
		return uint64(n), true
	case json.Number:
		i, err := n.Int64()
		return uint64(i), err == nil
	}
	return 0, false
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float32:
		return float64(n), true
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}
