package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/jllopis/kairos/pkg/a2a/agentcard"
	"github.com/jllopis/kairos/pkg/a2a/client"
	httpjson "github.com/jllopis/kairos/pkg/a2a/httpjson/client"
	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"github.com/jllopis/kairos/pkg/config"
	kairosmcp "github.com/jllopis/kairos/pkg/mcp"
	mcptypes "github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultGRPCAddr = "localhost:8080"
	defaultHTTPURL  = "http://localhost:8080"
)

type globalFlags struct {
	ConfigArgs []string
	GRPCAddr   string
	HTTPURL    string
	Timeout    time.Duration
	JSON       bool
	Help       bool
}

type statusResult struct {
	Version        string `json:"version"`
	GRPCAddr       string `json:"grpc_addr"`
	GRPCReachable  bool   `json:"grpc_reachable"`
	HTTPURL        string `json:"http_url"`
	HTTPReachable  bool   `json:"http_reachable"`
	ConfigPathUsed string `json:"config_path_used,omitempty"`
}

type agentResult struct {
	URL  string           `json:"url"`
	Card *a2av1.AgentCard `json:"card,omitempty"`
	Err  string           `json:"error,omitempty"`
}

type mcpToolResult struct {
	Server string        `json:"server"`
	Tool   mcptypes.Tool `json:"tool"`
	Error  string        `json:"error,omitempty"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	global, args, err := parseGlobalFlags(os.Args[1:])
	if err != nil {
		fatal(err)
	}
	if global.Help || len(args) == 0 {
		printUsage()
		return
	}

	cfg, err := config.LoadWithCLI(global.ConfigArgs)
	if err != nil {
		fatal(err)
	}

	cmd := args[0]
	sub := ""
	if len(args) > 1 {
		sub = args[1]
	}

	switch cmd {
	case "status":
		ensureNoArgs(args[1:])
		runStatus(global)
	case "agents":
		runAgents(ctx, global, args[1:])
	case "tasks":
		runTasks(ctx, global, args[1:])
	case "traces":
		runTraces(ctx, global, args[1:])
	case "approvals":
		runApprovals(ctx, global, args[1:])
	case "mcp":
		runMCP(ctx, global, cfg, args[1:])
	case "help":
		printUsage()
	case "version":
		printVersion()
	default:
		if sub == "" {
			fatal(fmt.Errorf("unknown command %q", cmd))
		}
		fatal(fmt.Errorf("unknown command %q %q", cmd, sub))
	}
}

func parseGlobalFlags(args []string) (globalFlags, []string, error) {
	flags := globalFlags{
		GRPCAddr: getenv("KAIROS_GRPC_ADDR", defaultGRPCAddr),
		HTTPURL:  getenv("KAIROS_HTTP_URL", defaultHTTPURL),
		Timeout:  30 * time.Second,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return flags, args[i+1:], nil
		}
		if !strings.HasPrefix(arg, "-") {
			return flags, args[i:], nil
		}
		switch {
		case arg == "-h" || arg == "--help":
			flags.Help = true
			return flags, nil, nil
		case arg == "--json":
			flags.JSON = true
		case arg == "--config":
			if i+1 >= len(args) {
				return flags, nil, fmt.Errorf("missing value for --config")
			}
			flags.ConfigArgs = append(flags.ConfigArgs, arg, args[i+1])
			i++
		case strings.HasPrefix(arg, "--config="):
			flags.ConfigArgs = append(flags.ConfigArgs, arg)
		case arg == "--set":
			if i+1 >= len(args) {
				return flags, nil, fmt.Errorf("missing value for --set")
			}
			flags.ConfigArgs = append(flags.ConfigArgs, arg, args[i+1])
			i++
		case strings.HasPrefix(arg, "--set="):
			flags.ConfigArgs = append(flags.ConfigArgs, arg)
		case arg == "--grpc":
			if i+1 >= len(args) {
				return flags, nil, fmt.Errorf("missing value for --grpc")
			}
			flags.GRPCAddr = args[i+1]
			i++
		case strings.HasPrefix(arg, "--grpc="):
			flags.GRPCAddr = strings.TrimPrefix(arg, "--grpc=")
		case arg == "--http":
			if i+1 >= len(args) {
				return flags, nil, fmt.Errorf("missing value for --http")
			}
			flags.HTTPURL = args[i+1]
			i++
		case strings.HasPrefix(arg, "--http="):
			flags.HTTPURL = strings.TrimPrefix(arg, "--http=")
		case arg == "--timeout":
			if i+1 >= len(args) {
				return flags, nil, fmt.Errorf("missing value for --timeout")
			}
			value, err := time.ParseDuration(args[i+1])
			if err != nil {
				return flags, nil, fmt.Errorf("invalid --timeout: %w", err)
			}
			flags.Timeout = value
			i++
		case strings.HasPrefix(arg, "--timeout="):
			value, err := time.ParseDuration(strings.TrimPrefix(arg, "--timeout="))
			if err != nil {
				return flags, nil, fmt.Errorf("invalid --timeout: %w", err)
			}
			flags.Timeout = value
		default:
			return flags, nil, fmt.Errorf("unknown global flag %q", arg)
		}
	}
	return flags, nil, nil
}

func runStatus(flags globalFlags) {
	result := statusResult{
		Version:       "dev",
		GRPCAddr:      flags.GRPCAddr,
		HTTPURL:       flags.HTTPURL,
		GRPCReachable: checkTCP(flags.GRPCAddr),
		HTTPReachable: checkHTTP(flags.HTTPURL),
	}

	if flags.JSON {
		printJSON(result)
		return
	}

	fmt.Printf("Kairos CLI: %s\n", result.Version)
	fmt.Printf("gRPC: %s (reachable=%t)\n", result.GRPCAddr, result.GRPCReachable)
	fmt.Printf("HTTP: %s (reachable=%t)\n", result.HTTPURL, result.HTTPReachable)
}

func runAgents(ctx context.Context, flags globalFlags, args []string) {
	if len(args) == 0 || args[0] != "list" {
		fatal(fmt.Errorf("usage: kairos agents list --agent-card <url>"))
	}

	cmd := flag.NewFlagSet("agents list", flag.ContinueOnError)
	var cardURLs multiFlag
	cmd.Var(&cardURLs, "agent-card", "AgentCard base URL (repeatable)")
	if err := cmd.Parse(args[1:]); err != nil {
		fatal(err)
	}
	urls := append([]string{}, cardURLs...)
	urls = append(urls, splitList(getenv("KAIROS_AGENT_CARD_URLS", ""))...)
	urls = uniqueStrings(urls)
	if len(urls) == 0 {
		fatal(fmt.Errorf("no agent cards provided; use --agent-card or KAIROS_AGENT_CARD_URLS"))
	}

	ctx, cancel := context.WithTimeout(ctx, flags.Timeout)
	defer cancel()

	results := make([]agentResult, 0, len(urls))
	for _, baseURL := range urls {
		card, err := agentcard.Fetch(ctx, baseURL)
		res := agentResult{URL: baseURL, Card: card}
		if err != nil {
			res.Err = err.Error()
			res.Card = nil
		}
		results = append(results, res)
	}

	if flags.JSON {
		out := make([]map[string]any, 0, len(results))
		for _, res := range results {
			entry := map[string]any{
				"url": res.URL,
			}
			if res.Err != "" {
				entry["error"] = res.Err
				out = append(out, entry)
				continue
			}
			payload, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(res.Card)
			if err != nil {
				entry["error"] = err.Error()
			} else {
				entry["card"] = json.RawMessage(payload)
			}
			out = append(out, entry)
		}
		printJSON(out)
		return
	}
	for _, res := range results {
		if res.Err != "" {
			continue
		}
	}

	writer := newTabWriter()
	writeRow(writer, "NAME", "VERSION", "URL", "DESCRIPTION")
	for _, res := range results {
		if res.Err != "" {
			writeRow(writer, "ERROR", "-", res.URL, res.Err)
			continue
		}
		desc := strings.TrimSpace(res.Card.GetDescription())
		writeRow(writer, res.Card.GetName(), res.Card.GetVersion(), res.URL, desc)
	}
	_ = writer.Flush()
}

func runTasks(ctx context.Context, flags globalFlags, args []string) {
	if len(args) == 0 {
		fatal(errors.New("usage: kairos tasks <list|follow>"))
	}
	conn, err := dialGRPC(ctx, flags.GRPCAddr, flags.Timeout)
	if err != nil {
		fatal(err)
	}
	defer conn.Close()

	client := client.New(conn, client.WithTimeout(flags.Timeout))

	switch args[0] {
	case "list":
		cmd := flag.NewFlagSet("tasks list", flag.ContinueOnError)
		status := cmd.String("status", "", "Task status filter")
		contextID := cmd.String("context", "", "Context ID filter")
		pageSize := cmd.Int("page-size", 0, "Page size")
		pageToken := cmd.String("page-token", "", "Page token")
		history := cmd.Int("history-length", 0, "History length")
		lastUpdated := cmd.Int64("updated-after", 0, "Updated after (ms since epoch)")
		if err := cmd.Parse(args[1:]); err != nil {
			fatal(err)
		}
		state, err := parseTaskState(*status)
		if err != nil {
			fatal(err)
		}
		req := &a2av1.ListTasksRequest{
			ContextId:        *contextID,
			Status:           state,
			PageToken:        *pageToken,
			LastUpdatedAfter: *lastUpdated,
		}
		if *pageSize > 0 {
			size := int32(*pageSize)
			req.PageSize = &size
		}
		if *history > 0 {
			length := int32(*history)
			req.HistoryLength = &length
		}
		resp, err := client.ListTasks(ctx, req)
		if err != nil {
			fatal(err)
		}
		if flags.JSON {
			printProtoJSON(resp)
			return
		}
		writer := newTabWriter()
		writeRow(writer, "TASK_ID", "STATUS", "UPDATED", "MESSAGE")
		for _, task := range resp.GetTasks() {
			state := strings.ToLower(strings.TrimPrefix(task.GetStatus().GetState().String(), "TASK_STATE_"))
			updated := formatTimestamp(task.GetStatus().GetTimestamp())
			msg := truncateMessage(server.ExtractText(task.GetStatus().GetMessage()), 80)
			writeRow(writer, task.GetId(), state, updated, msg)
		}
		_ = writer.Flush()
		if resp.GetNextPageToken() != "" {
			fmt.Printf("next_page_token=%s\n", resp.GetNextPageToken())
		}
	case "follow":
		cmd := flag.NewFlagSet("tasks follow", flag.ContinueOnError)
		outPath := cmd.String("out", "", "Write JSON stream to file")
		if err := cmd.Parse(args[1:]); err != nil {
			fatal(err)
		}
		if cmd.NArg() < 1 {
			fatal(errors.New("usage: kairos tasks follow <task_id>"))
		}
		taskID := cmd.Arg(0)
		req := &a2av1.SubscribeToTaskRequest{Name: fmt.Sprintf("tasks/%s", taskID)}
		stream, err := client.SubscribeToTask(ctx, req)
		if err != nil {
			fatal(err)
		}
		var outWriter io.WriteCloser
		if strings.TrimSpace(*outPath) != "" {
			file, err := os.Create(*outPath)
			if err != nil {
				fatal(err)
			}
			outWriter = file
			defer func() { _ = outWriter.Close() }()
		}

		for {
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				fatal(err)
			}
			printStreamResponse(resp, flags.JSON)
			if outWriter != nil {
				writeJSONLine(outWriter, resp)
			}
		}
	case "cancel":
		cmd := flag.NewFlagSet("tasks cancel", flag.ContinueOnError)
		if err := cmd.Parse(args[1:]); err != nil {
			fatal(err)
		}
		if cmd.NArg() < 1 {
			fatal(errors.New("usage: kairos tasks cancel <task_id>"))
		}
		taskID := cmd.Arg(0)
		task, err := client.CancelTask(ctx, &a2av1.CancelTaskRequest{Name: fmt.Sprintf("tasks/%s", taskID)})
		if err != nil {
			fatal(err)
		}
		if flags.JSON {
			printProtoJSON(task)
			return
		}
		fmt.Printf("task %s status=%s\n", task.GetId(), task.GetStatus().GetState().String())
	case "retry":
		cmd := flag.NewFlagSet("tasks retry", flag.ContinueOnError)
		history := cmd.Int("history-length", 50, "History length to scan for the last user message")
		if err := cmd.Parse(args[1:]); err != nil {
			fatal(err)
		}
		if cmd.NArg() < 1 {
			fatal(errors.New("usage: kairos tasks retry <task_id>"))
		}
		taskID := cmd.Arg(0)
		length := int32(*history)
		task, err := client.GetTask(ctx, &a2av1.GetTaskRequest{Name: fmt.Sprintf("tasks/%s", taskID), HistoryLength: &length})
		if err != nil {
			fatal(err)
		}
		msg := lastUserMessage(task.GetHistory())
		if msg == nil {
			fatal(errors.New("no user message found to retry"))
		}
		retryMsg := cloneMessage(msg)
		retryMsg.TaskId = ""
		retryMsg.ContextId = ""
		resp, err := client.SendMessage(ctx, &a2av1.SendMessageRequest{Request: retryMsg})
		if err != nil {
			fatal(err)
		}
		if flags.JSON {
			printProtoJSON(resp.GetMsg())
			return
		}
		fmt.Printf("retry submitted: task_id=%s\n", resp.GetMsg().GetTaskId())
	default:
		fatal(fmt.Errorf("unknown tasks command %q", args[0]))
	}
}

func runTraces(ctx context.Context, flags globalFlags, args []string) {
	if len(args) == 0 || args[0] != "tail" {
		fatal(errors.New("usage: kairos traces tail --task <task_id>"))
	}
	cmd := flag.NewFlagSet("traces tail", flag.ContinueOnError)
	taskID := cmd.String("task", "", "Task ID to follow")
	outPath := cmd.String("out", "", "Write JSON stream to file")
	if err := cmd.Parse(args[1:]); err != nil {
		fatal(err)
	}
	if strings.TrimSpace(*taskID) == "" {
		fatal(errors.New("missing --task"))
	}
	conn, err := dialGRPC(ctx, flags.GRPCAddr, flags.Timeout)
	if err != nil {
		fatal(err)
	}
	defer conn.Close()
	client := client.New(conn, client.WithTimeout(flags.Timeout))
	req := &a2av1.SubscribeToTaskRequest{Name: fmt.Sprintf("tasks/%s", strings.TrimSpace(*taskID))}
	stream, err := client.SubscribeToTask(ctx, req)
	if err != nil {
		fatal(err)
	}
	var outWriter io.WriteCloser
	if strings.TrimSpace(*outPath) != "" {
		file, err := os.Create(*outPath)
		if err != nil {
			fatal(err)
		}
		outWriter = file
		defer func() { _ = outWriter.Close() }()
	}
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			fatal(err)
		}
		printStreamResponse(resp, flags.JSON)
		if outWriter != nil {
			writeJSONLine(outWriter, resp)
		}
	}
}

func runApprovals(ctx context.Context, flags globalFlags, args []string) {
	if len(args) == 0 {
		fatal(errors.New("usage: kairos approvals <list|approve|reject|tail>"))
	}
	client := httpjson.New(flags.HTTPURL)

	switch args[0] {
	case "list":
		cmd := flag.NewFlagSet("approvals list", flag.ContinueOnError)
		status := cmd.String("status", "", "Approval status filter")
		expiresBefore := cmd.String("expires-before", "", "Expiry cutoff (RFC3339 or ms since epoch)")
		if err := cmd.Parse(args[1:]); err != nil {
			fatal(err)
		}
		filter := server.ApprovalFilter{}
		if strings.TrimSpace(*status) != "" {
			filter.Status = server.ApprovalStatus(*status)
		}
		if strings.TrimSpace(*expiresBefore) != "" {
			value, err := parseTimeMillis(*expiresBefore)
			if err != nil {
				fatal(err)
			}
			filter.ExpiringBefore = time.UnixMilli(value).UTC()
		}
		records, err := client.ListApprovals(ctx, filter)
		if err != nil {
			fatal(err)
		}
		if flags.JSON {
			printJSON(records)
			return
		}
		writer := newTabWriter()
		writeRow(writer, "APPROVAL_ID", "STATUS", "EXPIRES_AT", "REASON")
		for _, record := range records {
			expiresAt := formatTime(record.ExpiresAt)
			writeRow(writer, record.ID, string(record.Status), expiresAt, record.Reason)
		}
		_ = writer.Flush()
	case "approve", "reject":
		cmd := flag.NewFlagSet("approvals action", flag.ContinueOnError)
		reason := cmd.String("reason", "", "Approval reason")
		if err := cmd.Parse(args[1:]); err != nil {
			fatal(err)
		}
		if cmd.NArg() < 1 {
			fatal(fmt.Errorf("usage: kairos approvals %s <approval_id>", args[0]))
		}
		id := cmd.Arg(0)
		var task *a2av1.Task
		var err error
		if args[0] == "approve" {
			task, err = client.ApproveApproval(ctx, id, *reason)
		} else {
			task, err = client.RejectApproval(ctx, id, *reason)
		}
		if err != nil {
			fatal(err)
		}
		if flags.JSON {
			printProtoJSON(task)
			return
		}
		fmt.Printf("task %s status=%s\n", task.GetId(), task.GetStatus().GetState().String())
	case "tail":
		cmd := flag.NewFlagSet("approvals tail", flag.ContinueOnError)
		status := cmd.String("status", "pending", "Approval status filter")
		interval := cmd.Duration("interval", 5*time.Second, "Polling interval")
		outPath := cmd.String("out", "", "Write JSON lines to file")
		if err := cmd.Parse(args[1:]); err != nil {
			fatal(err)
		}
		filter := server.ApprovalFilter{}
		if strings.TrimSpace(*status) != "" {
			filter.Status = server.ApprovalStatus(*status)
		}
		var outWriter io.WriteCloser
		if strings.TrimSpace(*outPath) != "" {
			file, err := os.Create(*outPath)
			if err != nil {
				fatal(err)
			}
			outWriter = file
			defer func() { _ = outWriter.Close() }()
		}
		seen := map[string]struct{}{}
		for {
			records, err := client.ListApprovals(ctx, filter)
			if err != nil {
				fatal(err)
			}
			sort.Slice(records, func(i, j int) bool {
				return records[i].UpdatedAt.Before(records[j].UpdatedAt)
			})
			for _, record := range records {
				if _, ok := seen[record.ID]; ok {
					continue
				}
				seen[record.ID] = struct{}{}
				if flags.JSON {
					printJSON(record)
				} else {
					fmt.Printf("approval %s status=%s\n", record.ID, record.Status)
				}
				if outWriter != nil {
					writeJSONLine(outWriter, record)
				}
			}
			select {
			case <-time.After(*interval):
			case <-ctx.Done():
				return
			}
		}
	default:
		fatal(fmt.Errorf("unknown approvals command %q", args[0]))
	}
}

func runMCP(ctx context.Context, flags globalFlags, cfg *config.Config, args []string) {
	if len(args) == 0 || args[0] != "list" {
		fatal(errors.New("usage: kairos mcp list"))
	}
	ensureNoArgs(args[1:])
	if cfg == nil {
		fatal(errors.New("config not loaded"))
	}
	if len(cfg.MCP.Servers) == 0 {
		fmt.Println("no mcp servers configured")
		return
	}

	serverNames := make([]string, 0, len(cfg.MCP.Servers))
	for name := range cfg.MCP.Servers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	results := make([]mcpToolResult, 0)
	for _, name := range serverNames {
		srv := cfg.MCP.Servers[name]
		client, err := newMCPClient(name, srv)
		if err != nil {
			results = append(results, mcpToolResult{Server: name, Error: err.Error()})
			continue
		}
		ctx, cancel := context.WithTimeout(ctx, flags.Timeout)
		tools, err := client.ListTools(ctx)
		cancel()
		if err != nil {
			results = append(results, mcpToolResult{Server: name, Error: err.Error()})
			_ = client.Close()
			continue
		}
		for _, tool := range tools {
			results = append(results, mcpToolResult{Server: name, Tool: tool})
		}
		_ = client.Close()
	}

	if flags.JSON {
		printJSON(results)
		return
	}
	writer := newTabWriter()
	writeRow(writer, "SERVER", "TOOL", "DESCRIPTION")
	for _, res := range results {
		if res.Error != "" {
			writeRow(writer, res.Server, "ERROR", res.Error)
			continue
		}
		desc := strings.TrimSpace(res.Tool.Description)
		writeRow(writer, res.Server, res.Tool.Name, desc)
	}
	_ = writer.Flush()
}

func newMCPClient(name string, cfg config.MCPServerConfig) (*kairosmcp.Client, error) {
	opts := []kairosmcp.ClientOption{kairosmcp.WithServerName(name)}
	if cfg.TimeoutSeconds != nil {
		opts = append(opts, kairosmcp.WithTimeout(time.Duration(*cfg.TimeoutSeconds)*time.Second))
	}
	if cfg.RetryCount != nil || cfg.RetryBackoffMs != nil {
		retries := 0
		backoff := 0 * time.Millisecond
		if cfg.RetryCount != nil {
			retries = *cfg.RetryCount
		}
		if cfg.RetryBackoffMs != nil {
			backoff = time.Duration(*cfg.RetryBackoffMs) * time.Millisecond
		}
		opts = append(opts, kairosmcp.WithRetry(retries, backoff))
	}
	if cfg.CacheTTLSeconds != nil {
		opts = append(opts, kairosmcp.WithToolCacheTTL(time.Duration(*cfg.CacheTTLSeconds)*time.Second))
	}

	transport := strings.ToLower(strings.TrimSpace(cfg.Transport))
	if transport == "" || transport == "stdio" {
		if strings.TrimSpace(cfg.Command) == "" {
			return nil, fmt.Errorf("mcp server %q missing command", name)
		}
		return kairosmcp.NewClientWithStdioProtocol(cfg.Command, cfg.Args, cfg.ProtocolVersion, opts...)
	}
	if transport == "http" {
		if strings.TrimSpace(cfg.URL) == "" {
			return nil, fmt.Errorf("mcp server %q missing url", name)
		}
		return kairosmcp.NewClientWithStreamableHTTPProtocol(cfg.URL, cfg.ProtocolVersion, opts...)
	}
	return nil, fmt.Errorf("mcp server %q has unsupported transport %q", name, cfg.Transport)
}

func dialGRPC(ctx context.Context, addr string, timeout time.Duration) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
}

func parseTaskState(value string) (a2av1.TaskState, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return a2av1.TaskState_TASK_STATE_UNSPECIFIED, nil
	}
	states := map[string]a2av1.TaskState{
		"submitted":      a2av1.TaskState_TASK_STATE_SUBMITTED,
		"working":        a2av1.TaskState_TASK_STATE_WORKING,
		"completed":      a2av1.TaskState_TASK_STATE_COMPLETED,
		"failed":         a2av1.TaskState_TASK_STATE_FAILED,
		"cancelled":      a2av1.TaskState_TASK_STATE_CANCELLED,
		"canceled":       a2av1.TaskState_TASK_STATE_CANCELLED,
		"input_required": a2av1.TaskState_TASK_STATE_INPUT_REQUIRED,
		"input-required": a2av1.TaskState_TASK_STATE_INPUT_REQUIRED,
		"rejected":       a2av1.TaskState_TASK_STATE_REJECTED,
		"auth_required":  a2av1.TaskState_TASK_STATE_AUTH_REQUIRED,
		"auth-required":  a2av1.TaskState_TASK_STATE_AUTH_REQUIRED,
	}
	if state, ok := states[value]; ok {
		return state, nil
	}
	return a2av1.TaskState_TASK_STATE_UNSPECIFIED, fmt.Errorf("unknown task status %q", value)
}

func lastUserMessage(history []*a2av1.Message) *a2av1.Message {
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		if msg == nil {
			continue
		}
		if msg.GetRole() == a2av1.Role_ROLE_USER {
			return msg
		}
	}
	return nil
}

func cloneMessage(message *a2av1.Message) *a2av1.Message {
	if message == nil {
		return nil
	}
	return proto.Clone(message).(*a2av1.Message)
}

func printStreamResponse(resp *a2av1.StreamResponse, asJSON bool) {
	if asJSON {
		payload, _ := protojson.Marshal(resp)
		fmt.Println(string(payload))
		return
	}
	if resp == nil {
		return
	}
	switch payload := resp.Payload.(type) {
	case *a2av1.StreamResponse_StatusUpdate:
		update := payload.StatusUpdate
		state := update.GetStatus().GetState().String()
		msg := server.ExtractText(update.GetStatus().GetMessage())
		eventType, payloadSummary := extractEventMetadata(update.GetMetadata())
		traceID := extractTraceID(update.GetStatus().GetMessage())
		line := fmt.Sprintf("status=%s", strings.ToLower(strings.TrimPrefix(state, "TASK_STATE_")))
		if eventType != "" {
			line += fmt.Sprintf(" event=%s", eventType)
		}
		if traceID != "" {
			line += fmt.Sprintf(" trace_id=%s", traceID)
		}
		if msg != "" {
			line += fmt.Sprintf(" msg=%s", msg)
		}
		fmt.Println(line)
		if payloadSummary != "" {
			fmt.Printf("payload=%s\n", payloadSummary)
		}
	case *a2av1.StreamResponse_Task:
		fmt.Printf("task %s\n", payload.Task.GetId())
	case *a2av1.StreamResponse_Msg:
		text := server.ExtractText(payload.Msg)
		if text != "" {
			fmt.Printf("msg=%s\n", text)
		}
	default:
		fmt.Println("event received")
	}
}

func extractEventMetadata(meta *structpb.Struct) (string, string) {
	if meta == nil {
		return "", ""
	}
	fields := meta.GetFields()
	if fields == nil {
		return "", ""
	}
	var eventType string
	if v, ok := fields["event_type"]; ok {
		eventType = v.GetStringValue()
	}
	var payloadSummary string
	if v, ok := fields["payload"]; ok {
		if v.GetStructValue() != nil {
			if payload, err := protojson.Marshal(v.GetStructValue()); err == nil {
				payloadSummary = string(payload)
			} else {
				payloadSummary = v.String()
			}
		} else {
			payloadSummary = v.String()
		}
	}
	return eventType, payloadSummary
}

func extractTraceID(message *a2av1.Message) string {
	if message == nil {
		return ""
	}
	meta := message.GetMetadata()
	if meta == nil {
		return ""
	}
	fields := meta.GetFields()
	if fields == nil {
		return ""
	}
	if v, ok := fields["trace_id"]; ok {
		return v.GetStringValue()
	}
	return ""
}

func parseTimeMillis(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("empty time value")
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts.UnixMilli(), nil
	}
	var millis int64
	if _, err := fmt.Sscan(value, &millis); err == nil {
		return millis, nil
	}
	return 0, fmt.Errorf("invalid time value %q", value)
}

func checkTCP(addr string) bool {
	if strings.TrimSpace(addr) == "" {
		return false
	}
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func checkHTTP(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := parsed.Host
	if host == "" {
		host = parsed.Path
	}
	if host == "" {
		return false
	}
	if !strings.Contains(host, ":") {
		if parsed.Scheme == "https" {
			host += ":443"
		} else {
			host += ":80"
		}
	}
	return checkTCP(host)
}

func printJSON(value any) {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fatal(err)
	}
	fmt.Println(string(payload))
}

func printProtoJSON(msg proto.Message) {
	payload, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(msg)
	if err != nil {
		fatal(err)
	}
	fmt.Println(string(payload))
}

func writeJSONLine(writer io.Writer, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		fatal(err)
	}
	_, _ = writer.Write(append(payload, '\n'))
}

func newTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
}

func writeRow(writer *tabwriter.Writer, cols ...string) {
	for i, col := range cols {
		cols[i] = normalizeCell(col)
	}
	fmt.Fprintln(writer, strings.Join(cols, "\t"))
}

func normalizeCell(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return strings.Join(strings.Fields(value), " ")
}

func truncateMessage(value string, limit int) string {
	value = normalizeCell(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil || ts.Seconds == 0 {
		return "-"
	}
	return ts.AsTime().Format(time.RFC3339)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.UTC().Format(time.RFC3339)
}

func printVersion() {
	fmt.Println("dev")
}

func printUsage() {
	fmt.Println(`Kairos CLI (Phase 8.1 MVP)

Usage:
  kairos [global flags] <command> [args]

Global flags:
  --config <path>      Path to settings.json
  --set key=value      Override config (repeatable)
  --grpc <addr>        A2A gRPC address (default localhost:8080)
  --http <url>         A2A HTTP+JSON base URL (default http://localhost:8080)
  --timeout <dur>      Request timeout (default 30s)
  --json               JSON output

Commands:
  status
  agents list --agent-card <url>
  tasks list [--status <state>] [--context <id>] [--page-size N] [--page-token T]
  tasks follow <task_id> [--out <path>]
  tasks cancel <task_id>
  tasks retry <task_id> [--history-length N]
  traces tail --task <task_id> [--out <path>]
  approvals list [--status <status>] [--expires-before <time>]
  approvals approve <id> [--reason <text>]
  approvals reject <id> [--reason <text>]
  approvals tail [--status <status>] [--interval 5s] [--out <path>]
  mcp list
`)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func ensureNoArgs(args []string) {
	if len(args) > 0 {
		fatal(fmt.Errorf("unexpected args: %v", args))
	}
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func splitList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
