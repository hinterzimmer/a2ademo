package bedrock

import (
	"context"
	"strings"
	"testing"
)

func TestMockClientConverse(t *testing.T) {
	mock := NewMockClient("test-model")

	if mock.ModelID() != "test-model" {
		t.Errorf("expected modelID test-model, got %s", mock.ModelID())
	}

	// Test Research prompt
	resResp, err := mock.Converse(context.Background(), "You are a research agent", "Investigate quantum computing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(resResp, "Research Findings") {
		t.Errorf("expected research findings, got: %s", resResp)
	}

	// Test Analysis prompt
	anaResp, err := mock.Converse(context.Background(), "You are an analysis agent", "Analyze the security risks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(anaResp, "Strategic Analysis") {
		t.Errorf("expected strategic analysis, got: %s", anaResp)
	}

	// Test Synthesis prompt
	synResp, err := mock.Converse(context.Background(), "You are an orchestrator", "Synthesize findings")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(synResp, "Executive Synthesis") {
		t.Errorf("expected executive synthesis, got: %s", synResp)
	}
}
