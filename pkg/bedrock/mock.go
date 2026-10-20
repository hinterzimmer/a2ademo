package bedrock

import (
	"context"
	"fmt"
	"strings"

	"a2ademo/pkg/logger"
)

// MockClient provides simulated LLM responses for testing and offline runs.
type MockClient struct {
	modelID string
}

// NewMockClient creates a simulated LLMClient.
func NewMockClient(modelID string) *MockClient {
	if modelID == "" {
		modelID = "mock.bedrock-claude-3-5"
	}
	return &MockClient{modelID: modelID}
}

// ModelID returns the mock model name.
func (m *MockClient) ModelID() string {
	return m.modelID
}

// Converse generates context-appropriate mock responses based on the agent's role and prompts.
func (m *MockClient) Converse(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	logger.Log(logger.TagBedrock, "[MOCK] Simulating AWS Bedrock model: %s", m.modelID)

	sysLower := strings.ToLower(systemPrompt)

	if strings.Contains(sysLower, "orchestrator") || strings.Contains(sysLower, "synthe") {
		return fmt.Sprintf(
			"### Executive Synthesis\n\n"+
				"**Overview**: Multi-agent orchestration via the A2A protocol successfully decomposed the user goal and coordinated specialized sub-agents.\n\n"+
				"**Summary of Insights**:\n"+
				"- **Research Findings**: Decentralized protocols decouple agent execution from proprietary tool frameworks, utilizing standard Agent Cards for discovery and JSON-RPC for messaging.\n"+
				"- **Strategic Analysis**: Opacity and isolation protect agent internal state, while transport standardisation (HTTP/gRPC) provides vendor neutrality.\n\n"+
				"**Conclusion**: The A2A protocol provides a robust foundation for federated, heterogeneous agent networks without vendor lock-in.",
		), nil
	}

	if strings.Contains(sysLower, "auditor") || strings.Contains(sysLower, "securityauditor") {
		return fmt.Sprintf(
			"### Zero-Trust Security Audit & Protocol Compliance\n\n"+
				"1. **Threat Modeling (STRIDE Analysis)**:\n"+
				"   - Spoofing: Prevented by public key / token verification.\n"+
				"   - Tampering: JSON-RPC 2.0 messages are immutable across HTTPS boundaries.\n"+
				"   - Repudiation: Message IDs and UTC timestamps provide audit trails.\n"+
				"2. **Protocol Boundary & Isolation**:\n"+
				"   - Agents operate as opaque black boxes; internal tool sets and execution environments remain unexposed.\n"+
				"   - Incoming AgentCard metadata is safely validated before reasoning ingestion.\n"+
				"3. **Compliance Verdict**:\n"+
				"   - APPROVED: Compliant with zero-trust multi-agent architecture guidelines.",
		), nil
	}

	if strings.Contains(sysLower, "analyst") || strings.Contains(sysLower, "analy") {
		return fmt.Sprintf(
			"### Strategic Analysis & Risk Assessment\n\n"+
				"1. **Strengths**: Vendor neutrality, opacity preservation (agents don't need shared memory or internal state access), and protocol discoverability.\n"+
				"2. **Risks & Trade-offs**: Latency overhead across HTTP boundaries, input validation concerns (treating external AgentCards as untrusted), and decentralized error recovery.\n"+
				"3. **Recommendation**: Implement robust schema validation and mutual authentication (mTLS / JWT tokens) in production deployments.",
		), nil
	}

	// Default to research specialist response
	return fmt.Sprintf(
		"### Research Findings\n\n"+
			"- **State of the Art**: Contemporary multi-agent architectures leverage decentralized protocols like A2A to decouple agent execution from proprietary tool frameworks.\n"+
			"- **Key Technical Drivers**: Machine-readable Agent Cards (discovery), JSON-RPC standard messaging, and transport independence (HTTP/gRPC).\n"+
			"- **Ecosystem Observations**: Standards like MCP address tool/data binding, while A2A addresses inter-agent delegation and task negotiation.\n\n"+
			"Source Verification: Verified against open specification guidelines for query: %q",
		userPrompt,
	), nil
}
