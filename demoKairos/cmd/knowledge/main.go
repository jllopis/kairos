package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/jllopis/kairos/demoKairos/internal/demo"
	"github.com/jllopis/kairos/pkg/a2a/agentcard"
	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"github.com/jllopis/kairos/pkg/memory"
	"github.com/jllopis/kairos/pkg/memory/ollama"
	"google.golang.org/grpc"
)

type knowledgeExecutor struct {
	store      memory.VectorStore
	embedder   *ollama.Embedder
	collection string
}

func (e *knowledgeExecutor) Run(ctx context.Context, message *a2av1.Message) (any, []*a2av1.Artifact, error) {
	query := server.ExtractText(message)
	if query == "" {
		return demo.NewTextMessage(a2av1.Role_ROLE_AGENT, "No query provided.", message.ContextId, message.TaskId), nil, nil
	}

	results, err := demo.SearchDocs(ctx, e.store, e.embedder, e.collection, query, 4)
	if err != nil {
		return nil, nil, err
	}
	var b strings.Builder
	b.WriteString("Context for dataset questions:\n")
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
	return demo.NewTextMessage(a2av1.Role_ROLE_AGENT, b.String(), message.ContextId, message.TaskId), nil, nil
}

func main() {
	var (
		addr       = flag.String("addr", ":9031", "gRPC listen address")
		qdrantURL  = flag.String("qdrant", "localhost:6334", "Qdrant gRPC address")
		collection = flag.String("collection", "kairos_demo_docs", "Qdrant collection")
		docsDir    = flag.String("docs", "", "Docs directory")
		ollamaURL  = flag.String("ollama", "http://localhost:11434", "Ollama base URL")
		embedModel = flag.String("embed-model", "nomic-embed-text", "Ollama embed model")
	)
	flag.Parse()

	if *docsDir == "" {
		cwd, _ := os.Getwd()
		*docsDir = filepath.Join(cwd, "data")
	}

	ctx := context.Background()
	store, err := demo.NewQdrantStore(demo.QdrantConfig{URL: *qdrantURL, Collection: *collection})
	if err != nil {
		log.Fatalf("qdrant: %v", err)
	}
	embedder := ollama.NewEmbedder(*ollamaURL, *embedModel)
	if err := demo.EnsureCollection(ctx, store, embedder, *collection); err != nil {
		log.Fatalf("ensure collection: %v", err)
	}

	docs, err := loadDocs(*docsDir)
	if err != nil {
		log.Fatalf("load docs: %v", err)
	}
	if err := demo.IngestDocs(ctx, store, embedder, *collection, docs); err != nil {
		log.Fatalf("ingest docs: %v", err)
	}

	card := agentcard.Build(agentcard.Config{
		ProtocolVersion: "v1",
		Name:            "Kairos Knowledge Agent",
		Description:     "Answers dataset definition questions using vector search.",
		Version:         "0.1.0",
		SupportedInterfaces: []*a2av1.AgentInterface{
			{Url: "grpc://localhost" + *addr, ProtocolBinding: "grpc"},
		},
		Skills: []*a2av1.AgentSkill{
			{Id: "retrieve_domain_knowledge", Name: "retrieve_domain_knowledge", Description: "Retrieve dataset definitions and business rules."},
		},
	})

	exec := &knowledgeExecutor{store: store, embedder: embedder, collection: *collection}
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
