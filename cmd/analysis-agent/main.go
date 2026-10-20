package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"a2ademo/internal/agents/analysis"
	"a2ademo/pkg/a2a"
	"a2ademo/pkg/bedrock"
	"a2ademo/pkg/config"
	"a2ademo/pkg/logger"
)

func main() {
	mockFlag := flag.Bool("mock", false, "Use mock LLM instead of AWS Bedrock")
	flag.Parse()

	cfg := config.Load()
	port := cfg.AnalysisAgentPort
	baseURL := fmt.Sprintf("http://localhost:%s", port)

	var llm bedrock.LLMClient
	var err error

	if *mockFlag || !cfg.HasAWSCredentials() {
		logger.Log(logger.TagInfo, "Using mock LLM client for Analysis Agent")
		llm = bedrock.NewMockClient(cfg.BedrockModelID)
	} else {
		logger.Log(logger.TagInfo, "Connecting to AWS Bedrock (%s in %s)...", cfg.BedrockModelID, cfg.AWSRegion)
		llm, err = bedrock.NewClient(context.Background(), cfg)
		if err != nil {
			logger.Log(logger.TagError, "Failed to initialize Bedrock client: %v", err)
			os.Exit(1)
		}
	}

	agent := analysis.NewAgent(llm, baseURL)
	server := a2a.NewServer(agent.Card(), agent.HandleMessage)

	logger.Log(logger.TagInfo, "Analysis Agent listening on :%s (A2A discovery at %s/.well-known/agent-card.json)", port, baseURL)
	if err := http.ListenAndServe(":"+port, server.Handler()); err != nil {
		logger.Log(logger.TagError, "Server failed: %v", err)
		os.Exit(1)
	}
}
