package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jllopis/kairos/demoKairos/internal/demo"
	"github.com/jllopis/kairos/pkg/a2a/agentcard"
	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/memory"
	"github.com/jllopis/kairos/pkg/memory/ollama"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/grpc"
)

type spreadsheetExecutor struct {
	agent *agent.Agent
	store *demo.SpreadsheetStore
}

func (e *spreadsheetExecutor) Run(ctx context.Context, message *a2av1.Message) (any, []*a2av1.Artifact, error) {
	if e.agent == nil {
		return nil, nil, fmt.Errorf("spreadsheet agent not configured")
	}
	if e.store == nil {
		return nil, nil, fmt.Errorf("spreadsheet store not configured")
	}
	if data := server.ExtractData(message); data != nil {
		spec := decodeQuerySpec(data)
		if spec.Type == "" {
			return nil, nil, fmt.Errorf("spreadsheet query missing type")
		}
		log.Printf("spreadsheet agent query type=%s quarter=%s month=%s limit=%d", spec.Type, spec.Quarter, spec.Month, spec.Limit)
		result, err := e.store.Query(spec)
		if err != nil {
			return nil, nil, err
		}
		log.Printf("spreadsheet agent rows=%d", len(result.Rows))
		payload := map[string]interface{}{
			"headers": toInterfaceSlice(result.Headers),
			"rows":    toInterfaceRows(result.Rows),
			"meta":    result.Meta,
		}
		resp := demo.NewDataMessage(a2av1.Role_ROLE_AGENT, payload, message.ContextId, message.TaskId)
		return resp, nil, nil
	}
	input := server.ExtractText(message)
	if input == "" {
		input = "Genera la consulta adecuada usando query_spreadsheet y devuelve JSON con headers, rows, meta."
	}
	log.Printf("spreadsheet agent fallback to LLM")
	for attempt := 0; attempt < 2; attempt++ {
		output, err := e.agent.Run(ctx, input)
		if err != nil {
			return nil, nil, err
		}
		payload := extractJSON(output)
		if payload != nil {
			resp := demo.NewDataMessage(a2av1.Role_ROLE_AGENT, payload, message.ContextId, message.TaskId)
			return resp, nil, nil
		}
		input = "Devuelve SOLO JSON valido con headers, rows y meta. No incluyas texto adicional."
	}
	return nil, nil, fmt.Errorf("failed to parse JSON output from spreadsheet agent")
}

func main() {
	var (
		addr       = flag.String("addr", ":9032", "gRPC listen address")
		dataDir    = flag.String("data", "", "CSV data directory")
		configPath = flag.String("config", "", "Config file path")
		mcpAddr    = flag.String("mcp-addr", "127.0.0.1:9042", "MCP streamable HTTP address")
		cardAddr   = flag.String("card-addr", "127.0.0.1:9142", "AgentCard HTTP address")
		embedModel = flag.String("embed-model", "nomic-embed-text", "Ollama embed model")
		qdrantURL  = flag.String("qdrant", "localhost:6334", "Qdrant gRPC address")
		memColl    = flag.String("memory-collection", "kairos_demo_sheet_memory", "Qdrant memory collection")
		verbose    = flag.Bool("verbose", false, "Enable verbose telemetry output")
	)
	flag.Parse()

	if *dataDir == "" {
		cwd, _ := os.Getwd()
		*dataDir = filepath.Join(cwd, "data")
	}

	cfg, err := demo.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	shutdown, err := demo.InitTelemetry("spreadsheet-agent", cfg, *verbose)
	if err != nil {
		log.Fatalf("telemetry: %v", err)
	}
	defer func() {
		_ = shutdown(context.Background())
	}()

	llmProvider, err := demo.NewLLMProvider(cfg)
	if err != nil {
		log.Fatalf("llm: %v", err)
	}

	store, err := demo.NewQdrantStore(demo.QdrantConfig{URL: *qdrantURL, Collection: *memColl})
	if err != nil {
		log.Fatalf("qdrant: %v", err)
	}
	embedder := ollama.NewEmbedder(os.Getenv("OLLAMA_URL"), *embedModel)
	memStore, err := memory.NewVectorMemory(context.Background(), store, embedder, *memColl)
	if err != nil {
		log.Fatalf("memory: %v", err)
	}
	if err := memStore.Initialize(context.Background()); err != nil {
		log.Fatalf("memory init: %v", err)
	}

	storeCSV, err := demo.LoadSpreadsheetStore(*dataDir)
	if err != nil {
		log.Fatalf("load data: %v", err)
	}

	mcpServer, err := demo.StartMCPServer("spreadsheet-mcp", "0.1.0", *mcpAddr)
	if err != nil {
		log.Fatalf("mcp server: %v", err)
	}
	mcpServer.RegisterTool("query_spreadsheet", "Consulta segura sobre CSV", func(ctx context.Context, args map[string]interface{}) (*mcpgo.CallToolResult, error) {
		spec := decodeQuerySpec(args)
		result, err := storeCSV.Query(spec)
		if err != nil {
			return nil, err
		}
		return &mcpgo.CallToolResult{StructuredContent: map[string]interface{}{"headers": result.Headers, "rows": result.Rows, "meta": result.Meta}}, nil
	})
	mcpServer.RegisterTool("list_sheets", "Lista hojas disponibles", func(ctx context.Context, args map[string]interface{}) (*mcpgo.CallToolResult, error) {
		return &mcpgo.CallToolResult{StructuredContent: map[string]interface{}{"sheets": storeCSV.Sheets()}}, nil
	})
	mcpServer.RegisterTool("get_schema", "Devuelve columnas de una hoja", func(ctx context.Context, args map[string]interface{}) (*mcpgo.CallToolResult, error) {
		sheet, _ := args["sheet"].(string)
		headers, err := storeCSV.Schema(sheet)
		if err != nil {
			return nil, err
		}
		return &mcpgo.CallToolResult{StructuredContent: map[string]interface{}{"headers": headers, "sheet": sheet}}, nil
	})

	mcpClient, err := demo.NewMCPClient(mcpServer.BaseURL())
	if err != nil {
		log.Fatalf("mcp client: %v", err)
	}

	role := "Eres un agente de hojas de calculo. Usa SOLO la herramienta query_spreadsheet para responder. Devuelve un JSON con headers, rows y meta. No incluyas texto adicional."
	roleManifest, err := demo.LoadRoleManifest("role-spreadsheet.yaml")
	if err != nil {
		log.Printf("role manifest: %v", err)
	}
	agentOpts := []agent.Option{
		agent.WithRole(role),
		agent.WithRoleManifest(roleManifest),
		agent.WithModel(cfg.LLM.Model),
		agent.WithMCPClients(mcpClient),
		agent.WithMemory(memStore),
	}
	if len(cfg.MCP.Servers) > 0 {
		agentOpts = append(agentOpts, agent.WithMCPServerConfigs(cfg.MCP.Servers))
	}

	sheetAgent, err := agent.New("spreadsheet-agent", llmProvider, agentOpts...)
	if err != nil {
		log.Fatalf("agent: %v", err)
	}

	exec := &spreadsheetExecutor{agent: sheetAgent, store: storeCSV}
	card := agentcard.Build(agentcard.Config{
		ProtocolVersion: "v1",
		Name:            "Kairos Spreadsheet Agent",
		Description:     "Runs safe, structured queries over CSV sheets.",
		Version:         "0.2.0",
		Capabilities: func() *a2av1.AgentCapabilities {
			streaming := true
			return &a2av1.AgentCapabilities{Streaming: &streaming}
		}(),
		SupportedInterfaces: []*a2av1.AgentInterface{
			{Url: "grpc://localhost" + *addr, ProtocolBinding: "grpc"},
		},
		Skills: []*a2av1.AgentSkill{
			{Id: "query_spreadsheet", Name: "query_spreadsheet", Description: "Run safe, predefined queries on spreadsheets."},
			{Id: "list_sheets", Name: "list_sheets", Description: "List available sheets."},
			{Id: "get_schema", Name: "get_schema", Description: "Return schema for a sheet."},
		},
	})

	handler := &server.SimpleHandler{
		Store:    server.NewMemoryTaskStore(),
		Executor: exec,
		Card:     card,
		PushCfgs: server.NewMemoryPushConfigStore(),
	}
	service := server.New(handler)

	mux := http.NewServeMux()
	mux.Handle(agentcard.WellKnownPath, agentcard.PublishHandler(card))
	go func() {
		_ = http.ListenAndServe(*cardAddr, mux)
	}()

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	a2av1.RegisterA2AServiceServer(grpcServer, service)
	log.Printf("spreadsheet agent listening on %s", *addr)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func decodeQuerySpec(payload map[string]interface{}) demo.QuerySpec {
	spec := demo.QuerySpec{}
	if value, ok := payload["type"].(string); ok {
		spec.Type = value
	}
	if value, ok := payload["quarter"].(string); ok {
		spec.Quarter = value
	}
	if value, ok := payload["month"].(string); ok {
		spec.Month = value
	}
	if value, ok := payload["limit"].(float64); ok {
		spec.Limit = int(value)
	}
	return spec
}

func extractJSON(output any) map[string]interface{} {
	text := strings.TrimSpace(fmt.Sprint(output))
	if text == "" {
		return nil
	}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end <= start {
		return nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(text[start:end+1]), &payload); err != nil {
		return nil
	}
	return payload
}

func toInterfaceSlice(values []string) []interface{} {
	out := make([]interface{}, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func toInterfaceRows(rows [][]string) []interface{} {
	out := make([]interface{}, 0, len(rows))
	for _, row := range rows {
		out = append(out, toInterfaceSlice(row))
	}
	return out
}
