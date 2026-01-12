package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/jllopis/kairos/demoKairos/internal/demo"
	"github.com/jllopis/kairos/pkg/a2a/agentcard"
	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc"
)

type spreadsheetExecutor struct {
	store *demo.SpreadsheetStore
}

func (e *spreadsheetExecutor) Run(ctx context.Context, message *a2av1.Message) (any, []*a2av1.Artifact, error) {
	data := server.ExtractData(message)
	if data == nil {
		return demo.NewTextMessage(a2av1.Role_ROLE_AGENT, "Expected structured query input.", message.ContextId, message.TaskId), nil, nil
	}
	spec := decodeQuerySpec(data)
	result, err := e.store.Query(spec)
	if err != nil {
		return nil, nil, err
	}
	payload := map[string]interface{}{
		"headers": result.Headers,
		"rows":    result.Rows,
		"meta":    result.Meta,
	}
	return demo.NewDataMessage(a2av1.Role_ROLE_AGENT, payload, message.ContextId, message.TaskId), nil, nil
}

func main() {
	var (
		addr    = flag.String("addr", ":9032", "gRPC listen address")
		dataDir = flag.String("data", "", "CSV data directory")
	)
	flag.Parse()

	if *dataDir == "" {
		cwd, _ := os.Getwd()
		*dataDir = filepath.Join(cwd, "data")
	}

	store, err := demo.LoadSpreadsheetStore(*dataDir)
	if err != nil {
		log.Fatalf("load data: %v", err)
	}

	card := agentcard.Build(agentcard.Config{
		ProtocolVersion: "v1",
		Name:            "Kairos Spreadsheet Agent",
		Description:     "Runs safe, structured queries over CSV sheets.",
		Version:         "0.1.0",
		SupportedInterfaces: []*a2av1.AgentInterface{
			{Url: "grpc://localhost" + *addr, ProtocolBinding: "grpc"},
		},
		Skills: []*a2av1.AgentSkill{
			{Id: "query_spreadsheet", Name: "query_spreadsheet", Description: "Run safe, predefined queries on spreadsheets."},
			{Id: "list_sheets", Name: "list_sheets", Description: "List available sheets."},
			{Id: "get_schema", Name: "get_schema", Description: "Return schema for a sheet."},
		},
	})

	exec := &spreadsheetExecutor{store: store}
	handler := &server.SimpleHandler{
		Store:    server.NewMemoryTaskStore(),
		Executor: exec,
		Card:     card,
		PushCfgs: server.NewMemoryPushConfigStore(),
	}
	service := server.New(handler)

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
