package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client handles discovery and communication with remote A2A agents.
type Client struct {
	httpClient *http.Client
}

// NewClient creates an A2A client with default timeouts.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 60 * time.Second,
		}
	}
	return &Client{httpClient: httpClient}
}

// Discover fetches the AgentCard from the agent's well-known endpoint.
func (c *Client) Discover(ctx context.Context, baseURL string) (*AgentCard, error) {
	cleanURL := strings.TrimRight(baseURL, "/")
	reqURL := fmt.Sprintf("%s/.well-known/agent-card.json", cleanURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create discovery request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discovery request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discovery returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("failed to decode agent card: %w", err)
	}

	return &card, nil
}

// SendMessage sends a text prompt to an agent over JSON-RPC 2.0 using the standard message/send method.
func (c *Client) SendMessage(ctx context.Context, baseURL string, prompt string, contextParams map[string]any) (*Message, error) {
	return c.SendMessageWithContext(ctx, baseURL, prompt, "", contextParams)
}

// SendMessageWithContext sends a text prompt with an explicit contextId for multi-turn session continuity.
func (c *Client) SendMessageWithContext(ctx context.Context, baseURL string, prompt string, contextID string, contextParams map[string]any) (*Message, error) {
	cleanURL := strings.TrimRight(baseURL, "/")
	reqURL := fmt.Sprintf("%s/rpc", cleanURL)

	msg := NewTextMessageWithContext("user", prompt, contextID)
	rpcReq := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "message/send",
		Params: SendMessageParams{
			Message:   msg,
			ContextID: contextID,
			Context:   contextParams,
		},
		ID: GenerateID(),
	}

	payload, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create rpc request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rpc request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("rpc returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var rpcResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode rpc response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("agent returned error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	if rpcResp.Result == nil {
		return nil, fmt.Errorf("agent returned empty result")
	}

	return rpcResp.Result, nil
}
