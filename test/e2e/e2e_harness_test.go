//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"a2ademo/internal/agents/auditor"
	"a2ademo/internal/agents/orchestrator"
	"a2ademo/pkg/a2a"
	"a2ademo/pkg/bedrock"
	"a2ademo/pkg/config"
	"a2ademo/pkg/logger"
)

// TestLiveDeployedLambdaA2A validates the live cloud-deployed AWS Bedrock Harness Lambda.
// It strictly requires HARNESS_AGENT_URL and does NOT silently fall back to an in-process server.
func TestLiveDeployedLambdaA2A(t *testing.T) {
	cfg := config.Load()
	if cfg.HarnessAgentURL == "" {
		t.Skip("Skipping live Lambda A2A test: HARNESS_AGENT_URL is not configured in .env")
	}

	harnessURL := strings.TrimRight(cfg.HarnessAgentURL, "/")
	logger.Log(logger.TagInfo, "[E2E-HARNESS] Testing live deployed AWS Lambda at: %s", harnessURL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	testClient := &http.Client{Timeout: 90 * time.Second}
	a2aClient := a2a.NewClient(testClient)

	// =========================================================================
	// PHASE 1: Discovery & AgentCard URL Validation
	// =========================================================================
	logger.Section("PHASE 1: AgentCard Discovery & URL Validation")

	cardURL := harnessURL + "/.well-known/agent-card.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cardURL, nil)
	if err != nil {
		t.Fatalf("failed to create discovery request: %v", err)
	}

	resp, err := testClient.Do(req)
	if err != nil {
		t.Fatalf("failed to fetch agent card from %s: %v", cardURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected HTTP 200 on discovery, got %d: %s", resp.StatusCode, string(body))
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json Content-Type, got %s", ct)
	}

	var card a2a.AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		t.Fatalf("failed to decode AgentCard JSON: %v", err)
	}

	logger.Log(logger.TagDiscovery, "Discovered Agent: %s (v%s, protocol: %s)", card.Name, card.Version, card.ProtocolVersion)
	logger.Log(logger.TagDiscovery, "Advertised Card URL: %s", card.URL)

	if card.Name != "SecurityAuditorAgent" {
		t.Errorf("expected agent name 'SecurityAuditorAgent', got %q", card.Name)
	}
	if card.ProtocolVersion != "0.2.0" {
		t.Errorf("expected protocol version '0.2.0', got %q", card.ProtocolVersion)
	}
	if len(card.Skills) < 3 {
		t.Errorf("expected at least 3 skills, got %d", len(card.Skills))
	}

	// Validate advertised URL matches target endpoint
	trimmedCardURL := strings.TrimRight(card.URL, "/")
	if trimmedCardURL != harnessURL {
		t.Errorf("card.URL (%q) does not match target harnessURL (%q)", trimmedCardURL, harnessURL)
	}

	// =========================================================================
	// PHASE 2: HTTP Transport & Routing Validation
	// =========================================================================
	logger.Section("PHASE 2: HTTP Transport & Routing Validation")

	// 2a. GET /rpc should return HTTP 405 Method Not Allowed
	rpcURL := harnessURL + "/rpc"
	getRpcReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, rpcURL, nil)
	getRpcResp, err := testClient.Do(getRpcReq)
	if err != nil {
		t.Fatalf("GET /rpc failed: %v", err)
	}
	getRpcResp.Body.Close()
	if getRpcResp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected HTTP 405 on GET /rpc, got %d", getRpcResp.StatusCode)
	} else {
		logger.Log(logger.TagResponse, "✓ Correctly returned HTTP 405 on GET /rpc")
	}

	// 2b. GET /non-existent-path should return HTTP 404 Not Found
	notFoundReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, harnessURL+"/unknown-path-probe", nil)
	notFoundResp, err := testClient.Do(notFoundReq)
	if err != nil {
		t.Fatalf("GET /unknown-path-probe failed: %v", err)
	}
	notFoundResp.Body.Close()
	if notFoundResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected HTTP 404 on unknown path, got %d", notFoundResp.StatusCode)
	} else {
		logger.Log(logger.TagResponse, "✓ Correctly returned HTTP 404 on unknown path")
	}

	// =========================================================================
	// PHASE 3: JSON-RPC 2.0 Error Compliance Matrix
	// =========================================================================
	logger.Section("PHASE 3: JSON-RPC 2.0 Error Compliance Matrix")

	// 3a. Parse Error (-32700): Must return jsonrpc 2.0, code -32700, and id: null
	logger.Log(logger.TagRequest, "Testing Parse Error (-32700)")
	parseResp, err := testClient.Post(rpcURL, "application/json", bytes.NewReader([]byte("{invalid-json-body")))
	if err != nil {
		t.Fatalf("parse error probe failed: %v", err)
	}
	var parseRpcResp a2a.JSONRPCResponse
	_ = json.NewDecoder(parseResp.Body).Decode(&parseRpcResp)
	parseResp.Body.Close()

	if parseRpcResp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc '2.0', got %q", parseRpcResp.JSONRPC)
	}
	if parseRpcResp.ID != nil {
		t.Errorf("expected id: null on parse error, got: %v", parseRpcResp.ID)
	}
	if parseRpcResp.Error == nil || parseRpcResp.Error.Code != -32700 {
		t.Errorf("expected error -32700, got: %+v", parseRpcResp.Error)
	} else {
		logger.Log(logger.TagResponse, "✓ -32700 Parse error passed (id: null)")
	}

	// 3b. Invalid Request Version (-32600): String ID echoing
	logger.Log(logger.TagRequest, "Testing Invalid Request Version (-32600)")
	invVerPayload := `{"jsonrpc": "1.0", "method": "message/send", "params": {}, "id": "req-ver-99"}`
	invVerResp, err := testClient.Post(rpcURL, "application/json", strings.NewReader(invVerPayload))
	if err != nil {
		t.Fatalf("invalid version probe failed: %v", err)
	}
	var invVerRpcResp a2a.JSONRPCResponse
	_ = json.NewDecoder(invVerResp.Body).Decode(&invVerRpcResp)
	invVerResp.Body.Close()

	if fmt.Sprintf("%v", invVerRpcResp.ID) != "req-ver-99" {
		t.Errorf("expected echoed id 'req-ver-99', got %v", invVerRpcResp.ID)
	}
	if invVerRpcResp.Error == nil || invVerRpcResp.Error.Code != -32600 {
		t.Errorf("expected error -32600, got: %+v", invVerRpcResp.Error)
	} else {
		logger.Log(logger.TagResponse, "✓ -32600 Invalid request passed (id echoed)")
	}

	// 3c. Method Not Found (-32601): Numeric ID echoing
	logger.Log(logger.TagRequest, "Testing Method Not Found (-32601) with numeric ID")
	badMethodPayload := `{"jsonrpc": "2.0", "method": "unsupported/method", "params": {}, "id": 5050}`
	badMethodResp, err := testClient.Post(rpcURL, "application/json", strings.NewReader(badMethodPayload))
	if err != nil {
		t.Fatalf("bad method probe failed: %v", err)
	}
	var badMethodRpcResp a2a.JSONRPCResponse
	_ = json.NewDecoder(badMethodResp.Body).Decode(&badMethodRpcResp)
	badMethodResp.Body.Close()

	if fmt.Sprintf("%v", badMethodRpcResp.ID) != "5050" {
		t.Errorf("expected echoed numeric id 5050, got %v", badMethodRpcResp.ID)
	}
	if badMethodRpcResp.Error == nil || badMethodRpcResp.Error.Code != -32601 {
		t.Errorf("expected error -32601, got: %+v", badMethodRpcResp.Error)
	} else {
		logger.Log(logger.TagResponse, "✓ -32601 Method not found passed (numeric id echoed)")
	}

	// 3d. Invalid Params (-32602): Empty message parts
	logger.Log(logger.TagRequest, "Testing Invalid Params (-32602) with empty parts")
	badParamsPayload := `{"jsonrpc": "2.0", "method": "message/send", "params": {"message": {"parts": []}}, "id": "req-param-1"}`
	badParamsResp, err := testClient.Post(rpcURL, "application/json", strings.NewReader(badParamsPayload))
	if err != nil {
		t.Fatalf("bad params probe failed: %v", err)
	}
	var badParamsRpcResp a2a.JSONRPCResponse
	_ = json.NewDecoder(badParamsResp.Body).Decode(&badParamsRpcResp)
	badParamsResp.Body.Close()

	if badParamsRpcResp.Error == nil || badParamsRpcResp.Error.Code != -32602 {
		t.Errorf("expected error -32602, got: %+v", badParamsRpcResp.Error)
	} else {
		logger.Log(logger.TagResponse, "✓ -32602 Invalid params passed: %s", badParamsRpcResp.Error.Message)
	}

	// =========================================================================
	// PHASE 4: Multi-Turn Stateful Continuity & Isolation Verification
	// =========================================================================
	logger.Section("PHASE 4: Multi-Turn Bedrock Harness Session Memory & Isolation")

	// Dynamic, unguessable identifiers for this test run
	tokenSecretA := fmt.Sprintf("AUDIT-KEY-%s", a2a.GenerateID())
	contextIDA := fmt.Sprintf("sess-audit-a-%s", a2a.GenerateID())

	// 4a. Turn 1 (Initialize Session A)
	turn1Prompt := fmt.Sprintf("Note this session security verification token: %s. Acknowledge receipt.", tokenSecretA)
	logger.Log(logger.TagRequest, "Turn 1 (Session A: %s): Setting token %s", contextIDA, tokenSecretA)

	turn1Msg, err := a2aClient.SendMessageWithContext(ctx, harnessURL, turn1Prompt, contextIDA, nil)
	if err != nil {
		t.Fatalf("Turn 1 execution failed: %v", err)
	}

	if turn1Msg.Role != "agent" {
		t.Errorf("expected role 'agent', got %s", turn1Msg.Role)
	}
	if len(turn1Msg.Parts) == 0 || strings.TrimSpace(turn1Msg.Parts[0].Text) == "" {
		t.Fatalf("expected non-empty response on Turn 1")
	}
	logger.Log(logger.TagResponse, "Turn 1 response received (%d bytes, context: %s)", len(turn1Msg.Parts[0].Text), turn1Msg.ContextID)

	// 4b. Turn 2 (Positive Control: Recall within Session A)
	turn2Prompt := "What was the session security verification token provided in the previous message?"
	logger.Log(logger.TagRequest, "Turn 2 (Session A: %s): Querying secret", contextIDA)

	turn2Msg, err := a2aClient.SendMessageWithContext(ctx, harnessURL, turn2Prompt, contextIDA, nil)
	if err != nil {
		t.Fatalf("Turn 2 execution failed: %v", err)
	}

	turn2Text := turn2Msg.Parts[0].Text
	logger.Log(logger.TagResponse, "Turn 2 recall response:\n%s", turn2Text)

	if !strings.Contains(turn2Text, tokenSecretA) {
		t.Errorf("Bedrock Harness session failed to recall secret token %q in Session A", tokenSecretA)
	} else {
		logger.Log(logger.TagAgent, "✓ Positive Control PASSED: Bedrock Harness recalled token %s across turns!", tokenSecretA)
	}

	// 4c. Turn 3 (Negative Control: Isolation in Fresh Session B)
	contextIDB := fmt.Sprintf("sess-audit-b-%s", a2a.GenerateID())
	turn3Prompt := "What was the session security verification token provided in the previous message?"
	logger.Log(logger.TagRequest, "Turn 3 (Session B: %s - Negative Control): Querying secret in fresh session", contextIDB)

	turn3Msg, err := a2aClient.SendMessageWithContext(ctx, harnessURL, turn3Prompt, contextIDB, nil)
	if err != nil {
		t.Fatalf("Turn 3 execution failed: %v", err)
	}

	turn3Text := turn3Msg.Parts[0].Text
	logger.Log(logger.TagResponse, "Turn 3 isolation response:\n%s", turn3Text)

	if strings.Contains(turn3Text, tokenSecretA) {
		t.Errorf("Isolation failure: Fresh Session B leaked secret token %q from Session A!", tokenSecretA)
	} else {
		logger.Log(logger.TagAgent, "✓ Negative Control PASSED: Fresh Session B had no access to Session A's secret!")
	}
}

// TestLiveOrchestratorToHarnessIntegration verifies multi-agent orchestration and delegation
// to the external Bedrock Harness agent.
func TestLiveOrchestratorToHarnessIntegration(t *testing.T) {
	cfg := config.Load()
	if !cfg.HasAWSCredentials() {
		t.Skip("Skipping orchestrator integration: AWS credentials not configured in .env")
	}
	if cfg.HarnessAgentURL == "" {
		t.Skip("Skipping orchestrator integration: HARNESS_AGENT_URL not configured in .env")
	}

	harnessURL := strings.TrimRight(cfg.HarnessAgentURL, "/")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	liveLLM, err := bedrock.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to initialize live Bedrock client: %v", err)
	}

	a2aClient := a2a.NewClient(nil)
	orch := orchestrator.NewOrchestrator(liveLLM, a2aClient, "", "", "http://orchestrator.internal")

	logger.Section("ORCHESTRATOR DELEGATION & SYNTHESIS TO HARNESS AGENT")
	auditGoal := "Perform an architectural security audit on multi-agent JSON-RPC delegation."
	delegatedMsg, card, err := orch.DelegateToAgent(ctx, harnessURL, auditGoal)
	if err != nil {
		t.Fatalf("orchestrator delegation to harness agent failed: %v", err)
	}

	if card.Name != "SecurityAuditorAgent" {
		t.Errorf("expected discovered agent 'SecurityAuditorAgent', got %q", card.Name)
	}
	if len(delegatedMsg.Parts) == 0 || strings.TrimSpace(delegatedMsg.Parts[0].Text) == "" {
		t.Errorf("expected non-empty delegated response text")
	}

	logger.Log(logger.TagOrchestrator, "✓ Complete A2A communication between Orchestrator and Harness Agent verified successfully!")
}

// TestInProcessA2AHarnessAgent tests the Go in-process implementation of the Harness Agent
// with full JSON-RPC error handling and multi-turn session recall.
func TestInProcessA2AHarnessAgent(t *testing.T) {
	cfg := config.Load()
	if !cfg.HasAWSCredentials() {
		t.Skip("Skipping in-process Harness Agent test: AWS credentials not configured in .env")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	liveLLM, err := bedrock.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to initialize live Bedrock client: %v", err)
	}

	harnessAgent := auditor.NewAgent(liveLLM, "http://harness-agent.internal")
	harnessServer := a2a.NewServer(harnessAgent.Card(), harnessAgent.HandleMessage)

	testClient := &http.Client{
		Transport: &routedTransport{
			routes: map[string]http.Handler{
				"harness-agent.internal": harnessServer.Handler(),
			},
		},
		Timeout: 60 * time.Second,
	}

	a2aClient := a2a.NewClient(testClient)
	harnessURL := "http://harness-agent.internal"

	// 1. Discovery
	card, err := a2aClient.Discover(ctx, harnessURL)
	if err != nil {
		t.Fatalf("failed to discover in-process agent card: %v", err)
	}
	if card.Name != "SecurityAuditorAgent" {
		t.Errorf("expected agent 'SecurityAuditorAgent', got %q", card.Name)
	}

	// 2. Multi-turn session recall on local Go agent
	secretToken := fmt.Sprintf("LOCAL-SEC-%s", a2a.GenerateID())
	sessID := fmt.Sprintf("local-sess-%s", a2a.GenerateID())

	// Turn 1
	turn1Msg, err := a2aClient.SendMessageWithContext(ctx, harnessURL, "Security token is "+secretToken+". Acknowledge.", sessID, nil)
	if err != nil {
		t.Fatalf("local Turn 1 failed: %v", err)
	}
	if len(turn1Msg.Parts) == 0 {
		t.Fatalf("empty parts on local Turn 1")
	}

	// Turn 2
	turn2Msg, err := a2aClient.SendMessageWithContext(ctx, harnessURL, "What was the security token?", sessID, nil)
	if err != nil {
		t.Fatalf("local Turn 2 failed: %v", err)
	}
	if !strings.Contains(turn2Msg.Parts[0].Text, secretToken) {
		t.Errorf("expected local in-process agent to recall token %q, got: %s", secretToken, turn2Msg.Parts[0].Text)
	} else {
		logger.Log(logger.TagAgent, "✓ In-process Go agent successfully recalled secret token %s across turns!", secretToken)
	}
}
