package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jllopis/kairos/demoKairos/internal/demo"
)

func main() {
	log.SetFlags(log.LstdFlags)

	qdrantAddr := flag.String("qdrant", getenv("QDRANT_URL", "localhost:6334"), "Qdrant gRPC address")
	ollamaURL := flag.String("ollama", getenv("OLLAMA_URL", "http://localhost:11434"), "Ollama base URL")
	embedModel := flag.String("embed-model", getenv("EMBED_MODEL", "nomic-embed-text"), "Embedding model")
	llmModel := flag.String("llm-model", getenv("KAIROS_LLM_MODEL", "qwen2.5-coder:7b-instruct-q5_K_M"), "LLM model")
	llmProvider := flag.String("llm-provider", getenv("KAIROS_LLM_PROVIDER", "ollama"), "LLM provider")
	configPath := flag.String("config", os.Getenv("CONFIG_PATH"), "Config file path")
	verbose := flag.Bool("verbose", envBool("DEMO_VERBOSE"), "Enable verbose traces")

	knowledgeAddr := flag.String("knowledge-addr", getenv("KNOWLEDGE_ADDR", ":9031"), "Knowledge agent listen address")
	spreadsheetAddr := flag.String("spreadsheet-addr", getenv("SPREADSHEET_ADDR", ":9032"), "Spreadsheet agent listen address")
	orchestratorAddr := flag.String("orchestrator-addr", getenv("ORCH_ADDR", ":9030"), "Orchestrator listen address")
	knowledgeMCP := flag.String("knowledge-mcp", getenv("KNOWLEDGE_MCP_ADDR", "127.0.0.1:9041"), "Knowledge MCP address")
	spreadsheetMCP := flag.String("spreadsheet-mcp", getenv("SPREADSHEET_MCP_ADDR", "127.0.0.1:9042"), "Spreadsheet MCP address")
	knowledgeCard := flag.String("knowledge-card", getenv("KNOWLEDGE_CARD_ADDR", "127.0.0.1:9141"), "Knowledge card address")
	spreadsheetCard := flag.String("spreadsheet-card", getenv("SPREADSHEET_CARD_ADDR", "127.0.0.1:9142"), "Spreadsheet card address")
	orchestratorCard := flag.String("orchestrator-card", getenv("ORCH_CARD_ADDR", "127.0.0.1:9140"), "Orchestrator card address")

	rootOverride := flag.String("root", os.Getenv("DEMO_ROOT"), "Demo root override")
	flag.Parse()

	checkTCP("Qdrant gRPC", *qdrantAddr)
	checkHTTP("Ollama HTTP", *ollamaURL)

	system, err := demo.NewSystem()
	if err != nil {
		log.Fatalf("demo: %v", err)
	}
	if strings.TrimSpace(*rootOverride) != "" {
		if err := system.SetRoot(*rootOverride); err != nil {
			log.Fatalf("demo: %v", err)
		}
	}

	system.WithEnvMap(map[string]string{
		"OLLAMA_URL":          *ollamaURL,
		"KAIROS_LLM_PROVIDER": *llmProvider,
		"KAIROS_LLM_MODEL":    *llmModel,
	})
	flow := "UserQuery -> Knowledge -> Spreadsheet -> Orchestrator"
	system.WithFlow(demo.Flow(flow))

	root := system.Root()
	dataDir := filepath.Join(root, "data")

	knowledgeArgs := []string{
		"run", "./cmd/knowledge",
		"--addr", *knowledgeAddr,
		"--qdrant", *qdrantAddr,
		"--embed-model", *embedModel,
		"--mcp-addr", *knowledgeMCP,
		"--card-addr", *knowledgeCard,
	}
	spreadsheetArgs := []string{
		"run", "./cmd/spreadsheet",
		"--addr", *spreadsheetAddr,
		"--data", dataDir,
		"--qdrant", *qdrantAddr,
		"--embed-model", *embedModel,
		"--mcp-addr", *spreadsheetMCP,
		"--card-addr", *spreadsheetCard,
	}
	orchestratorArgs := []string{
		"run", "./cmd/orchestrator",
		"--addr", *orchestratorAddr,
		"--knowledge", hostAddr(*knowledgeAddr),
		"--spreadsheet", hostAddr(*spreadsheetAddr),
		"--qdrant", *qdrantAddr,
		"--embed-model", *embedModel,
		"--knowledge-card-url", fmt.Sprintf("http://%s", *knowledgeCard),
		"--spreadsheet-card-url", fmt.Sprintf("http://%s", *spreadsheetCard),
		"--card-addr", *orchestratorCard,
	}

	if strings.TrimSpace(*configPath) != "" {
		knowledgeArgs = append(knowledgeArgs, "--config", *configPath)
		spreadsheetArgs = append(spreadsheetArgs, "--config", *configPath)
		orchestratorArgs = append(orchestratorArgs, "--config", *configPath)
	}
	if *verbose {
		knowledgeArgs = append(knowledgeArgs, "--verbose")
		spreadsheetArgs = append(spreadsheetArgs, "--verbose")
		orchestratorArgs = append(orchestratorArgs, "--verbose")
	}

	system.WithAgent(demo.AgentSpec{
		Name:    "knowledge",
		Command: "go",
		Args:    knowledgeArgs,
	})
	system.WithAgent(demo.AgentSpec{
		Name:    "spreadsheet",
		Command: "go",
		Args:    spreadsheetArgs,
	})
	system.WithAgent(demo.AgentSpec{
		Name:    "orchestrator",
		Command: "go",
		Args:    orchestratorArgs,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := system.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("demo run: %v", err)
	}
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return false
	}
	value = strings.ToLower(value)
	return value == "1" || value == "true" || value == "yes"
}

func hostAddr(addr string) string {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, ":") {
		return "localhost" + trimmed
	}
	return trimmed
}

func checkTCP(name, addr string) {
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		log.Printf("WARN: %s not reachable at %s", name, addr)
		return
	}
	_ = conn.Close()
}

func checkHTTP(name, rawURL string) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("WARN: %s URL invalid: %s", name, rawURL)
		return
	}
	host := parsed.Host
	if host == "" {
		host = parsed.Path
	}
	if host == "" {
		log.Printf("WARN: %s URL invalid: %s", name, rawURL)
		return
	}
	if !strings.Contains(host, ":") {
		if parsed.Scheme == "https" {
			host += ":443"
		} else {
			host += ":80"
		}
	}
	checkTCP(name, host)
}
