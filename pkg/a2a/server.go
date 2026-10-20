package a2a

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AgentHandler is the function signature for executing agent logic.
type AgentHandler func(ctx context.Context, req Message, contextParams map[string]any) (*Message, error)

// Server handles A2A protocol requests for an agent.
type Server struct {
	Card         AgentCard
	agentHandler AgentHandler
	mux          *http.ServeMux
}

// NewServer creates a new HTTP server configured with standard A2A endpoints.
func NewServer(card AgentCard, handler AgentHandler) *Server {
	s := &Server{
		Card:         card,
		agentHandler: handler,
		mux:          http.NewServeMux(),
	}
	s.routes()
	return s
}

// Handler returns the underlying http.Handler.
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /.well-known/agent-card.json", s.handleAgentCard)
	s.mux.HandleFunc("POST /rpc", s.handleRPC)
	s.mux.HandleFunc("GET /rpc", s.handleMethodNotAllowed)
}

func (s *Server) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(s.Card)
}

func (s *Server) handleMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": fmt.Sprintf("Method %s not allowed on /rpc; use POST for JSON-RPC 2.0", r.Method),
	})
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeRPCError(w, nil, -32700, "Parse error: "+err.Error())
		return
	}
	defer r.Body.Close()

	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		s.writeRPCError(w, nil, -32700, "Parse error: invalid JSON")
		return
	}

	if req.JSONRPC != "2.0" {
		s.writeRPCError(w, req.ID, -32600, "Invalid Request: jsonrpc must be '2.0'")
		return
	}

	if req.Method != "message/send" && req.Method != "send_message" {
		s.writeRPCError(w, req.ID, -32601, fmt.Sprintf("Method not found: '%s' is not supported (expected 'message/send' or 'send_message')", req.Method))
		return
	}

	if len(req.Params.Message.Parts) == 0 {
		s.writeRPCError(w, req.ID, -32602, "Invalid params: 'message.parts' must not be empty")
		return
	}

	// Propagate ContextID into contextParams if not present
	ctxParams := req.Params.Context
	if ctxParams == nil {
		ctxParams = make(map[string]any)
	}
	if req.Params.ContextID != "" {
		ctxParams["contextId"] = req.Params.ContextID
	} else if req.Params.Message.ContextID != "" {
		ctxParams["contextId"] = req.Params.Message.ContextID
	}

	resMsg, err := s.agentHandler(r.Context(), req.Params.Message, ctxParams)
	if err != nil {
		s.writeRPCError(w, req.ID, -32603, "Internal error: "+err.Error())
		return
	}

	if resMsg.ContextID == "" && ctxParams["contextId"] != nil {
		resMsg.ContextID = fmt.Sprintf("%v", ctxParams["contextId"])
	}

	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  resMsg,
		ID:      req.ID,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) writeRPCError(w http.ResponseWriter, id any, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
	_ = json.NewEncoder(w).Encode(resp)
}
