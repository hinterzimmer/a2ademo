package a2a

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"
)

// AgentSkill represents a distinct capability advertised by an A2A agent.
type AgentSkill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
}

// AgentCard contains metadata describing an agent's identity and capabilities.
// This is conventionally hosted at /.well-known/agent-card.json.
type AgentCard struct {
	Name            string       `json:"name"`
	Description     string       `json:"description"`
	URL             string       `json:"url"`
	Version         string       `json:"version"`
	ProtocolVersion string       `json:"protocol_version"`
	Skills          []AgentSkill `json:"skills"`
}

// Part represents a single content piece within an A2A message.
type Part struct {
	Text string `json:"text"`
}

// Message represents an A2A conversational message.
type Message struct {
	Role      string    `json:"role"` // "user", "agent", "system"
	Parts     []Part    `json:"parts"`
	MessageID string    `json:"messageId,omitempty"`
	ContextID string    `json:"contextId,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// UnmarshalJSON supports both standard camelCase and legacy snake_case for Message.
func (m *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	aux := struct {
		*Alias
		LegacyMessageID string `json:"message_id"`
		LegacyContextID string `json:"context_id"`
	}{
		Alias: (*Alias)(m),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if m.MessageID == "" && aux.LegacyMessageID != "" {
		m.MessageID = aux.LegacyMessageID
	}
	if m.ContextID == "" && aux.LegacyContextID != "" {
		m.ContextID = aux.LegacyContextID
	}
	return nil
}

// SendMessageParams holds parameters for the message/send and send_message JSON-RPC methods.
type SendMessageParams struct {
	Message   Message        `json:"message"`
	ContextID string         `json:"contextId,omitempty"`
	Context   map[string]any `json:"context,omitempty"`
}

// UnmarshalJSON supports both standard camelCase and legacy snake_case for SendMessageParams.
func (p *SendMessageParams) UnmarshalJSON(data []byte) error {
	type Alias SendMessageParams
	aux := struct {
		*Alias
		LegacyContextID string `json:"context_id"`
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if p.ContextID == "" && aux.LegacyContextID != "" {
		p.ContextID = aux.LegacyContextID
	}
	return nil
}

// JSONRPCRequest represents a JSON-RPC 2.0 request payload.
type JSONRPCRequest struct {
	JSONRPC string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	Params  SendMessageParams `json:"params"`
	ID      any               `json:"id"`
}

// JSONRPCError represents a standard JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response payload.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  *Message      `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      any           `json:"id"`
}

// NewTextMessage creates a simple Message with text content.
func NewTextMessage(role string, text string) Message {
	return Message{
		Role:      role,
		Parts:     []Part{{Text: text}},
		MessageID: GenerateID(),
		Timestamp: time.Now().UTC(),
	}
}

// NewTextMessageWithContext creates a Message with text content and an associated context/session ID.
func NewTextMessageWithContext(role string, text string, contextID string) Message {
	return Message{
		Role:      role,
		Parts:     []Part{{Text: text}},
		MessageID: GenerateID(),
		ContextID: contextID,
		Timestamp: time.Now().UTC(),
	}
}

// GenerateID returns a random hex identifier.
func GenerateID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "msg-fallback-id"
	}
	return hex.EncodeToString(bytes)
}
