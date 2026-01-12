package httpjson

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Server exposes an HTTP+JSON binding for A2A handlers.
type Server struct {
	Handler server.Handler
}

// New creates a new HTTP+JSON server wrapper.
func New(handler server.Handler) *Server {
	return &Server{Handler: handler}
}

// ServeHTTP routes HTTP+JSON requests to the A2A handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.Handler == nil {
		writeError(w, status.Error(codes.Unimplemented, "handler not configured"))
		return
	}
	segments := normalizePath(r.URL.Path)
	if len(segments) == 0 {
		http.NotFound(w, r)
		return
	}
	switch segments[0] {
	case "message:send":
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		s.handleSendMessage(w, r)
		return
	case "message:stream":
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		s.handleSendStreamingMessage(w, r)
		return
	case "tasks":
		s.handleTasks(w, r, segments)
		return
	case "extendedAgentCard":
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		s.handleExtendedAgentCard(w, r)
		return
	default:
		http.NotFound(w, r)
		return
	}
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	req := &a2av1.SendMessageRequest{}
	if err := decodeProtoJSON(r, req); err != nil {
		writeError(w, err)
		return
	}
	resp, err := s.Handler.SendMessage(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeProtoJSON(w, resp)
}

func (s *Server) handleSendStreamingMessage(w http.ResponseWriter, r *http.Request) {
	req := &a2av1.SendMessageRequest{}
	if err := decodeProtoJSON(r, req); err != nil {
		writeError(w, err)
		return
	}
	writer, ok := w.(http.Flusher)
	if !ok {
		writeError(w, status.Error(codes.Internal, "streaming not supported"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	stream := &sseStream{ctx: r.Context(), w: w, f: writer}
	if err := s.Handler.SendStreamingMessage(req, stream); err != nil {
		writeError(w, err)
		return
	}
}

func (s *Server) handleExtendedAgentCard(w http.ResponseWriter, r *http.Request) {
	resp, err := s.Handler.GetExtendedAgentCard(r.Context(), &a2av1.GetExtendedAgentCardRequest{})
	if err != nil {
		writeError(w, err)
		return
	}
	writeProtoJSON(w, resp)
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		s.handleListTasks(w, r)
		return
	}
	name := fmt.Sprintf("tasks/%s", segments[1])
	switch {
	case strings.HasSuffix(segments[1], ":cancel"):
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		name = strings.TrimSuffix(name, ":cancel")
		s.handleCancelTask(w, r, name)
	case strings.HasSuffix(segments[1], ":subscribe"):
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		name = strings.TrimSuffix(name, ":subscribe")
		s.handleSubscribeTask(w, r, name)
	case len(segments) >= 3 && segments[2] == "pushNotificationConfigs":
		s.handlePushConfigs(w, r, segments)
	default:
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		s.handleGetTask(w, r, name)
	}
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request, name string) {
	req := &a2av1.GetTaskRequest{Name: name}
	if history := r.URL.Query().Get("historyLength"); history != "" {
		if value, err := strconv.ParseInt(history, 10, 32); err == nil {
			parsed := int32(value)
			req.HistoryLength = &parsed
		}
	}
	resp, err := s.Handler.GetTask(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeProtoJSON(w, resp)
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	req := &a2av1.ListTasksRequest{
		ContextId:        query.Get("contextId"),
		PageToken:        query.Get("pageToken"),
		LastUpdatedAfter: parseInt64(query.Get("lastUpdatedAfter")),
	}
	if statusValue := parseTaskState(query.Get("status")); statusValue != a2av1.TaskState_TASK_STATE_UNSPECIFIED {
		req.Status = statusValue
	}
	if pageSize := parseInt32(query.Get("pageSize")); pageSize != nil {
		req.PageSize = pageSize
	}
	if history := parseInt32(query.Get("historyLength")); history != nil {
		req.HistoryLength = history
	}
	if include := parseBool(query.Get("includeArtifacts")); include != nil {
		req.IncludeArtifacts = include
	}
	resp, err := s.Handler.ListTasks(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeProtoJSON(w, resp)
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request, name string) {
	req := &a2av1.CancelTaskRequest{Name: name}
	if r.Body != nil {
		defer r.Body.Close()
		body, _ := io.ReadAll(r.Body)
		if len(body) > 0 {
			_ = protojson.Unmarshal(body, req)
		}
	}
	resp, err := s.Handler.CancelTask(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeProtoJSON(w, resp)
}

func (s *Server) handleSubscribeTask(w http.ResponseWriter, r *http.Request, name string) {
	writer, ok := w.(http.Flusher)
	if !ok {
		writeError(w, status.Error(codes.Internal, "streaming not supported"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	req := &a2av1.SubscribeToTaskRequest{Name: name}
	stream := &sseStream{ctx: r.Context(), w: w, f: writer}
	if err := s.Handler.SubscribeToTask(req, stream); err != nil {
		writeError(w, err)
		return
	}
}

func (s *Server) handlePushConfigs(w http.ResponseWriter, r *http.Request, segments []string) {
	handler, ok := s.Handler.(server.PushNotificationHandler)
	if !ok {
		writeError(w, status.Error(codes.Unimplemented, "push notifications not supported"))
		return
	}
	taskID := segments[1]
	parent := fmt.Sprintf("tasks/%s", taskID)
	switch r.Method {
	case http.MethodPost:
		config := &a2av1.TaskPushNotificationConfig{}
		if err := decodeProtoJSON(r, config); err != nil {
			writeError(w, err)
			return
		}
		req := &a2av1.SetTaskPushNotificationConfigRequest{
			Parent: parent + "/pushNotificationConfigs",
			Config: config,
		}
		resp, err := handler.SetTaskPushNotificationConfig(r.Context(), req)
		if err != nil {
			writeError(w, err)
			return
		}
		writeProtoJSON(w, resp)
	case http.MethodGet:
		if len(segments) == 3 {
			req := &a2av1.ListTaskPushNotificationConfigRequest{Parent: parent}
			resp, err := handler.ListTaskPushNotificationConfig(r.Context(), req)
			if err != nil {
				writeError(w, err)
				return
			}
			writeProtoJSON(w, resp)
			return
		}
		name := fmt.Sprintf("%s/pushNotificationConfigs/%s", parent, segments[3])
		req := &a2av1.GetTaskPushNotificationConfigRequest{Name: name}
		resp, err := handler.GetTaskPushNotificationConfig(r.Context(), req)
		if err != nil {
			writeError(w, err)
			return
		}
		writeProtoJSON(w, resp)
	case http.MethodDelete:
		if len(segments) < 4 {
			http.NotFound(w, r)
			return
		}
		name := fmt.Sprintf("%s/pushNotificationConfigs/%s", parent, segments[3])
		req := &a2av1.DeleteTaskPushNotificationConfigRequest{Name: name}
		if _, err := handler.DeleteTaskPushNotificationConfig(r.Context(), req); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

func decodeProtoJSON(r *http.Request, msg proto.Message) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return status.Error(codes.InvalidArgument, "invalid body")
	}
	if len(body) == 0 {
		return status.Error(codes.InvalidArgument, "empty body")
	}
	if err := protojson.Unmarshal(body, msg); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	return nil
}

func writeProtoJSON(w http.ResponseWriter, msg proto.Message) {
	opts := protojson.MarshalOptions{Indent: "  "}
	payload, err := opts.Marshal(msg)
	if err != nil {
		writeError(w, status.Error(codes.Internal, err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}

func writeError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		st = status.New(codes.Unknown, err.Error())
	}
	code := httpStatusFromCode(st.Code())
	body := map[string]interface{}{
		"type":   "about:blank",
		"title":  st.Code().String(),
		"detail": st.Message(),
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func httpStatusFromCode(code codes.Code) int {
	switch code {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.NotFound:
		return http.StatusNotFound
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.FailedPrecondition:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func normalizePath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	segments := strings.Split(path, "/")
	if len(segments) > 1 && !isRootPath(segments[0]) && isRootPath(segments[1]) {
		return segments[1:]
	}
	return segments
}

func isRootPath(value string) bool {
	switch value {
	case "message:send", "message:stream", "tasks", "extendedAgentCard":
		return true
	default:
		return false
	}
}

func parseInt32(value string) *int32 {
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return nil
	}
	out := int32(parsed)
	return &out
}

func parseInt64(value string) int64 {
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func parseBool(value string) *bool {
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil
	}
	return &parsed
}

func parseTaskState(value string) a2av1.TaskState {
	value = strings.TrimSpace(strings.ToUpper(value))
	if value == "" {
		return a2av1.TaskState_TASK_STATE_UNSPECIFIED
	}
	if !strings.HasPrefix(value, "TASK_STATE_") {
		value = "TASK_STATE_" + value
	}
	if mapped, ok := a2av1.TaskState_value[value]; ok {
		return a2av1.TaskState(mapped)
	}
	return a2av1.TaskState_TASK_STATE_UNSPECIFIED
}

type sseStream struct {
	ctx context.Context
	w   http.ResponseWriter
	f   http.Flusher
}

func (s *sseStream) Context() context.Context {
	return s.ctx
}

func (s *sseStream) Send(resp *a2av1.StreamResponse) error {
	payload, err := protojson.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := s.w.Write([]byte("data: ")); err != nil {
		return err
	}
	if _, err := s.w.Write(payload); err != nil {
		return err
	}
	if _, err := s.w.Write([]byte("\n\n")); err != nil {
		return err
	}
	s.f.Flush()
	return nil
}

func (s *sseStream) SendHeader(metadata.MD) error { return nil }
func (s *sseStream) SetHeader(metadata.MD) error  { return nil }
func (s *sseStream) SetTrailer(metadata.MD)       {}
func (s *sseStream) SendMsg(any) error            { return nil }
func (s *sseStream) RecvMsg(any) error            { return nil }
