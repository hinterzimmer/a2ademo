package analysis

import (
	"context"
	"fmt"

	"a2ademo/pkg/a2a"
	"a2ademo/pkg/bedrock"
	"a2ademo/pkg/logger"
)

// Agent implements the Analysis Agent service.
type Agent struct {
	llm     bedrock.LLMClient
	baseURL string
}

// NewAgent creates a new Analysis Agent instance.
func NewAgent(llm bedrock.LLMClient, baseURL string) *Agent {
	return &Agent{
		llm:     llm,
		baseURL: baseURL,
	}
}

// Card returns the A2A AgentCard describing the agent's capabilities.
func (a *Agent) Card() a2a.AgentCard {
	return a2a.AgentCard{
		Name:            "AnalysisAgent",
		Description:     "Specialized agent for critical analysis, risk evaluation, and trade-off assessment.",
		URL:             a.baseURL,
		Version:         "1.0.0",
		ProtocolVersion: "0.2.0",
		Skills: []a2a.AgentSkill{
			{
				ID:          "analyze_findings",
				Name:        "Analyze Research Findings",
				Description: "Evaluates findings for technical trade-offs, scalability, and integration complexity.",
				Tags:        []string{"analysis", "evaluation", "trade-offs"},
			},
			{
				ID:          "risk_assessment",
				Name:        "Risk & Security Assessment",
				Description: "Identifies security, operational, and protocol boundary vulnerabilities.",
				Tags:        []string{"security", "risks", "governance"},
			},
		},
	}
}

// HandleMessage processes incoming A2A JSON-RPC requests.
func (a *Agent) HandleMessage(ctx context.Context, msg a2a.Message, ctxParams map[string]any) (*a2a.Message, error) {
	userPrompt := ""
	if len(msg.Parts) > 0 {
		userPrompt = msg.Parts[0].Text
	}

	logger.Log(logger.TagAgent, "AnalysisAgent processing request: %q", userPrompt)

	systemPrompt := "You are a Critical Analysis and Risk Assessment Agent in an A2A protocol network. " +
		"Analyze the provided findings, identify key strengths, operational vulnerabilities, security considerations, " +
		"and formulate strategic recommendations."

	respText, err := a.llm.Converse(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("analysis LLM inference failed: %w", err)
	}

	logger.Log(logger.TagAgent, "AnalysisAgent finished processing (%d bytes)", len(respText))

	res := a2a.NewTextMessage("agent", respText)
	return &res, nil
}
