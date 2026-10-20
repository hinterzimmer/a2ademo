package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"a2ademo/internal/agents/analysis"
	"a2ademo/internal/agents/orchestrator"
	"a2ademo/internal/agents/research"
	"a2ademo/pkg/a2a"
	"a2ademo/pkg/bedrock"
	"a2ademo/pkg/config"
	"a2ademo/pkg/logger"
)

func main() {
	mockFlag := flag.Bool("mock", false, "Force mock LLM mode even if AWS credentials exist")
	queryFlag := flag.String("query", "Evaluate the benefits, architectural patterns, and security risks of the Agent-to-Agent (A2A) protocol", "Query/goal for the multi-agent system")
	flag.Parse()

	cfg := config.Load()

	var llm bedrock.LLMClient
	var err error

	useMock := *mockFlag || !cfg.HasAWSCredentials()
	if useMock {
		if !cfg.HasAWSCredentials() && !*mockFlag {
			logger.Log(logger.TagInfo, "No AWS credentials found in .env; running in MOCK mode")
		} else {
			logger.Log(logger.TagInfo, "Explicit --mock flag set; running in MOCK mode")
		}
		llm = bedrock.NewMockClient(cfg.BedrockModelID)
	} else {
		logger.Log(logger.TagInfo, "AWS credentials loaded from .env. Initializing AWS Bedrock runtime...")
		logger.Log(logger.TagInfo, "Region: %s | Model: %s", cfg.AWSRegion, cfg.BedrockModelID)
		llm, err = bedrock.NewClient(context.Background(), cfg)
		if err != nil {
			logger.Log(logger.TagError, "Failed to initialize AWS Bedrock client: %v", err)
			os.Exit(1)
		}
	}

	// Start Research Agent listener
	resListener, err := net.Listen("tcp", "127.0.0.1:"+cfg.ResearchAgentPort)
	if err != nil {
		// Fallback to dynamic port if port is taken
		resListener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			logger.Log(logger.TagError, "Failed to listen for ResearchAgent: %v", err)
			os.Exit(1)
		}
	}
	defer resListener.Close()
	researchURL := "http://" + resListener.Addr().String()

	resAgent := research.NewAgent(llm, researchURL)
	resServer := a2a.NewServer(resAgent.Card(), resAgent.HandleMessage)
	resHTTPServer := &http.Server{Handler: resServer.Handler()}
	go func() {
		_ = resHTTPServer.Serve(resListener)
	}()
	logger.Log(logger.TagInfo, "Started ResearchAgent at %s", researchURL)

	// Start Analysis Agent listener
	anaListener, err := net.Listen("tcp", "127.0.0.1:"+cfg.AnalysisAgentPort)
	if err != nil {
		anaListener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			logger.Log(logger.TagError, "Failed to listen for AnalysisAgent: %v", err)
			os.Exit(1)
		}
	}
	defer anaListener.Close()
	analysisURL := "http://" + anaListener.Addr().String()

	anaAgent := analysis.NewAgent(llm, analysisURL)
	anaServer := a2a.NewServer(anaAgent.Card(), anaAgent.HandleMessage)
	anaHTTPServer := &http.Server{Handler: anaServer.Handler()}
	go func() {
		_ = anaHTTPServer.Serve(anaListener)
	}()
	logger.Log(logger.TagInfo, "Started AnalysisAgent at %s", analysisURL)

	// Wait briefly for listeners to accept connections
	time.Sleep(50 * time.Millisecond)

	orchURL := fmt.Sprintf("http://localhost:%s", cfg.OrchestratorPort)
	a2aClient := a2a.NewClient(nil)
	orch := orchestrator.NewOrchestrator(llm, a2aClient, researchURL, analysisURL, orchURL)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := orch.Run(ctx, *queryFlag)
	if err != nil {
		logger.Log(logger.TagError, "Multi-agent orchestration workflow failed: %v", err)
		os.Exit(1)
	}

	logger.Section("FINAL SYNTHESIZED MULTI-AGENT REPORT")
	fmt.Printf("%s\n\n", result.FinalSynthesis)

	// Graceful shutdown
	_ = resHTTPServer.Shutdown(context.Background())
	_ = anaHTTPServer.Shutdown(context.Background())
	logger.Log(logger.TagInfo, "Demo run completed successfully. Sub-agents terminated.")
}
