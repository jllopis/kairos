package jsonrpc

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Server exposes the JSON-RPC binding for A2A handlers.
type Server struct {
	Handler server.Handler
}

// New creates a new JSON-RPC server wrapper.
func New(handler server.Handler) *Server {
	return &Server{Handler: handler}
}

// ServeHTTP handles JSON-RPC 2.0 requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.Handler == nil {
		writeError(w, rpcError{Code: -32001, Message: "handler not configured"})
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, rpcError{Code: -32700, Message: "invalid json"})
		return
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		writeError(w, rpcError{Code: -32600, Message: "invalid request"})
		return
	}
	switch req.Method {
	case "SendMessage":
		s.handleSendMessage(w, r, req)
	case "SendStreamingMessage":
		s.handleSendStreamingMessage(w, r, req)
	case "GetTask":
		s.handleGetTask(w, r, req)
	case "ListTasks":
		s.handleListTasks(w, r, req)
	case "CancelTask":
		s.handleCancelTask(w, r, req)
	case "SubscribeToTask":
		s.handleSubscribe(w, r, req)
	case "GetExtendedAgentCard":
		s.handleExtendedAgentCard(w, r, req)
	case "GetApproval":
		s.handleGetApproval(w, r, req)
	case "ListApprovals":
		s.handleListApprovals(w, r, req)
	case "ApproveApproval":
		s.handleApproveApproval(w, r, req)
	case "RejectApproval":
		s.handleRejectApproval(w, r, req)
	default:
		writeError(w, rpcError{Code: -32601, Message: "method not found"})
	}
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	payload := &a2av1.SendMessageRequest{}
	if err := decodeParams(req.Params, payload); err != nil {
		writeError(w, rpcError{Code: -32602, Message: err.Error()})
		return
	}
	resp, err := s.Handler.SendMessage(r.Context(), payload)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeResult(w, req.ID, resp)
}

func (s *Server) handleSendStreamingMessage(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	payload := &a2av1.SendMessageRequest{}
	if err := decodeParams(req.Params, payload); err != nil {
		writeError(w, rpcError{Code: -32602, Message: err.Error()})
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeRPCError(w, status.Error(codes.Internal, "streaming not supported"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	stream := &rpcStream{ctx: r.Context(), w: w, f: flusher, id: req.ID}
	if err := s.Handler.SendStreamingMessage(payload, stream); err != nil {
		writeRPCError(w, err)
		return
	}
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	payload := &a2av1.GetTaskRequest{}
	if err := decodeParams(req.Params, payload); err != nil {
		writeError(w, rpcError{Code: -32602, Message: err.Error()})
		return
	}
	resp, err := s.Handler.GetTask(r.Context(), payload)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeResult(w, req.ID, resp)
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	payload := &a2av1.ListTasksRequest{}
	if err := decodeParams(req.Params, payload); err != nil {
		writeError(w, rpcError{Code: -32602, Message: err.Error()})
		return
	}
	resp, err := s.Handler.ListTasks(r.Context(), payload)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeResult(w, req.ID, resp)
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	payload := &a2av1.CancelTaskRequest{}
	if err := decodeParams(req.Params, payload); err != nil {
		writeError(w, rpcError{Code: -32602, Message: err.Error()})
		return
	}
	resp, err := s.Handler.CancelTask(r.Context(), payload)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeResult(w, req.ID, resp)
}

func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	payload := &a2av1.SubscribeToTaskRequest{}
	if err := decodeParams(req.Params, payload); err != nil {
		writeError(w, rpcError{Code: -32602, Message: err.Error()})
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeRPCError(w, status.Error(codes.Internal, "streaming not supported"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	stream := &rpcStream{ctx: r.Context(), w: w, f: flusher, id: req.ID}
	if err := s.Handler.SubscribeToTask(payload, stream); err != nil {
		writeRPCError(w, err)
		return
	}
}

func (s *Server) handleExtendedAgentCard(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	resp, err := s.Handler.GetExtendedAgentCard(r.Context(), &a2av1.GetExtendedAgentCardRequest{})
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeResult(w, req.ID, resp)
}

func (s *Server) handleGetApproval(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	handler, ok := s.Handler.(server.ApprovalHandler)
	if !ok {
		writeRPCError(w, status.Error(codes.Unimplemented, "approvals not supported"))
		return
	}
	var payload struct {
		ID string `json:"id"`
	}
	if err := decodeJSONParams(req.Params, &payload); err != nil {
		writeError(w, rpcError{Code: -32602, Message: err.Error()})
		return
	}
	resp, err := handler.GetApproval(r.Context(), payload.ID)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeJSONResult(w, req.ID, resp)
}

func (s *Server) handleListApprovals(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	handler, ok := s.Handler.(server.ApprovalHandler)
	if !ok {
		writeRPCError(w, status.Error(codes.Unimplemented, "approvals not supported"))
		return
	}
	var payload struct {
		TaskID        string `json:"task_id"`
		ContextID     string `json:"context_id"`
		Status        string `json:"status"`
		Limit         int    `json:"limit"`
		ExpiresBefore int64  `json:"expires_before"`
	}
	if len(req.Params) > 0 {
		if err := decodeJSONParams(req.Params, &payload); err != nil {
			writeError(w, rpcError{Code: -32602, Message: err.Error()})
			return
		}
	}
	filter := server.ApprovalFilter{
		TaskID:    payload.TaskID,
		ContextID: payload.ContextID,
		Status:    server.ApprovalStatus(payload.Status),
		Limit:     payload.Limit,
	}
	if payload.ExpiresBefore > 0 {
		filter.ExpiringBefore = time.UnixMilli(payload.ExpiresBefore).UTC()
	}
	if filter.Status != "" && filter.Status != server.ApprovalStatusPending && filter.Status != server.ApprovalStatusApproved && filter.Status != server.ApprovalStatusRejected {
		writeError(w, rpcError{Code: -32602, Message: "invalid status"})
		return
	}
	resp, err := handler.ListApprovals(r.Context(), filter)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeJSONResult(w, req.ID, resp)
}

func (s *Server) handleApproveApproval(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	handler, ok := s.Handler.(server.ApprovalHandler)
	if !ok {
		writeRPCError(w, status.Error(codes.Unimplemented, "approvals not supported"))
		return
	}
	var payload struct {
		ID     string `json:"id"`
		Reason string `json:"reason"`
	}
	if err := decodeJSONParams(req.Params, &payload); err != nil {
		writeError(w, rpcError{Code: -32602, Message: err.Error()})
		return
	}
	resp, err := handler.Approve(r.Context(), payload.ID, payload.Reason)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeResult(w, req.ID, resp)
}

func (s *Server) handleRejectApproval(w http.ResponseWriter, r *http.Request, req rpcRequest) {
	handler, ok := s.Handler.(server.ApprovalHandler)
	if !ok {
		writeRPCError(w, status.Error(codes.Unimplemented, "approvals not supported"))
		return
	}
	var payload struct {
		ID     string `json:"id"`
		Reason string `json:"reason"`
	}
	if err := decodeJSONParams(req.Params, &payload); err != nil {
		writeError(w, rpcError{Code: -32602, Message: err.Error()})
		return
	}
	resp, err := handler.Reject(r.Context(), payload.ID, payload.Reason)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeResult(w, req.ID, resp)
}

func decodeParams(params json.RawMessage, msg proto.Message) error {
	if len(params) == 0 {
		return status.Error(codes.InvalidArgument, "missing params")
	}
	return protojson.Unmarshal(params, msg)
}

func decodeJSONParams(params json.RawMessage, target any) error {
	if len(params) == 0 {
		return status.Error(codes.InvalidArgument, "missing params")
	}
	return json.Unmarshal(params, target)
}

func writeResult(w http.ResponseWriter, id any, msg proto.Message) {
	payload, err := protojson.Marshal(msg)
	if err != nil {
		writeRPCError(w, status.Error(codes.Internal, err.Error()))
		return
	}
	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  json.RawMessage(payload),
	}
	writeJSON(w, resp)
}

func writeJSONResult(w http.ResponseWriter, id any, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		writeRPCError(w, status.Error(codes.Internal, err.Error()))
		return
	}
	resp := rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  json.RawMessage(raw),
	}
	writeJSON(w, resp)
}

func writeRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		writeError(w, rpcError{Code: -32000, Message: err.Error()})
		return
	}
	code := -32000
	switch st.Code() {
	case codes.InvalidArgument:
		code = -32602
	case codes.NotFound:
		code = -32004
	case codes.Unauthenticated:
		code = -32001
	case codes.PermissionDenied:
		code = -32003
	case codes.Unimplemented:
		code = -32601
	}
	writeError(w, rpcError{Code: code, Message: st.Message()})
}

func writeError(w http.ResponseWriter, err rpcError) {
	resp := rpcResponse{
		JSONRPC: "2.0",
		Error:   &err,
	}
	writeJSON(w, resp)
}

func writeJSON(w http.ResponseWriter, payload rpcResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcStream struct {
	ctx context.Context
	w   http.ResponseWriter
	f   http.Flusher
	id  any
}

func (s *rpcStream) Context() context.Context {
	return s.ctx
}

func (s *rpcStream) Send(resp *a2av1.StreamResponse) error {
	payload, err := protojson.Marshal(resp)
	if err != nil {
		return err
	}
	result := rpcResponse{
		JSONRPC: "2.0",
		ID:      s.id,
		Result:  json.RawMessage(payload),
	}
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if _, err := s.w.Write([]byte("data: ")); err != nil {
		return err
	}
	if _, err := s.w.Write(data); err != nil {
		return err
	}
	if _, err := s.w.Write([]byte("\n\n")); err != nil {
		return err
	}
	s.f.Flush()
	return nil
}

func (s *rpcStream) SendHeader(metadata.MD) error { return nil }
func (s *rpcStream) SetHeader(metadata.MD) error  { return nil }
func (s *rpcStream) SetTrailer(metadata.MD)       {}
func (s *rpcStream) SendMsg(any) error            { return nil }
func (s *rpcStream) RecvMsg(any) error            { return nil }
