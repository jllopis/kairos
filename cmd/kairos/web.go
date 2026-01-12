package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jllopis/kairos/pkg/a2a/agentcard"
	"github.com/jllopis/kairos/pkg/a2a/client"
	httpjson "github.com/jllopis/kairos/pkg/a2a/httpjson/client"
	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/discovery"
)

const defaultWebAddr = ":8088"

//go:embed web/templates/*.html web/static/*
var webFS embed.FS

var (
	webPartials = template.Must(template.New("partials").Funcs(template.FuncMap{
		"lower": strings.ToLower,
	}).ParseFS(webFS, "web/templates/agents_list.html", "web/templates/tasks_list.html", "web/templates/approvals_list.html"))
	pageTemplates = map[string]*template.Template{}
)

type webServer struct {
	flags     globalFlags
	cfg       *config.Config
	http      *httpjson.Client
	agentURLs []string
}

type agentRow struct {
	Name        string
	Version     string
	URL         string
	Description string
}

type listAgentsData struct {
	Agents []agentRow
	Error  string
	Empty  bool
}

type taskRow struct {
	ID        string
	Status    string
	UpdatedAt string
	Message   string
}

type listTasksData struct {
	Tasks []taskRow
	Error string
	Empty bool
}

type approvalRow struct {
	ID        string
	Status    string
	ExpiresAt string
	Reason    string
	Pending   bool
}

type listApprovalsData struct {
	Approvals []approvalRow
	Error     string
	Empty     bool
}

type taskDetail struct {
	TaskID  string
	Status  string
	History []taskHistoryRow
}

type taskHistoryRow struct {
	Role      string
	Timestamp string
	Text      string
}

type pageData struct {
	Title string
	Data  any
}

func runWeb(ctx context.Context, flags globalFlags, cfg *config.Config) {
	server := &webServer{
		flags:     flags,
		cfg:       cfg,
		http:      httpjson.New(flags.HTTPURL),
		agentURLs: loadAgentURLs(),
	}
	mux := http.NewServeMux()

	staticFS, err := fs.Sub(webFS, "web/static")
	if err != nil {
		fatal(err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("/", server.handleRoot)
	mux.HandleFunc("/agents", server.handleAgentsPage)
	mux.HandleFunc("/tasks", server.handleTasksPage)
	mux.HandleFunc("/tasks/", server.handleTaskDetail)
	mux.HandleFunc("/approvals", server.handleApprovalsPage)

	mux.HandleFunc("/ui/agents/list", server.handleAgentsList)
	mux.HandleFunc("/ui/tasks/list", server.handleTasksList)
	mux.HandleFunc("/ui/approvals/list", server.handleApprovalsList)
	mux.HandleFunc("/ui/approvals/", server.handleApprovalAction)

	serverAddr := flags.WebAddr
	if strings.TrimSpace(serverAddr) == "" {
		serverAddr = defaultWebAddr
	}

	httpServer := &http.Server{
		Addr:              serverAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	displayAddr := serverAddr
	if strings.HasPrefix(displayAddr, ":") {
		displayAddr = "localhost" + displayAddr
	}
	fmt.Printf("Kairos UI listening on http://%s\n", displayAddr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fatal(err)
	}
}

func (s *webServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/tasks", http.StatusFound)
}

func (s *webServer) handleAgentsPage(w http.ResponseWriter, _ *http.Request) {
	renderPage(w, "agents", "Agents", nil)
}

func (s *webServer) handleTasksPage(w http.ResponseWriter, _ *http.Request) {
	renderPage(w, "tasks", "Tasks", nil)
}

func (s *webServer) handleApprovalsPage(w http.ResponseWriter, _ *http.Request) {
	renderPage(w, "approvals", "Approvals", nil)
}

func (s *webServer) handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/tasks/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		http.NotFound(w, r)
		return
	}
	taskID := parts[0]
	length := int32(50)
	var task *a2av1.Task
	if err := s.withGRPC(r.Context(), func(c *client.Client) error {
		var err error
		task, err = c.GetTask(r.Context(), &a2av1.GetTaskRequest{Name: fmt.Sprintf("tasks/%s", taskID), HistoryLength: &length})
		return err
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	data := taskDetail{
		TaskID: task.GetId(),
		Status: strings.ToLower(strings.TrimPrefix(task.GetStatus().GetState().String(), "TASK_STATE_")),
	}
	for _, msg := range task.GetHistory() {
		data.History = append(data.History, taskHistoryRow{
			Role:      msg.GetRole().String(),
			Timestamp: "-",
			Text:      normalizeCell(server.ExtractText(msg)),
		})
	}
	renderPage(w, "task_detail", fmt.Sprintf("Task %s", taskID), data)
}

func (s *webServer) handleAgentsList(w http.ResponseWriter, r *http.Request) {
	data := listAgentsData{}
	ctx, cancel := context.WithTimeout(r.Context(), s.flags.Timeout)
	defer cancel()
	providers := discovery.BuildProviders(s.cfg, s.agentURLs)
	resolver, err := discovery.NewResolver(providers...)
	if err != nil {
		data.Error = err.Error()
		data.Empty = true
		renderPartial(w, "agents_list", data)
		return
	}
	entries, err := resolver.Resolve(ctx)
	if err != nil {
		data.Error = err.Error()
		renderPartial(w, "agents_list", data)
		return
	}
	discovery.SortByName(entries)
	for _, entry := range entries {
		if entry.AgentCardURL == "" {
			data.Agents = append(data.Agents, agentRow{
				Name:        entry.Name,
				Version:     "",
				URL:         entry.AgentCardURL,
				Description: "",
			})
			continue
		}
		card, err := agentcard.Fetch(ctx, entry.AgentCardURL)
		if err != nil {
			data.Error = err.Error()
			continue
		}
		data.Agents = append(data.Agents, agentRow{
			Name:        card.GetName(),
			Version:     card.GetVersion(),
			URL:         entry.AgentCardURL,
			Description: normalizeCell(card.GetDescription()),
		})
	}
	if len(data.Agents) == 0 {
		data.Empty = true
	}
	renderPartial(w, "agents_list", data)
}

func (s *webServer) handleTasksList(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	contextID := strings.TrimSpace(r.URL.Query().Get("context"))
	state, err := parseTaskState(status)
	data := listTasksData{}
	if err != nil {
		data.Error = err.Error()
		renderPartial(w, "tasks_list", data)
		return
	}
	var resp *a2av1.ListTasksResponse
	if err := s.withGRPC(r.Context(), func(c *client.Client) error {
		var err error
		resp, err = c.ListTasks(r.Context(), &a2av1.ListTasksRequest{ContextId: contextID, Status: state})
		return err
	}); err != nil {
		data.Error = err.Error()
		renderPartial(w, "tasks_list", data)
		return
	}
	for _, task := range resp.GetTasks() {
		data.Tasks = append(data.Tasks, taskRow{
			ID:        task.GetId(),
			Status:    strings.ToLower(strings.TrimPrefix(task.GetStatus().GetState().String(), "TASK_STATE_")),
			UpdatedAt: formatTimestamp(task.GetStatus().GetTimestamp()),
			Message:   truncateMessage(server.ExtractText(task.GetStatus().GetMessage()), 80),
		})
	}
	if len(data.Tasks) == 0 {
		data.Empty = true
	}
	renderPartial(w, "tasks_list", data)
}

func (s *webServer) handleApprovalsList(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	filter := server.ApprovalFilter{}
	if status != "" {
		filter.Status = server.ApprovalStatus(status)
	}
	records, err := s.http.ListApprovals(r.Context(), filter)
	data := listApprovalsData{}
	if err != nil {
		data.Error = err.Error()
		renderPartial(w, "approvals_list", data)
		return
	}
	for _, record := range records {
		data.Approvals = append(data.Approvals, approvalRow{
			ID:        record.ID,
			Status:    string(record.Status),
			ExpiresAt: formatTime(record.ExpiresAt),
			Reason:    normalizeCell(record.Reason),
			Pending:   record.Status == server.ApprovalStatusPending,
		})
	}
	if len(data.Approvals) == 0 {
		data.Empty = true
	}
	sort.Slice(data.Approvals, func(i, j int) bool {
		return data.Approvals[i].ID < data.Approvals[j].ID
	})
	renderPartial(w, "approvals_list", data)
}

func (s *webServer) handleApprovalAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/ui/approvals/")
	if path == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	id := strings.TrimSuffix(path, ":approve")
	if id != path {
		_, err := s.http.ApproveApproval(r.Context(), id, "approved via UI")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.handleApprovalsList(w, r)
		return
	}
	id = strings.TrimSuffix(path, ":reject")
	if id != path {
		_, err := s.http.RejectApproval(r.Context(), id, "rejected via UI")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.handleApprovalsList(w, r)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func renderPage(w http.ResponseWriter, pageName string, title string, data any) {
	tmpl, ok := pageTemplates[pageName]
	if !ok {
		http.Error(w, "page template not found", http.StatusInternalServerError)
		return
	}
	payload := pageData{Title: title, Data: data}
	if err := tmpl.ExecuteTemplate(w, "layout", payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderPartial(w http.ResponseWriter, name string, data any) {
	if err := webPartials.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *webServer) withGRPC(ctx context.Context, fn func(*client.Client) error) error {
	ctx, cancel := context.WithTimeout(ctx, s.flags.Timeout)
	defer cancel()
	conn, err := dialGRPC(ctx, s.flags.GRPCAddr, s.flags.Timeout)
	if err != nil {
		return err
	}
	defer conn.Close()
	return fn(client.New(conn, client.WithTimeout(s.flags.Timeout)))
}

func loadAgentURLs() []string {
	urls := splitList(getenv("KAIROS_AGENT_CARD_URLS", ""))
	return uniqueStrings(urls)
}

func init() {
	pageTemplates["agents"] = mustPageTemplate("web/templates/layout.html", "web/templates/agents.html")
	pageTemplates["tasks"] = mustPageTemplate("web/templates/layout.html", "web/templates/tasks.html")
	pageTemplates["approvals"] = mustPageTemplate("web/templates/layout.html", "web/templates/approvals.html")
	pageTemplates["task_detail"] = mustPageTemplate("web/templates/layout.html", "web/templates/task_detail.html")
}

func mustPageTemplate(layout string, page string) *template.Template {
	tmpl, err := template.New("page").Funcs(template.FuncMap{
		"lower": strings.ToLower,
	}).ParseFS(webFS, layout, page)
	if err != nil {
		panic(err)
	}
	return tmpl
}
