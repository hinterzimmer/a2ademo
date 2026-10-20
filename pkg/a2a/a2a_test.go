package a2a

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// inMemoryTransport routes requests directly to an http.Handler without binding a network socket.
type inMemoryTransport struct {
	handler http.Handler
}

func (t *inMemoryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	t.handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}

func TestA2AServerAndClient(t *testing.T) {
	card := AgentCard{
		Name:            "EchoAgent",
		Description:     "Test Echo Agent",
		URL:             "http://echo.internal",
		Version:         "1.0.0",
		ProtocolVersion: "0.2.0",
		Skills: []AgentSkill{
			{
				ID:          "echo",
				Name:        "Echo",
				Description: "Echoes messages back",
				Tags:        []string{"test"},
			},
		},
	}

	handler := func(ctx context.Context, req Message, contextParams map[string]any) (*Message, error) {
		text := ""
		if len(req.Parts) > 0 {
			text = req.Parts[0].Text
		}
		res := NewTextMessage("agent", fmt.Sprintf("Echo: %s", text))
		return &res, nil
	}

	server := NewServer(card, handler)
	client := NewClient(&http.Client{
		Transport: &inMemoryTransport{handler: server.Handler()},
	})

	baseURL := "http://echo.internal"

	// Test Discovery
	discoveredCard, err := client.Discover(context.Background(), baseURL)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	if discoveredCard.Name != "EchoAgent" {
		t.Errorf("expected agent name EchoAgent, got %s", discoveredCard.Name)
	}
	if len(discoveredCard.Skills) != 1 || discoveredCard.Skills[0].ID != "echo" {
		t.Errorf("unexpected skills: %+v", discoveredCard.Skills)
	}

	// Test SendMessage
	respMsg, err := client.SendMessage(context.Background(), baseURL, "Hello A2A", nil)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}
	if len(respMsg.Parts) == 0 {
		t.Fatalf("expected message parts, got none")
	}
	expected := "Echo: Hello A2A"
	if respMsg.Parts[0].Text != expected {
		t.Errorf("expected %q, got %q", expected, respMsg.Parts[0].Text)
	}
}

func TestA2AErrorHandling(t *testing.T) {
	card := AgentCard{
		Name:    "FailingAgent",
		Version: "1.0.0",
	}

	handler := func(ctx context.Context, req Message, contextParams map[string]any) (*Message, error) {
		return nil, fmt.Errorf("simulated failure")
	}

	server := NewServer(card, handler)
	client := NewClient(&http.Client{
		Transport: &inMemoryTransport{handler: server.Handler()},
	})

	baseURL := "http://failing.internal"

	_, err := client.SendMessage(context.Background(), baseURL, "test", nil)
	if err == nil {
		t.Fatalf("expected error from failing handler, got nil")
	}
}
