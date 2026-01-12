package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/jllopis/kairos/demoKairos/internal/demo"
	"github.com/jllopis/kairos/pkg/a2a/agentcard"
	"github.com/jllopis/kairos/pkg/a2a/client"
	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"github.com/jllopis/kairos/pkg/memory"
	"github.com/jllopis/kairos/pkg/memory/ollama"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type orchestratorHandler struct {
	store           server.TaskStore
	knowledgeClient *client.Client
	spreadClient    *client.Client
	memory          *memory.VectorMemory
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

	h.store.UpdateStatus(ctx, task.Id, &a2av1.TaskStatus{
		State:     a2av1.TaskState_TASK_STATE_WORKING,
		Message:   message,
		Timestamp: timestamppb.Now(),
	})

	sendStatus(stream, task, demo.EventThinking, "Analizando la peticion...")

	intent := detectIntent(query)
	if intent == "" {
		resp := demo.NewTextMessage(a2av1.Role_ROLE_AGENT, "No pude determinar la intencion de la pregunta.", task.ContextId, task.Id)
		sendFinal(stream, task, resp)
		return resp, nil
	}

	knowledge := ""
	sendStatus(stream, task, demo.EventRetrievalStart, "Buscando definiciones y reglas...")
	knowledge, err := h.askKnowledge(ctx, task, query)
	if err != nil {
		h.failWithStatus(ctx, stream, task, fmt.Sprintf("Fallo en knowledge agent: %v", err))
		return nil, err
	}
	sendStatus(stream, task, demo.EventRetrievalDone, "Contexto obtenido.")

	sendStatus(stream, task, demo.EventToolStart, "Consultando hoja de calculo...")
	dataMsg, err := h.askSpreadsheet(ctx, task, intent)
	if err != nil {
		h.failWithStatus(ctx, stream, task, fmt.Sprintf("Fallo consultando hoja: %v", err))
		return nil, err
	}
	sendStatus(stream, task, demo.EventToolDone, "Datos listos.")

	payload := server.ExtractData(dataMsg)
	response := composeResponse(intent, knowledge, payload)
	respMsg := demo.NewTextMessage(a2av1.Role_ROLE_AGENT, response, task.ContextId, task.Id)

	if h.memory != nil {
		h.memory.Store(ctx, fmt.Sprintf("User: %s\nAgent: %s", query, response))
	}

	_ = h.store.AppendHistory(ctx, task.Id, respMsg)
	h.store.UpdateStatus(ctx, task.Id, &a2av1.TaskStatus{
		State:     a2av1.TaskState_TASK_STATE_COMPLETED,
		Message:   respMsg,
		Timestamp: timestamppb.Now(),
	})

	sendResponseDelta(stream, task, response)
	sendFinal(stream, task, respMsg)
	return respMsg, nil
}

func (h *orchestratorHandler) askKnowledge(ctx context.Context, task *a2av1.Task, query string) (string, error) {
	if h.knowledgeClient == nil {
		return "", nil
	}
	msg := demo.NewTextMessage(a2av1.Role_ROLE_USER, query, task.ContextId, task.Id)
	resp, err := h.knowledgeClient.SendMessage(ctx, &a2av1.SendMessageRequest{Request: msg})
	if err != nil {
		return "", err
	}
	return server.ExtractText(resp.GetMsg()), nil
}

func (h *orchestratorHandler) askSpreadsheet(ctx context.Context, task *a2av1.Task, intent string) (*a2av1.Message, error) {
	if h.spreadClient == nil {
		return nil, fmt.Errorf("spreadsheet client not configured")
	}
	query := spreadsheetQuery(intent)
	msg := demo.NewDataMessage(a2av1.Role_ROLE_USER, query, task.ContextId, task.Id)
	resp, err := h.spreadClient.SendMessage(ctx, &a2av1.SendMessageRequest{Request: msg})
	if err != nil {
		return nil, err
	}
	return resp.GetMsg(), nil
}

func sendStatus(stream a2av1.A2AService_SendStreamingMessageServer, task *a2av1.Task, eventType, message string) {
	if stream == nil {
		return
	}
	_ = stream.Send(demo.StatusEvent(task.Id, task.ContextId, eventType, message, false))
}

func sendResponseDelta(stream a2av1.A2AService_SendStreamingMessageServer, task *a2av1.Task, text string) {
	if stream == nil {
		return
	}
	chunks := chunkText(text, 120)
	for _, chunk := range chunks {
		msg := demo.NewTextMessage(a2av1.Role_ROLE_AGENT, chunk, task.ContextId, task.Id)
		_ = stream.Send(&a2av1.StreamResponse{Payload: &a2av1.StreamResponse_Msg{Msg: msg}})
	}
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

func detectIntent(text string) string {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "ventas") && strings.Contains(lower, "q4") && strings.Contains(lower, "region") {
		return "sales_by_region"
	}
	if strings.Contains(lower, "top") && strings.Contains(lower, "margen") {
		return "top_products_margin_compare"
	}
	if strings.Contains(lower, "gastos") && strings.Contains(lower, "anomal") {
		return "gastos_anomalies"
	}
	return ""
}

func spreadsheetQuery(intent string) map[string]interface{} {
	switch intent {
	case "sales_by_region":
		return map[string]interface{}{
			"type":    "sales_by_region",
			"quarter": "Q4",
		}
	case "top_products_margin_compare":
		return map[string]interface{}{
			"type":    "top_products_margin_compare",
			"quarter": "Q4",
			"limit":   10,
		}
	case "gastos_anomalies":
		return map[string]interface{}{
			"type": "gastos_anomalies",
		}
	default:
		return map[string]interface{}{"type": ""}
	}
}

func composeResponse(intent, knowledge string, data map[string]interface{}) string {
	var b strings.Builder
	if knowledge != "" {
		b.WriteString("Contexto:\n")
		b.WriteString(knowledge)
		b.WriteString("\n")
	}
	b.WriteString("Resultado:\n")
	headers, rows := extractTable(data)
	b.WriteString(demo.FormatTable(headers, rows))
	if meta, ok := data["meta"].(map[string]interface{}); ok {
		b.WriteString("\nTrazabilidad:\n")
		for key, value := range meta {
			b.WriteString(fmt.Sprintf("- %s: %v\n", key, value))
		}
	}
	switch intent {
	case "sales_by_region":
		b.WriteString("\nFuente: hoja Ventas, filtro Q4, agregacion por region.\n")
	case "top_products_margin_compare":
		b.WriteString("\nFuente: hoja Ventas, comparado con trimestre anterior.\n")
	case "gastos_anomalies":
		b.WriteString("\nFuente: hoja Gastos, deteccion por desviacion estandar.\n")
	}
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
		addr        = flag.String("addr", ":9030", "gRPC listen address")
		knowledge   = flag.String("knowledge", "localhost:9031", "Knowledge agent gRPC endpoint")
		spreadsheet = flag.String("spreadsheet", "localhost:9032", "Spreadsheet agent gRPC endpoint")
		qdrantURL   = flag.String("qdrant", "localhost:6334", "Qdrant gRPC address")
		memColl     = flag.String("memory-collection", "kairos_demo_memory", "Qdrant memory collection")
		ollamaURL   = flag.String("ollama", "http://localhost:11434", "Ollama base URL")
		embedModel  = flag.String("embed-model", "nomic-embed-text", "Ollama embed model")
	)
	flag.Parse()

	ctx := context.Background()
	store, err := demo.NewQdrantStore(demo.QdrantConfig{URL: *qdrantURL, Collection: *memColl})
	if err != nil {
		log.Fatalf("qdrant: %v", err)
	}
	embedder := ollama.NewEmbedder(*ollamaURL, *embedModel)
	if err := demo.EnsureCollection(ctx, store, embedder, *memColl); err != nil {
		log.Fatalf("ensure memory collection: %v", err)
	}
	memoryStore, err := memory.NewVectorMemory(ctx, store, embedder, *memColl)
	if err != nil {
		log.Fatalf("memory: %v", err)
	}
	if err := memoryStore.Initialize(ctx); err != nil {
		log.Fatalf("memory init: %v", err)
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
		knowledgeClient: knowledgeClient,
		spreadClient:    spreadClient,
		memory:          memoryStore,
		card: agentcard.Build(agentcard.Config{
			ProtocolVersion: "v1",
			Name:            "Kairos Orchestrator",
			Description:     "Routes user questions to knowledge + spreadsheet agents.",
			Version:         "0.1.0",
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
