package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"a2ademo/internal/agents/analysis"
	"a2ademo/internal/agents/orchestrator"
	"a2ademo/internal/agents/research"
	"a2ademo/pkg/a2a"
	"a2ademo/pkg/bedrock"
)

// routedTransport routes requests in-memory based on URL host prefix without network sockets.
type routedTransport struct {
	routes map[string]http.Handler
}

func (rt *routedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	handler, ok := rt.routes[req.URL.Host]
	if !ok {
		// Default to first route if single handler
		for _, h := range rt.routes {
			handler = h
			break
		}
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}

func TestEndToEndA2AOrchestrationIntegration(t *testing.T) {
	mockLLM := bedrock.NewMockClient("mock.claude-3-5-sonnet")

	resBaseURL := "http://research-agent.local"
	anaBaseURL := "http://analysis-agent.local"
	orchBaseURL := "http://orchestrator.local"

	resAgent := research.NewAgent(mockLLM, resBaseURL)
	resServer := a2a.NewServer(resAgent.Card(), resAgent.HandleMessage)

	anaAgent := analysis.NewAgent(mockLLM, anaBaseURL)
	anaServer := a2a.NewServer(anaAgent.Card(), anaAgent.HandleMessage)

	transport := &routedTransport{
		routes: map[string]http.Handler{
			"research-agent.local": resServer.Handler(),
			"analysis-agent.local": anaServer.Handler(),
		},
	}

	httpClient := &http.Client{Transport: transport}
	a2aClient := a2a.NewClient(httpClient)

	// Verify Discovery
	ctx := context.Background()
	resCard, err := a2aClient.Discover(ctx, resBaseURL)
	if err != nil {
		t.Fatalf("failed to discover research agent: %v", err)
	}
	if resCard.Name != "ResearchAgent" {
		t.Errorf("expected ResearchAgent, got %s", resCard.Name)
	}
	if len(resCard.Skills) < 2 {
		t.Errorf("expected at least 2 skills in ResearchAgent, got %d", len(resCard.Skills))
	}

	anaCard, err := a2aClient.Discover(ctx, anaBaseURL)
	if err != nil {
		t.Fatalf("failed to discover analysis agent: %v", err)
	}
	if anaCard.Name != "AnalysisAgent" {
		t.Errorf("expected AnalysisAgent, got %s", anaCard.Name)
	}
	if len(anaCard.Skills) < 2 {
		t.Errorf("expected at least 2 skills in AnalysisAgent, got %d", len(anaCard.Skills))
	}

	// Verify Direct A2A Message Exchange
	resMsg, err := a2aClient.SendMessage(ctx, resBaseURL, "Research topic X", nil)
	if err != nil {
		t.Fatalf("failed to send message to research agent: %v", err)
	}
	if len(resMsg.Parts) == 0 || !strings.Contains(resMsg.Parts[0].Text, "Research Findings") {
		t.Errorf("unexpected research message response: %+v", resMsg)
	}

	anaMsg, err := a2aClient.SendMessage(ctx, anaBaseURL, "Analyze findings Y", nil)
	if err != nil {
		t.Fatalf("failed to send message to analysis agent: %v", err)
	}
	if len(anaMsg.Parts) == 0 || !strings.Contains(anaMsg.Parts[0].Text, "Strategic Analysis") {
		t.Errorf("unexpected analysis message response: %+v", anaMsg)
	}

	// Run Full Orchestration Workflow
	orch := orchestrator.NewOrchestrator(mockLLM, a2aClient, resBaseURL, anaBaseURL, orchBaseURL)
	workflowResult, err := orch.Run(ctx, "Evaluate distributed agent systems with A2A protocol")
	if err != nil {
		t.Fatalf("orchestration workflow failed: %v", err)
	}

	if workflowResult.ResearchFindings == "" {
		t.Errorf("expected non-empty research findings")
	}
	if workflowResult.AnalysisFindings == "" {
		t.Errorf("expected non-empty analysis findings")
	}
	if workflowResult.FinalSynthesis == "" {
		t.Errorf("expected non-empty final synthesis")
	}
	if !strings.Contains(workflowResult.FinalSynthesis, "Executive Synthesis") {
		t.Errorf("expected Executive Synthesis in final report, got: %s", workflowResult.FinalSynthesis)
	}
}
