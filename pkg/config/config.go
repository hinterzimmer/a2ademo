package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config holds project configuration loaded from environment or .env.
type Config struct {
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSSessionToken    string
	AWSRegion          string
	BedrockModelID     string
	OrchestratorPort   string
	ResearchAgentPort  string
	AnalysisAgentPort  string
	AuditorAgentPort   string
	HarnessAgentURL    string
}

// Load loads configuration from .env and environment variables.
// It searches for .env in the current directory and parent directories.
func Load() *Config {
	loadEnvFile()

	region := os.Getenv("AWS_DEFAULT_REGION")
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}
	if region == "" {
		region = "us-east-1"
	}

	modelID := os.Getenv("BEDROCK_MODEL_ID")
	if modelID == "" {
		modelID = "anthropic.claude-3-5-sonnet-20241022-v2:0"
	}

	orchPort := os.Getenv("ORCHESTRATOR_PORT")
	if orchPort == "" {
		orchPort = "8000"
	}

	resPort := os.Getenv("RESEARCH_AGENT_PORT")
	if resPort == "" {
		resPort = "8001"
	}

	anaPort := os.Getenv("ANALYSIS_AGENT_PORT")
	if anaPort == "" {
		anaPort = "8002"
	}

	audPort := os.Getenv("AUDITOR_AGENT_PORT")
	if audPort == "" {
		audPort = "8003"
	}

	harnessURL := os.Getenv("HARNESS_AGENT_URL")

	return &Config{
		AWSAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AWSSessionToken:    os.Getenv("AWS_SESSION_TOKEN"),
		AWSRegion:          region,
		BedrockModelID:     modelID,
		OrchestratorPort:   orchPort,
		ResearchAgentPort:  resPort,
		AnalysisAgentPort:  anaPort,
		AuditorAgentPort:   audPort,
		HarnessAgentURL:    harnessURL,
	}
}

// HasAWSCredentials returns true if AWS access key and secret key are provided.
func (c *Config) HasAWSCredentials() bool {
	return c.AWSAccessKeyID != "" && c.AWSSecretAccessKey != ""
}

func loadEnvFile() {
	// Try current directory
	if err := godotenv.Load(".env"); err == nil {
		return
	}

	// Walk up to find .env if running from a subpackage (e.g., test/e2e)
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	for i := 0; i < 5; i++ {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			_ = godotenv.Load(envPath)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}
