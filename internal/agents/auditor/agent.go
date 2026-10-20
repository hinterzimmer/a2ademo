package auditor

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"a2ademo/pkg/a2a"
	"a2ademo/pkg/bedrock"
	"a2ademo/pkg/logger"
)

// Agent implements the SecurityAuditorAgent service.
type Agent struct {
	llm      bedrock.LLMClient
	baseURL  string
	sessions sync.Map // map[string][]string for multi-turn conversational memory
}

// NewAgent creates a new SecurityAuditorAgent instance.
func NewAgent(llm bedrock.LLMClient, baseURL string) *Agent {
	return &Agent{
		llm:     llm,
		baseURL: baseURL,
	}
}

// Card returns the A2A AgentCard describing the agent's identity and capabilities.
func (a *Agent) Card() a2a.AgentCard {
	return a2a.AgentCard{
		Name:            "SecurityAuditorAgent",
		Description:     "Autonomous cloud-deployed A2A agent for zero-trust security auditing, protocol verification, and compliance.",
		URL:             a.baseURL,
		Version:         "1.0.0",
		ProtocolVersion: "0.2.0",
		Skills: []a2a.AgentSkill{
			{
				ID:          "security_audit",
				Name:        "Security Audit",
				Description: "Performs zero-trust threat modeling and security posture analysis on multi-agent architectures.",
				Tags:        []string{"security", "audit", "threat-modeling"},
			},
			{
				ID:          "protocol_verification",
				Name:        "Protocol Verification",
				Description: "Validates A2A protocol message safety, sandbox integrity, and isolation boundaries.",
				Tags:        []string{"a2a", "verification", "sandbox"},
			},
			{
				ID:          "compliance_certification",
				Name:        "Compliance Certification",
				Description: "Audits distributed agent communication against enterprise security and governance standards.",
				Tags:        []string{"compliance", "governance", "soc2"},
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

	contextID := msg.ContextID
	if contextID == "" && ctxParams != nil && ctxParams["contextId"] != nil {
		contextID = fmt.Sprintf("%v", ctxParams["contextId"])
	}

	logger.Log(logger.TagAgent, "SecurityAuditorAgent processing request (context: %s): %q", contextID, userPrompt)

	systemPrompt := "You are the SecurityAuditorAgent, an autonomous Cloud & Protocol Security Auditor operating in an A2A protocol network. " +
		"Your mission is to perform rigorous zero-trust threat modeling, evaluate protocol boundaries and isolation, " +
		"identify vulnerabilities (prompt injection, unauthorized delegation, data exfiltration), " +
		"and formulate concrete remediation controls and security audit sign-offs."

	fullPrompt := userPrompt
	if contextID != "" {
		if rawHistory, ok := a.sessions.Load(contextID); ok {
			history := rawHistory.([]string)
			if len(history) > 0 {
				fullPrompt = fmt.Sprintf("Previous session dialogue:\n%s\n\nCurrent user message:\n%s", strings.Join(history, "\n"), userPrompt)
			}
		}
	}

	respText, err := a.llm.Converse(ctx, systemPrompt, fullPrompt)
	if err != nil {
		return nil, fmt.Errorf("security auditor LLM inference failed: %w", err)
	}

	if contextID != "" {
		var history []string
		if rawHistory, ok := a.sessions.Load(contextID); ok {
			history = rawHistory.([]string)
		}
		history = append(history, fmt.Sprintf("User: %s", userPrompt), fmt.Sprintf("Agent: %s", respText))
		a.sessions.Store(contextID, history)
	}

	logger.Log(logger.TagAgent, "SecurityAuditorAgent finished processing (%d bytes)", len(respText))

	res := a2a.NewTextMessageWithContext("agent", respText, contextID)
	return &res, nil
}
