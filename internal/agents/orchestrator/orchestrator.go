package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"a2ademo/pkg/a2a"
	"a2ademo/pkg/bedrock"
	"a2ademo/pkg/logger"
)

// Orchestrator coordinates specialized sub-agents over the A2A protocol.
type Orchestrator struct {
	llm         bedrock.LLMClient
	a2aClient   *a2a.Client
	researchURL string
	analysisURL string
	baseURL     string
}

// WorkflowResult holds the output of the full orchestration pipeline.
type WorkflowResult struct {
	Goal             string         `json:"goal"`
	ResearchCard     *a2a.AgentCard `json:"research_agent"`
	AnalysisCard     *a2a.AgentCard `json:"analysis_agent"`
	ResearchFindings string         `json:"research_findings"`
	AnalysisFindings string         `json:"analysis_findings"`
	FinalSynthesis   string         `json:"final_synthesis"`
}

// NewOrchestrator creates a new Orchestrator instance.
func NewOrchestrator(llm bedrock.LLMClient, client *a2a.Client, researchURL, analysisURL, baseURL string) *Orchestrator {
	return &Orchestrator{
		llm:         llm,
		a2aClient:   client,
		researchURL: researchURL,
		analysisURL: analysisURL,
		baseURL:     baseURL,
	}
}

// Card returns the Orchestrator's AgentCard.
func (o *Orchestrator) Card() a2a.AgentCard {
	return a2a.AgentCard{
		Name:            "OrchestratorAgent",
		Description:     "Main application orchestrator coordinating specialized sub-agents across the A2A protocol network.",
		URL:             o.baseURL,
		Version:         "1.0.0",
		ProtocolVersion: "0.2.0",
		Skills: []a2a.AgentSkill{
			{
				ID:          "coordinate_workflow",
				Name:        "Coordinate Multi-Agent Workflow",
				Description: "Discovers, delegates, and synthesizes multi-step agent tasks via A2A protocol.",
				Tags:        []string{"orchestration", "multi-agent", "coordination"},
			},
		},
	}
}

// DelegateToAgent discovers a target A2A agent via its AgentCard and executes a task over JSON-RPC 2.0.
func (o *Orchestrator) DelegateToAgent(ctx context.Context, agentURL string, prompt string) (*a2a.Message, *a2a.AgentCard, error) {
	logger.Log(logger.TagDiscovery, "Querying target agent at %s/.well-known/agent-card.json", agentURL)
	card, err := o.a2aClient.Discover(ctx, agentURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to discover agent at %s: %w", agentURL, err)
	}

	logger.Log(logger.TagDiscovery, "✓ Discovered %s v%s (skills: %s)",
		card.Name, card.Version, formatSkills(card.Skills))

	logger.Log(logger.TagRequest, "Orchestrator -> %s (%s/rpc)", card.Name, agentURL)
	logger.Log(logger.TagRequest, "Payload: %q", prompt)

	respMsg, err := o.a2aClient.SendMessage(ctx, agentURL, prompt, nil)
	if err != nil {
		return nil, card, fmt.Errorf("A2A call to %s failed: %w", card.Name, err)
	}

	resText := extractText(respMsg)
	logger.Log(logger.TagResponse, "%s -> Orchestrator: Response received (%d bytes)", card.Name, len(resText))

	return respMsg, card, nil
}

// Run executes the complete multi-agent workflow:
// 1. Discover sub-agents via their Agent Cards
// 2. Delegate research task to ResearchAgent over A2A
// 3. Delegate analysis task to AnalysisAgent over A2A
// 4. Synthesize final comprehensive report using Bedrock
func (o *Orchestrator) Run(ctx context.Context, goal string) (*WorkflowResult, error) {
	logger.Section("ORCHESTRATOR: Starting Multi-Agent Task")
	logger.Log(logger.TagOrchestrator, "Goal: %q", goal)

	// Step 1: Agent Discovery
	logger.Section("PHASE 1: A2A Protocol Discovery")
	logger.Log(logger.TagDiscovery, "Querying ResearchAgent at %s/.well-known/agent-card.json", o.researchURL)
	resCard, err := o.a2aClient.Discover(ctx, o.researchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover ResearchAgent: %w", err)
	}
	logger.Log(logger.TagDiscovery, "✓ Discovered %s v%s (skills: %s)",
		resCard.Name, resCard.Version, formatSkills(resCard.Skills))

	logger.Log(logger.TagDiscovery, "Querying AnalysisAgent at %s/.well-known/agent-card.json", o.analysisURL)
	anaCard, err := o.a2aClient.Discover(ctx, o.analysisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover AnalysisAgent: %w", err)
	}
	logger.Log(logger.TagDiscovery, "✓ Discovered %s v%s (skills: %s)",
		anaCard.Name, anaCard.Version, formatSkills(anaCard.Skills))

	// Step 2: Delegate to Research Agent
	logger.Section("PHASE 2: Delegating to ResearchAgent via A2A JSON-RPC")
	resPrompt := fmt.Sprintf("Perform in-depth research and gather core architectural facts for: %s", goal)
	logger.Log(logger.TagRequest, "Orchestrator -> %s (%s/rpc)", resCard.Name, o.researchURL)
	logger.Log(logger.TagRequest, "Payload: %q", resPrompt)

	resMsg, err := o.a2aClient.SendMessage(ctx, o.researchURL, resPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("A2A call to ResearchAgent failed: %w", err)
	}

	resText := extractText(resMsg)
	logger.Log(logger.TagResponse, "%s -> Orchestrator: Response received (%d bytes)", resCard.Name, len(resText))

	// Step 3: Delegate to Analysis Agent
	logger.Section("PHASE 3: Delegating to AnalysisAgent via A2A JSON-RPC")
	anaPrompt := fmt.Sprintf(
		"Based on the following research findings for the goal %q, conduct a critical analysis, identify trade-offs, security considerations, and provide strategic recommendations:\n\n%s",
		goal, resText,
	)
	logger.Log(logger.TagRequest, "Orchestrator -> %s (%s/rpc)", anaCard.Name, o.analysisURL)
	logger.Log(logger.TagRequest, "Payload length: %d bytes (forwarding research output)", len(anaPrompt))

	anaMsg, err := o.a2aClient.SendMessage(ctx, o.analysisURL, anaPrompt, nil)
	if err != nil {
		return nil, fmt.Errorf("A2A call to AnalysisAgent failed: %w", err)
	}

	anaText := extractText(anaMsg)
	logger.Log(logger.TagResponse, "%s -> Orchestrator: Response received (%d bytes)", anaCard.Name, len(anaText))

	// Step 4: Synthesize Final Report
	logger.Section("PHASE 4: Orchestrator Bedrock Synthesis")
	logger.Log(logger.TagOrchestrator, "Synthesizing research and analysis with Bedrock LLM...")

	synthSystem := "You are the Lead Multi-Agent Orchestrator. " +
		"Synthesize the research findings and critical analysis from your specialized sub-agents into an executive, highly actionable report."

	synthUser := fmt.Sprintf(
		"User Goal: %s\n\n"+
			"--- [SUB-AGENT: Research Findings] ---\n%s\n\n"+
			"--- [SUB-AGENT: Analysis & Risk Assessment] ---\n%s\n\n"+
			"Produce the final comprehensive report.",
		goal, resText, anaText,
	)

	finalReport, err := o.llm.Converse(ctx, synthSystem, synthUser)
	if err != nil {
		return nil, fmt.Errorf("orchestrator synthesis failed: %w", err)
	}

	logger.Log(logger.TagOrchestrator, "✓ Multi-agent orchestration completed (%d bytes final synthesis)", len(finalReport))

	return &WorkflowResult{
		Goal:             goal,
		ResearchCard:     resCard,
		AnalysisCard:     anaCard,
		ResearchFindings: resText,
		AnalysisFindings: anaText,
		FinalSynthesis:   finalReport,
	}, nil
}

func formatSkills(skills []a2a.AgentSkill) string {
	ids := make([]string, len(skills))
	for i, s := range skills {
		ids[i] = s.ID
	}
	return strings.Join(ids, ", ")
}

func extractText(msg *a2a.Message) string {
	if msg == nil || len(msg.Parts) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, p := range msg.Parts {
		sb.WriteString(p.Text)
	}
	return sb.String()
}
