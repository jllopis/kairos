package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jllopis/kairos/demoKairos/internal/demo"
	"github.com/jllopis/kairos/pkg/a2a/agentcard"
	"github.com/jllopis/kairos/pkg/a2a/client"
	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/memory"
	"github.com/jllopis/kairos/pkg/memory/ollama"
	"github.com/jllopis/kairos/pkg/planner"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type orchestratorHandler struct {
	store           server.TaskStore
	plan            *planner.Graph
	classifier      *agent.Agent
	synthesizer     *agent.Agent
	knowledgeClient *client.Client
	spreadClient    *client.Client
	knowledgeCard   string
	spreadsheetCard string
	card            *a2av1.AgentCard
}

func (h *orchestratorHandler) AgentCard() *a2av1.AgentCard {
	return h.card
}

func (h *orchestratorHandler) SendMessage(ctx context.Context, req *a2av1.SendMessageRequest) (*a2av1.SendMessageResponse, error) {
	message := req.GetRequest()
	if err := server.ValidateMessage(message); err != nil {
		log.Printf("orchestrator validate error: %v", err)
		return nil, err
	}
	task, _, err := h.ensureTask(ctx, message)
	if err != nil {
		log.Printf("orchestrator ensure task error: %v", err)
		return nil, err
	}
	resp, err := h.runOrchestration(ctx, task, message, nil)
	if err != nil {
		log.Printf("orchestrator run error: %v", err)
		return nil, err
	}
	return &a2av1.SendMessageResponse{Payload: &a2av1.SendMessageResponse_Msg{Msg: resp}}, nil
}

func (h *orchestratorHandler) SendStreamingMessage(req *a2av1.SendMessageRequest, stream a2av1.A2AService_SendStreamingMessageServer) error {
	message := req.GetRequest()
	if err := server.ValidateMessage(message); err != nil {
		log.Printf("orchestrator validate error: %v", err)
		return err
	}
	task, _, err := h.ensureTask(stream.Context(), message)
	if err != nil {
		log.Printf("orchestrator ensure task error: %v", err)
		return err
	}
	if err := stream.Send(&a2av1.StreamResponse{Payload: &a2av1.StreamResponse_Task{Task: task}}); err != nil {
		log.Printf("orchestrator send task error: %v", err)
		return err
	}
	_, err = h.runOrchestration(stream.Context(), task, message, stream)
	if err != nil {
		log.Printf("orchestrator run error: %v", err)
	}
	return err
}

func (h *orchestratorHandler) GetTask(ctx context.Context, req *a2av1.GetTaskRequest) (*a2av1.Task, error) {
	taskID, err := parseTaskName(req.GetName())
	if err != nil {
		return nil, err
	}
	return h.store.GetTask(ctx, taskID, req.GetHistoryLength(), false)
}

func (h *orchestratorHandler) ListTasks(ctx context.Context, req *a2av1.ListTasksRequest) (*a2av1.ListTasksResponse, error) {
	filter := server.TaskFilter{
		ContextID:        req.GetContextId(),
		Status:           req.GetStatus(),
		PageSize:         req.GetPageSize(),
		PageToken:        req.GetPageToken(),
		HistoryLength:    req.GetHistoryLength(),
		IncludeArtifacts: req.GetIncludeArtifacts(),
	}
	if req.GetLastUpdatedAfter() > 0 {
		filter.LastUpdatedAfter = time.UnixMilli(req.GetLastUpdatedAfter()).UTC()
	}
	tasks, total, err := h.store.ListTasks(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &a2av1.ListTasksResponse{
		Tasks:     tasks,
		PageSize:  filter.PageSize,
		TotalSize: int32(total),
	}, nil
}

func (h *orchestratorHandler) CancelTask(ctx context.Context, req *a2av1.CancelTaskRequest) (*a2av1.Task, error) {
	taskID, err := parseTaskName(req.GetName())
	if err != nil {
		return nil, err
	}
	return h.store.CancelTask(ctx, taskID)
}

func (h *orchestratorHandler) SubscribeToTask(req *a2av1.SubscribeToTaskRequest, stream a2av1.A2AService_SubscribeToTaskServer) error {
	return fmt.Errorf("SubscribeToTask not implemented in demo")
}

func (h *orchestratorHandler) GetExtendedAgentCard(ctx context.Context, req *a2av1.GetExtendedAgentCardRequest) (*a2av1.AgentCard, error) {
	return h.AgentCard(), nil
}

func (h *orchestratorHandler) ensureTask(ctx context.Context, message *a2av1.Message) (*a2av1.Task, bool, error) {
	if message.TaskId == "" {
		task, err := h.store.CreateTask(ctx, message)
		return task, true, err
	}
	task, err := h.store.GetTask(ctx, message.TaskId, 0, true)
	if err != nil {
		return nil, false, err
	}
	message.ContextId = task.ContextId
	if err := h.store.AppendHistory(ctx, task.Id, message); err != nil {
		return nil, false, err
	}
	return task, true, nil
}

func (h *orchestratorHandler) runOrchestration(ctx context.Context, task *a2av1.Task, message *a2av1.Message, stream a2av1.A2AService_SendStreamingMessageServer) (*a2av1.Message, error) {
	query := server.ExtractText(message)
	if query == "" {
		resp := demo.NewTextMessage(a2av1.Role_ROLE_AGENT, "No query provided.", task.ContextId, task.Id)
		return resp, nil
	}

	_ = h.store.UpdateStatus(ctx, task.Id, &a2av1.TaskStatus{
		State:     a2av1.TaskState_TASK_STATE_WORKING,
		Message:   message,
		Timestamp: timestamppb.Now(),
	})

	state := planner.NewState()
	state.Outputs["user_query"] = query
	state.Outputs["task_id"] = task.Id
	state.Outputs["context_id"] = task.ContextId
	state.Outputs["stream"] = stream

	executor := planner.NewExecutor(map[string]planner.Handler{
		"detect_intent": func(ctx context.Context, node planner.Node, state *planner.State) (any, error) {
			return h.handleDetectIntent(ctx, state)
		},
		"knowledge": func(ctx context.Context, node planner.Node, state *planner.State) (any, error) {
			return h.handleKnowledge(ctx, state)
		},
		"spreadsheet": func(ctx context.Context, node planner.Node, state *planner.State) (any, error) {
			return h.handleSpreadsheet(ctx, state)
		},
		"synthesize": func(ctx context.Context, node planner.Node, state *planner.State) (any, error) {
			return h.handleSynthesize(ctx, state)
		},
	})

	_, err := executor.Execute(ctx, h.plan, state)
	if err != nil {
		h.failWithStatus(ctx, stream, task, fmt.Sprintf("Fallo ejecutando plan: %v", err))
		return nil, err
	}
	finalText, _ := state.Outputs["final"].(string)
	respMsg := demo.NewTextMessage(a2av1.Role_ROLE_AGENT, finalText, task.ContextId, task.Id)
	_ = h.store.AppendHistory(ctx, task.Id, respMsg)
	_ = h.store.UpdateStatus(ctx, task.Id, &a2av1.TaskStatus{
		State:     a2av1.TaskState_TASK_STATE_COMPLETED,
		Message:   respMsg,
		Timestamp: timestamppb.Now(),
	})

	sendResponseDelta(stream, task, finalText)
	sendFinal(stream, task, respMsg)
	return respMsg, nil
}

func (h *orchestratorHandler) handleDetectIntent(ctx context.Context, state *planner.State) (any, error) {
	query, _ := state.Outputs["user_query"].(string)
	stream := streamFromState(state)
	taskID, _ := state.Outputs["task_id"].(string)
	contextID, _ := state.Outputs["context_id"].(string)

	sendStatus(stream, taskID, contextID, demo.EventThinking, "Analizando la peticion...")
	prompt := fmt.Sprintf("Clasifica la intencion de la consulta en uno de estos ids: sales_by_region, top_products_margin_compare, gastos_anomalies. Responde solo con el id. Consulta: %s", query)
	output, err := h.classifier.Run(ctx, prompt)
	if err != nil {
		return nil, err
	}
	intent := parseIntent(fmt.Sprint(output))
	if intent == "" {
		return nil, fmt.Errorf("intent not recognized")
	}
	state.Outputs["intent"] = intent
	return intent, nil
}

func (h *orchestratorHandler) handleKnowledge(ctx context.Context, state *planner.State) (any, error) {
	stream := streamFromState(state)
	query, _ := state.Outputs["user_query"].(string)
	intent, _ := state.Outputs["intent"].(string)
	taskID, _ := state.Outputs["task_id"].(string)
	contextID, _ := state.Outputs["context_id"].(string)

	if err := h.ensureAgentCard(ctx, h.knowledgeCard, "knowledge"); err != nil {
		return nil, err
	}
	sendStatus(stream, taskID, contextID, demo.EventRetrievalStart, "Buscando definiciones y reglas...")
	msg := demo.NewTextMessage(a2av1.Role_ROLE_USER, fmt.Sprintf("Consulta: %s\nIntencion: %s", query, intent), contextID, "")
	resp, err := h.knowledgeClient.SendMessage(ctx, &a2av1.SendMessageRequest{
		Request:       msg,
		Configuration: &a2av1.SendMessageConfiguration{Blocking: true},
	})
	if err != nil {
		return nil, err
	}
	knowledge := server.ExtractText(resp.GetMsg())
	sendStatus(stream, taskID, contextID, demo.EventRetrievalDone, "Contexto obtenido.")
	state.Outputs["knowledge"] = knowledge
	return knowledge, nil
}

func (h *orchestratorHandler) handleSpreadsheet(ctx context.Context, state *planner.State) (any, error) {
	stream := streamFromState(state)
	intent, _ := state.Outputs["intent"].(string)
	query, _ := state.Outputs["user_query"].(string)
	taskID, _ := state.Outputs["task_id"].(string)
	contextID, _ := state.Outputs["context_id"].(string)
	if err := h.ensureAgentCard(ctx, h.spreadsheetCard, "spreadsheet"); err != nil {
		return nil, err
	}
	sendStatus(stream, taskID, contextID, demo.EventToolStart, "Consultando hoja de calculo...")
	spec := querySpecForIntent(intent, query)
	msg := demo.NewDataMessage(a2av1.Role_ROLE_USER, spec, contextID, "")
	resp, err := h.spreadClient.SendMessage(ctx, &a2av1.SendMessageRequest{
		Request:       msg,
		Configuration: &a2av1.SendMessageConfiguration{Blocking: true},
	})
	if err != nil {
		return nil, err
	}
	data := server.ExtractData(resp.GetMsg())
	if data == nil {
		return nil, fmt.Errorf("spreadsheet response missing data")
	}
	data["intent"] = intent
	sendStatus(stream, taskID, contextID, demo.EventToolDone, "Datos listos.")
	state.Outputs["data"] = data
	return data, nil
}

func (h *orchestratorHandler) handleSynthesize(ctx context.Context, state *planner.State) (any, error) {
	knowledge, _ := state.Outputs["knowledge"].(string)
	data, _ := state.Outputs["data"].(map[string]interface{})
	intent, _ := state.Outputs["intent"].(string)
	query, _ := state.Outputs["user_query"].(string)

	headers, rows := extractTable(data)
	prompt := buildSynthesisPrompt(query, intent, knowledge, headers, rows, data)
	output, err := h.synthesizer.Run(ctx, prompt)
	if err != nil {
		return nil, err
	}
	final := strings.TrimSpace(fmt.Sprint(output))
	state.Outputs["final"] = final
	return final, nil
}

func streamFromState(state *planner.State) a2av1.A2AService_SendStreamingMessageServer {
	if state == nil {
		return nil
	}
	stream, _ := state.Outputs["stream"].(a2av1.A2AService_SendStreamingMessageServer)
	return stream
}

func sendResponseDelta(stream a2av1.A2AService_SendStreamingMessageServer, task *a2av1.Task, text string) {
	if stream == nil {
		return
	}
	chunks := chunkText(text, 160)
	for _, chunk := range chunks {
		msg := demo.NewTextMessage(a2av1.Role_ROLE_AGENT, chunk, task.ContextId, task.Id)
		_ = stream.Send(&a2av1.StreamResponse{Payload: &a2av1.StreamResponse_Msg{Msg: msg}})
	}
}

func sendStatus(stream a2av1.A2AService_SendStreamingMessageServer, taskID, contextID, eventType, message string) {
	if stream == nil {
		return
	}
	_ = stream.Send(demo.StatusEvent(taskID, contextID, eventType, message, false))
}

func (h *orchestratorHandler) ensureAgentCard(ctx context.Context, baseURL, label string) error {
	if baseURL == "" {
		return fmt.Errorf("missing %s agent card url", label)
	}
	card, err := agentcard.Fetch(ctx, baseURL)
	if err != nil {
		return fmt.Errorf("%s agent card fetch failed: %w", label, err)
	}
	if card.GetCapabilities() == nil || !card.GetCapabilities().GetStreaming() {
		return fmt.Errorf("%s agent does not advertise streaming capability", label)
	}
	return nil
}

func sendFinal(stream a2av1.A2AService_SendStreamingMessageServer, task *a2av1.Task, msg *a2av1.Message) {
	if stream == nil {
		return
	}
	status := demo.StatusEventWithState(task.Id, task.ContextId, "response.final", "Respuesta completa.", a2av1.TaskState_TASK_STATE_COMPLETED, true)
	_ = stream.Send(status)
	_ = stream.Send(&a2av1.StreamResponse{Payload: &a2av1.StreamResponse_Msg{Msg: msg}})
}

func (h *orchestratorHandler) failWithStatus(ctx context.Context, stream a2av1.A2AService_SendStreamingMessageServer, task *a2av1.Task, message string) {
	status := demo.StatusEventWithState(task.Id, task.ContextId, "error", message, a2av1.TaskState_TASK_STATE_FAILED, true)
	if stream != nil {
		_ = stream.Send(status)
	}
	_ = h.store.UpdateStatus(ctx, task.Id, &a2av1.TaskStatus{
		State:     a2av1.TaskState_TASK_STATE_FAILED,
		Message:   demo.NewTextMessage(a2av1.Role_ROLE_AGENT, message, task.ContextId, task.Id),
		Timestamp: timestamppb.Now(),
	})
}

func parseIntent(value string) string {
	lower := strings.ToLower(value)
	switch {
	case strings.Contains(lower, "sales_by_region"):
		return "sales_by_region"
	case strings.Contains(lower, "top_products_margin_compare"):
		return "top_products_margin_compare"
	case strings.Contains(lower, "gastos_anomalies"):
		return "gastos_anomalies"
	default:
		return ""
	}
}

func querySpecForIntent(intent, query string) map[string]interface{} {
	spec := map[string]interface{}{
		"type": intent,
	}
	if quarter := extractQuarter(query); quarter != "" {
		spec["quarter"] = quarter
	}
	if limit := extractLimit(query); limit > 0 {
		spec["limit"] = limit
	}
	if month := extractMonth(query); month != "" {
		spec["month"] = month
	}
	return spec
}

func extractQuarter(query string) string {
	re := regexp.MustCompile(`(?i)\bq([1-4])\b`)
	match := re.FindStringSubmatch(query)
	if len(match) != 2 {
		return ""
	}
	return "Q" + match[1]
}

func extractLimit(query string) int {
	re := regexp.MustCompile(`(?i)\btop\s+(\d+)\b`)
	match := re.FindStringSubmatch(query)
	if len(match) != 2 {
		return 0
	}
	value, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return value
}

func extractMonth(query string) string {
	re := regexp.MustCompile(`\b20\d{2}-(0[1-9]|1[0-2])\b`)
	return re.FindString(query)
}

func buildSynthesisPrompt(query, intent, knowledge string, headers []string, rows [][]string, data map[string]interface{}) string {
	var b strings.Builder
	b.WriteString("Eres un agente que responde preguntas sobre datos. Responde en espanol y con trazabilidad.\n")
	b.WriteString("Consulta del usuario: ")
	b.WriteString(query)
	b.WriteString("\nIntencion: ")
	b.WriteString(intent)
	b.WriteString("\nContexto del conocimiento:\n")
	b.WriteString(knowledge)
	b.WriteString("\nDatos:\n")
	b.WriteString(demo.FormatTable(headers, rows))
	if meta, ok := data["meta"].(map[string]interface{}); ok {
		b.WriteString("\nTrazabilidad (meta):\n")
		for key, value := range meta {
			b.WriteString(fmt.Sprintf("- %s: %v\n", key, value))
		}
	}
	b.WriteString("\nResponde con un resumen claro y la tabla si aplica.")
	return b.String()
}

func extractTable(data map[string]interface{}) ([]string, [][]string) {
	var headers []string
	var rows [][]string
	if data == nil {
		return headers, rows
	}
	if raw, ok := data["headers"].([]interface{}); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				headers = append(headers, s)
			}
		}
	}
	if raw, ok := data["rows"].([]interface{}); ok {
		for _, row := range raw {
			if rowSlice, ok := row.([]interface{}); ok {
				cells := make([]string, 0, len(rowSlice))
				for _, cell := range rowSlice {
					cells = append(cells, fmt.Sprint(cell))
				}
				rows = append(rows, cells)
			}
		}
	}
	return headers, rows
}

func chunkText(text string, size int) []string {
	if size <= 0 || len(text) <= size {
		return []string{text}
	}
	var chunks []string
	for len(text) > 0 {
		if len(text) <= size {
			chunks = append(chunks, text)
			break
		}
		chunks = append(chunks, text[:size])
		text = text[size:]
	}
	return chunks
}

func parseTaskName(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("task name is required")
	}
	parts := strings.Split(name, "/")
	if len(parts) == 2 && parts[0] == "tasks" {
		return parts[1], nil
	}
	if strings.Contains(name, "/") {
		return "", fmt.Errorf("invalid task name %q", name)
	}
	return name, nil
}

func main() {
	var (
		addr               = flag.String("addr", ":9030", "gRPC listen address")
		knowledge          = flag.String("knowledge", "localhost:9031", "Knowledge agent gRPC endpoint")
		spreadsheet        = flag.String("spreadsheet", "localhost:9032", "Spreadsheet agent gRPC endpoint")
		knowledgeCardURL   = flag.String("knowledge-card-url", "http://localhost:9141", "Knowledge AgentCard base URL")
		spreadsheetCardURL = flag.String("spreadsheet-card-url", "http://localhost:9142", "Spreadsheet AgentCard base URL")
		qdrantURL          = flag.String("qdrant", "localhost:6334", "Qdrant gRPC address")
		memColl            = flag.String("memory-collection", "kairos_demo_orch_memory", "Qdrant memory collection")
		configPath         = flag.String("config", "", "Config file path")
		embedModel         = flag.String("embed-model", "nomic-embed-text", "Ollama embed model")
		planPath           = flag.String("plan", "", "Planner YAML path")
		cardAddr           = flag.String("card-addr", "127.0.0.1:9140", "AgentCard HTTP address")
		verbose            = flag.Bool("verbose", false, "Enable verbose telemetry output")
	)
	flag.Parse()

	cfg, err := demo.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	shutdown, err := demo.InitTelemetry("orchestrator", cfg, *verbose)
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

	classifierRole := "Eres un clasificador de intencion. Responde solo con uno de estos ids: sales_by_region, top_products_margin_compare, gastos_anomalies."
	synthRole := "Eres un agente que sintetiza respuestas con datos y contexto."

	classifierOpts := []agent.Option{agent.WithRole(classifierRole), agent.WithModel(cfg.LLM.Model)}
	synthOpts := []agent.Option{agent.WithRole(synthRole), agent.WithModel(cfg.LLM.Model), agent.WithMemory(memStore)}
	if len(cfg.MCP.Servers) > 0 {
		classifierOpts = append(classifierOpts, agent.WithMCPServerConfigs(cfg.MCP.Servers))
		synthOpts = append(synthOpts, agent.WithMCPServerConfigs(cfg.MCP.Servers))
	}

	classifier, err := agent.New("orchestrator-classifier", llmProvider, classifierOpts...)
	if err != nil {
		log.Fatalf("classifier: %v", err)
	}
	synthesizer, err := agent.New("orchestrator-synth", llmProvider, synthOpts...)
	if err != nil {
		log.Fatalf("synthesizer: %v", err)
	}

	planFile := *planPath
	if planFile == "" {
		cwd, _ := os.Getwd()
		planFile = filepath.Join(cwd, "data", "orchestrator_plan.yaml")
	}
	payload, err := os.ReadFile(planFile)
	if err != nil {
		log.Fatalf("plan: %v", err)
	}
	graph, err := planner.ParseYAML(payload)
	if err != nil {
		log.Fatalf("plan parse: %v", err)
	}

	knowledgeConn, err := grpc.Dial(*knowledge, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("knowledge dial: %v", err)
	}
	spreadConn, err := grpc.Dial(*spreadsheet, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("spreadsheet dial: %v", err)
	}

	knowledgeClient := client.New(knowledgeConn)
	spreadClient := client.New(spreadConn)

	handler := &orchestratorHandler{
		store:           server.NewMemoryTaskStore(),
		plan:            graph,
		classifier:      classifier,
		synthesizer:     synthesizer,
		knowledgeClient: knowledgeClient,
		spreadClient:    spreadClient,
		knowledgeCard:   *knowledgeCardURL,
		spreadsheetCard: *spreadsheetCardURL,
		card: agentcard.Build(agentcard.Config{
			ProtocolVersion: "v1",
			Name:            "Kairos Orchestrator",
			Description:     "Routes user questions to knowledge + spreadsheet agents.",
			Version:         "0.2.0",
			Capabilities: func() *a2av1.AgentCapabilities {
				streaming := true
				return &a2av1.AgentCapabilities{Streaming: &streaming}
			}(),
			SupportedInterfaces: []*a2av1.AgentInterface{
				{Url: "grpc://localhost" + *addr, ProtocolBinding: "grpc"},
			},
			Skills: []*a2av1.AgentSkill{
				{Id: "orchestrate", Name: "orchestrate", Description: "Coordinate multi-agent data answers."},
			},
		}),
	}

	mux := http.NewServeMux()
	mux.Handle(agentcard.WellKnownPath, agentcard.PublishHandler(handler.card))
	go func() {
		_ = http.ListenAndServe(*cardAddr, mux)
	}()

	grpcServer := grpc.NewServer()
	a2av1.RegisterA2AServiceServer(grpcServer, server.New(handler))

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("orchestrator listening on %s", *addr)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
