//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"a2ademo/internal/agents/analysis"
	"a2ademo/internal/agents/orchestrator"
	"a2ademo/internal/agents/research"
	"a2ademo/pkg/a2a"
	"a2ademo/pkg/bedrock"
	"a2ademo/pkg/config"
	"a2ademo/pkg/logger"
)

// routedTransport routes requests in-memory based on URL host prefix without network sockets.
type routedTransport struct {
	routes map[string]http.Handler
}

func (rt *routedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	handler, ok := rt.routes[req.URL.Host]
	if !ok {
		for _, h := range rt.routes {
			handler = h
			break
		}
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Result(), nil
}

// TestLiveBedrockDirectConverse tests direct communication with the configured AWS Bedrock model.
func TestLiveBedrockDirectConverse(t *testing.T) {
	cfg := config.Load()
	if !cfg.HasAWSCredentials() {
		t.Skip("Skipping live Bedrock E2E test: AWS credentials not configured in .env")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	logger.Log(logger.TagInfo, "[E2E] Initializing live Bedrock client for model: %s (region: %s)", cfg.BedrockModelID, cfg.AWSRegion)
	client, err := bedrock.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to initialize Bedrock client: %v", err)
	}

	systemPrompt := "You are a concise testing assistant. Reply in one short sentence."
	userPrompt := "Say 'Bedrock live E2E test passed successfully.'"

	resp, err := client.Converse(ctx, systemPrompt, userPrompt)
	if err != nil {
		t.Fatalf("live Bedrock Converse call failed: %v", err)
	}

	logger.Log(logger.TagInfo, "[E2E] Live Bedrock response received: %q", resp)
	if strings.TrimSpace(resp) == "" {
		t.Errorf("expected non-empty response from live Bedrock")
	}
}

// TestLiveMultiAgentE2EOrchestration runs the complete multi-agent workflow using live AWS Bedrock models.
func TestLiveMultiAgentE2EOrchestration(t *testing.T) {
	cfg := config.Load()
	if !cfg.HasAWSCredentials() {
		t.Skip("Skipping live multi-agent E2E test: AWS credentials not configured in .env")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	logger.Log(logger.TagInfo, "[E2E] Initializing multi-agent system with live Bedrock model: %s", cfg.BedrockModelID)
	liveLLM, err := bedrock.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create live Bedrock client: %v", err)
	}

	resBaseURL := "http://research-agent.internal"
	anaBaseURL := "http://analysis-agent.internal"
	orchBaseURL := "http://orchestrator.internal"

	resAgent := research.NewAgent(liveLLM, resBaseURL)
	resServer := a2a.NewServer(resAgent.Card(), resAgent.HandleMessage)

	anaAgent := analysis.NewAgent(liveLLM, anaBaseURL)
	anaServer := a2a.NewServer(anaAgent.Card(), anaAgent.HandleMessage)

	transport := &routedTransport{
		routes: map[string]http.Handler{
			"research-agent.internal": resServer.Handler(),
			"analysis-agent.internal": anaServer.Handler(),
		},
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   120 * time.Second,
	}
	a2aClient := a2a.NewClient(httpClient)

	orch := orchestrator.NewOrchestrator(liveLLM, a2aClient, resBaseURL, anaBaseURL, orchBaseURL)

	goal := "In under 200 words, summarize the primary benefits and latency trade-offs of the Agent-to-Agent (A2A) protocol in edge computing."
	result, err := orch.Run(ctx, goal)
	if err != nil {
		t.Fatalf("live multi-agent orchestration failed: %v", err)
	}

	if result.ResearchFindings == "" {
		t.Errorf("expected non-empty research findings from live agent")
	}
	if result.AnalysisFindings == "" {
		t.Errorf("expected non-empty analysis findings from live agent")
	}
	if result.FinalSynthesis == "" {
		t.Errorf("expected non-empty final synthesis from live orchestrator")
	}

	logger.Log(logger.TagInfo, "[E2E] Live multi-agent workflow completed successfully (%d bytes final report)", len(result.FinalSynthesis))
}
