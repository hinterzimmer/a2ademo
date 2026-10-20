package research

import (
	"context"
	"fmt"

	"a2ademo/pkg/a2a"
	"a2ademo/pkg/bedrock"
	"a2ademo/pkg/logger"
)

// Agent implements the Research Agent service.
type Agent struct {
	llm     bedrock.LLMClient
	baseURL string
}

// NewAgent creates a new Research Agent instance.
func NewAgent(llm bedrock.LLMClient, baseURL string) *Agent {
	return &Agent{
		llm:     llm,
		baseURL: baseURL,
	}
}

// Card returns the A2A AgentCard describing the agent's capabilities.
func (a *Agent) Card() a2a.AgentCard {
	return a2a.AgentCard{
		Name:            "ResearchAgent",
		Description:     "Specialized agent for deep technical research, data collection, and fact-finding.",
		URL:             a.baseURL,
		Version:         "1.0.0",
		ProtocolVersion: "0.2.0",
		Skills: []a2a.AgentSkill{
			{
				ID:          "research_topic",
				Name:        "Research Topic",
				Description: "Conducts structured technical research on specified subjects.",
				Tags:        []string{"research", "data-collection", "technical"},
			},
			{
				ID:          "gather_data",
				Name:        "Gather Architecture Data",
				Description: "Collects architectural patterns, specifications, and ecosystem benchmarks.",
				Tags:        []string{"architecture", "specifications"},
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

	logger.Log(logger.TagAgent, "ResearchAgent processing request: %q", userPrompt)

	systemPrompt := "You are a Research Specialist Agent in an A2A (Agent-to-Agent) protocol network. " +
		"Your responsibility is to provide comprehensive, factual, and structured technical data on the requested topic. " +
		"Format your output cleanly with bullet points and clear sections."

	respText, err := a.llm.Converse(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("research LLM inference failed: %w", err)
	}

	logger.Log(logger.TagAgent, "ResearchAgent finished processing (%d bytes)", len(respText))

	res := a2a.NewTextMessage("agent", respText)
	return &res, nil
}
