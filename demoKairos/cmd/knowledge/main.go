package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
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

func main() {
	var (
		addr       = flag.String("addr", ":9031", "gRPC listen address")
		qdrantURL  = flag.String("qdrant", "localhost:6334", "Qdrant gRPC address")
		collection = flag.String("collection", "kairos_demo_docs", "Qdrant collection")
		docsDir    = flag.String("docs", "", "Docs directory")
		configPath = flag.String("config", "", "Config file path")
		mcpAddr    = flag.String("mcp-addr", "127.0.0.1:9041", "MCP streamable HTTP address")
		embedModel = flag.String("embed-model", "nomic-embed-text", "Ollama embed model")
		cardAddr   = flag.String("card-addr", "127.0.0.1:9141", "AgentCard HTTP address")
		verbose    = flag.Bool("verbose", false, "Enable verbose telemetry output")
	)
	flag.Parse()

	if *docsDir == "" {
		cwd, _ := os.Getwd()
		*docsDir = filepath.Join(cwd, "data")
	}

	cfg, err := demo.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	shutdown, err := demo.InitTelemetry("knowledge-agent", cfg, *verbose)
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

	store, err := demo.NewQdrantStore(demo.QdrantConfig{URL: *qdrantURL, Collection: *collection})
	if err != nil {
		log.Fatalf("qdrant: %v", err)
	}
	embedder := ollama.NewEmbedder(os.Getenv("OLLAMA_URL"), *embedModel)
	if err := demo.EnsureCollection(context.Background(), store, embedder, *collection); err != nil {
		log.Fatalf("ensure collection: %v", err)
	}

	docs, err := loadDocs(*docsDir)
	if err != nil {
		log.Fatalf("load docs: %v", err)
	}
	if err := demo.IngestDocs(context.Background(), store, embedder, *collection, docs); err != nil {
		log.Fatalf("ingest docs: %v", err)
	}

	mcpServer, err := demo.StartMCPServer("knowledge-mcp", "0.1.0", *mcpAddr)
	if err != nil {
		log.Fatalf("mcp server: %v", err)
	}
	mcpServer.RegisterTool("retrieve_domain_knowledge", "Buscar definiciones del dataset", func(ctx context.Context, args map[string]interface{}) (*mcpgo.CallToolResult, error) {
		query, _ := args["query"].(string)
		if query == "" {
			query, _ = args["text"].(string)
		}
		if query == "" {
			return &mcpgo.CallToolResult{IsError: true, Content: []mcpgo.Content{mcpgo.TextContent{Type: "text", Text: "query requerido"}}}, nil
		}
		results, err := demo.SearchDocs(ctx, store, embedder, *collection, query, 4)
		if err != nil {
			return nil, err
		}
		var b strings.Builder
		for _, res := range results {
			text, _ := res.Point.Payload["text"].(string)
			source, _ := res.Point.Payload["source"].(string)
			if text == "" {
				continue
			}
			b.WriteString("- ")
			b.WriteString(text)
			if source != "" {
				b.WriteString(" (source: ")
				b.WriteString(source)
				b.WriteString(")")
			}
			b.WriteString("\n")
		}
		return &mcpgo.CallToolResult{Content: []mcpgo.Content{mcpgo.TextContent{Type: "text", Text: b.String()}}}, nil
	})

	mcpClient, err := demo.NewMCPClient(mcpServer.BaseURL())
	if err != nil {
		log.Fatalf("mcp client: %v", err)
	}

	memStore, err := memory.NewVectorMemory(context.Background(), store, embedder, "kairos_demo_knowledge_memory")
	if err != nil {
		log.Fatalf("memory: %v", err)
	}
	if err := memStore.Initialize(context.Background()); err != nil {
		log.Fatalf("memory init: %v", err)
	}

	role := "Eres un agente de conocimiento. Usa la herramienta retrieve_domain_knowledge para responder preguntas sobre el dataset. Responde en espanol con definiciones concisas."

	agentOpts := []agent.Option{
		agent.WithRole(role),
		agent.WithModel(cfg.LLM.Model),
		agent.WithMCPClients(mcpClient),
		agent.WithMemory(memStore),
	}
	if len(cfg.MCP.Servers) > 0 {
		agentOpts = append(agentOpts, agent.WithMCPServerConfigs(cfg.MCP.Servers))
	}
	knowledgeAgent, err := agent.New("knowledge-agent", llmProvider, agentOpts...)
	if err != nil {
		log.Fatalf("agent: %v", err)
	}

	card := agentcard.Build(agentcard.Config{
		ProtocolVersion: "v1",
		Name:            "Kairos Knowledge Agent",
		Description:     "Answers dataset definition questions using vector search.",
		Version:         "0.2.0",
		Capabilities: func() *a2av1.AgentCapabilities {
			streaming := true
			return &a2av1.AgentCapabilities{Streaming: &streaming}
		}(),
		SupportedInterfaces: []*a2av1.AgentInterface{
			{Url: "grpc://localhost" + *addr, ProtocolBinding: "grpc"},
		},
		Skills: []*a2av1.AgentSkill{
			{Id: "retrieve_domain_knowledge", Name: "retrieve_domain_knowledge", Description: "Retrieve dataset definitions and business rules."},
		},
	})

	handler := server.NewAgentHandler(knowledgeAgent, server.WithAgentCard(card))
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
	log.Printf("knowledge agent listening on %s", *addr)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func loadDocs(dir string) ([]demo.Doc, error) {
	files := []string{"dataset_docs.md", "columns.md"}
	var docs []demo.Doc
	for _, name := range files {
		path := filepath.Join(dir, name)
		payload, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		text := strings.TrimSpace(string(payload))
		if text == "" {
			continue
		}
		docs = append(docs, demo.Doc{
			ID:   uuid.NewString(),
			Text: text,
			Metadata: map[string]interface{}{
				"source": name,
			},
		})
	}
	return docs, nil
}

func init() {}
