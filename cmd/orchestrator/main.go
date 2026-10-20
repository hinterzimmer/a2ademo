package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"a2ademo/internal/agents/orchestrator"
	"a2ademo/pkg/a2a"
	"a2ademo/pkg/bedrock"
	"a2ademo/pkg/config"
	"a2ademo/pkg/logger"
)

func main() {
	mockFlag := flag.Bool("mock", false, "Use mock LLM instead of AWS Bedrock")
	queryFlag := flag.String("query", "Analyze the impact of Agent-to-Agent (A2A) protocol on enterprise AI ecosystems", "Goal/task to orchestrate")
	flag.Parse()

	cfg := config.Load()
	researchURL := fmt.Sprintf("http://localhost:%s", cfg.ResearchAgentPort)
	analysisURL := fmt.Sprintf("http://localhost:%s", cfg.AnalysisAgentPort)
	orchURL := fmt.Sprintf("http://localhost:%s", cfg.OrchestratorPort)

	var llm bedrock.LLMClient
	var err error

	if *mockFlag || !cfg.HasAWSCredentials() {
		logger.Log(logger.TagInfo, "Using mock LLM client for Orchestrator")
		llm = bedrock.NewMockClient(cfg.BedrockModelID)
	} else {
		logger.Log(logger.TagInfo, "Connecting to AWS Bedrock (%s in %s)...", cfg.BedrockModelID, cfg.AWSRegion)
		llm, err = bedrock.NewClient(context.Background(), cfg)
		if err != nil {
			logger.Log(logger.TagError, "Failed to initialize Bedrock client: %v", err)
			os.Exit(1)
		}
	}

	client := a2a.NewClient(nil)
	orch := orchestrator.NewOrchestrator(llm, client, researchURL, analysisURL, orchURL)

	res, err := orch.Run(context.Background(), *queryFlag)
	if err != nil {
		logger.Log(logger.TagError, "Orchestration failed: %v", err)
		os.Exit(1)
	}

	logger.Section("FINAL SYNTHESIZED MULTI-AGENT REPORT")
	fmt.Println(res.FinalSynthesis)
}
