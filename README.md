# A2A Protocol Multi-Agent Demo (Go & AWS Bedrock Harness)

A multi-agent demonstration project implementing the **Agent-to-Agent (A2A)** protocol, powered by **AWS Bedrock** (`bedrockruntime` Converse API) and a cloud-deployable **AWS Bedrock AgentCore Harness** on **AWS Lambda**.

---

## Architecture Overview

```
                      ┌──────────────────────┐
                      │      User / CLI      │
                      └──────────┬───────────┘
                                 │
                                 ▼
                      ┌──────────────────────┐
                      │  Orchestrator Agent  │
                      │       (:8000)        │
                      └──────┬───┬───┬───────┘
                             │   │   │
          ┌──────────────────┘   │   └──────────────────┐
          │                      │                      │
     1. A2A JSON-RPC        2. A2A JSON-RPC        3. A2A JSON-RPC
          ▼                      ▼                      ▼
┌──────────────────┐   ┌──────────────────┐   ┌───────────────────────────┐
│  Research Agent  │   │  Analysis Agent  │   │   Security Auditor Agent  │
│     (:8001)      │   │     (:8002)      │   │ (:8003 or AWS Lambda URL) │
└─────────┬────────┘   └─────────┬────────┘   └──────────────┬────────────┘
          │                      │                           │
          ▼                      ▼                           ▼
┌─────────────────────────────────────────┐   ┌───────────────────────────┐
│      Amazon Bedrock Converse API        │   │    AWS Bedrock Harness    │
│ (eu.anthropic.claude-haiku-4-5-20251001)│   │ (aws_bedrockagentcore_...)│
└─────────────────────────────────────────┘   └───────────────────────────┘
```

### The 4 Agents
1. **Orchestrator Agent** (`:8000`): Coordinates sub-agents, delegates tasks, and synthesizes executive reports.
2. **Research Agent** (`:8001`): Specialized in data gathering, technical research, and fact-finding (`research_topic`, `gather_data`).
3. **Analysis Agent** (`:8002`): Specialized in trade-off evaluation, risk assessment, and synthesis (`analyze_findings`, `risk_assessment`).
4. **Security Auditor Agent** (`:8003` or **AWS Lambda Function URL**): Autonomous cloud-deployed agent running via **AWS Bedrock Harness** performing zero-trust threat modeling, protocol verification, and compliance auditing (`security_audit`, `protocol_verification`, `compliance_certification`).

---

## Project Structure

```
a2ademo/
├── .env.example              # Configuration template
├── .env                      # Local credentials (AWS Bedrock & ports)
├── go.mod                    # Go module definition
├── README.md                 # Complete documentation & usage guide
├── cmd/
│   ├── demo/main.go          # All-in-one runner (starts sub-agents + orchestrator)
│   ├── orchestrator/main.go  # Standalone Orchestrator service / CLI
│   ├── research-agent/main.go# Standalone Research Agent service (:8001)
│   ├── analysis-agent/main.go# Standalone Analysis Agent service (:8002)
│   └── auditor-agent/main.go # Standalone Security Auditor Agent service (:8003)
├── pkg/
│   ├── a2a/                  # Core A2A protocol: types, HTTP client, and server handler
│   ├── bedrock/              # AWS Bedrock Converse API client & mock client
│   ├── config/               # Configuration & .env file loader
│   └── logger/               # Color-coded structured trace logging
├── internal/
│   └── agents/
│       ├── orchestrator/     # Planning, discovery, task delegation, and synthesis
│       ├── research/         # Research agent service implementation
│       ├── analysis/         # Analysis agent service implementation
│       └── auditor/          # Security auditor agent service implementation
├── test/
    ├── integration_test.go   # Fast in-memory integration test (runs in normal suite)
    └── e2e/
        ├── e2e_test.go       # Live multi-agent Bedrock E2E tests (//go:build e2e)
        └── e2e_harness_test.go# Orchestrator <-> Harness Agent full A2A test (//go:build e2e)
└── terraform/                # Infrastructure for deploying Bedrock Harness on AWS
    ├── main.tf               # Root Terraform module
    ├── variables.tf          # Configurable variables (region, model_id, etc.)
    ├── outputs.tf            # Output URLs (Function URL, AgentCard, RPC endpoint)
    ├── lambda_src/           # Python 3.12 Lambda implementation of A2A protocol
    │   └── index.py          # A2A AgentCard & JSON-RPC 2.0 handler invoking Bedrock
    └── modules/
        ├── bedrock_harness/  # aws_bedrockagentcore_harness & IAM role
        └── a2a_lambda/       # AWS Lambda function, Function URL & CORS
```

---

## Deploying the Bedrock Harness Agent on AWS (Terraform)

You can deploy the autonomous **Security Auditor Agent** directly onto AWS as a serverless A2A agent powered by Bedrock Harness and Lambda:

```bash
cd terraform
terraform init
terraform apply
```

Outputs will display:
```
agent_card_url       = "https://<unique-id>.lambda-url.<region>.on.aws/.well-known/agent-card.json"
rpc_endpoint_url     = "https://<unique-id>.lambda-url.<region>.on.aws/rpc"
lambda_function_url  = "https://<unique-id>.lambda-url.<region>.on.aws/"
harness_arn          = "arn:aws:bedrock-agentcore:..."
```

Copy `lambda_function_url` into your root `.env`:
```env
HARNESS_AGENT_URL=https://<unique-id>.lambda-url.eu-central-1.on.aws/
```

---

## How to Run the Go Demo

### 1. All-in-One Live Demo
```bash
go run ./cmd/demo
```

With custom goal:
```bash
go run ./cmd/demo --query "Analyze the scalability and zero-trust security of A2A protocol networks"
```

Offline mock mode (runs without calling AWS):
```bash
go run ./cmd/demo --mock
```

### 2. Running Agents Separately Across Multiple Terminals
- **Terminal 1**: `go run ./cmd/research-agent` (runs on `:8001`)
- **Terminal 2**: `go run ./cmd/analysis-agent` (runs on `:8002`)
- **Terminal 3**: `go run ./cmd/auditor-agent` (runs on `:8003`)
- **Terminal 4**: `go run ./cmd/orchestrator --query "Perform architecture review of A2A mesh"`

---

## Testing Guide

### Normal Tests (Fast & Offline)
Runs fast unit and mock integration tests in sub-second time. Automatically excludes all E2E tests:
```bash
go test -v ./...
```

### Python Unit Tests (Lambda Handler)
Runs 11 standalone unit tests against synthetic Lambda events in milliseconds (verifies SHA-256 session normalization, ARN scrubbing, base64 decoding, 404/405 routing, and JSON-RPC 2.0 error handling):
```bash
python3 -m unittest terraform/lambda_src/test_handler.py -v
```

### Live E2E Tests (Against AWS Bedrock)
The E2E tests are marked with `//go:build e2e` and use credentials from `.env`:

#### 1. Full E2E Test Suite:
```bash
go test -tags=e2e -v ./test/e2e/...
```

#### 2. Live Deployed Lambda A2A Protocol & Stateful Session Test:
Tests the live cloud-deployed Lambda function URL. Strictly requires `HARNESS_AGENT_URL` in `.env` and skips explicitly if unset (no silent in-process fallbacks):
```bash
go test -tags=e2e -v -run TestLiveDeployedLambdaA2A ./test/e2e/...
```

This test rigorously verifies:
- **Phase 1: Discovery & URL Match (`GET /.well-known/agent-card.json`)**: Validates `AgentCard` schema, protocol version `0.2.0`, skills, and asserts `card.URL == HARNESS_AGENT_URL`.
- **Phase 2: HTTP Transport & Routing**: Validates `GET /rpc` returns `405 Method Not Allowed`, and unrecognized routes return `404 Not Found`.
- **Phase 3: JSON-RPC 2.0 Error Matrix**: Validates `-32700` (parse error returns `id: null`), `-32600` (invalid version echoes request ID), `-32601` (method not found echoes numeric ID), and `-32602` (invalid params on empty parts).
- **Phase 4: Multi-Turn Bedrock Harness Session Memory & Isolation**:
  - **Turn 1**: Initializes session with dynamic secret token and `contextId`.
  - **Turn 2 (Positive Control)**: Asks to recall secret using the same `contextId`. Asserts Bedrock recalls it across turns.
  - **Turn 3 (Negative Control)**: Queries secret under a fresh, isolated `contextId`. Asserts secret is NOT recalled across session boundaries.

#### 3. In-Process Go Harness Agent Test:
Validates the in-process Go implementation of the Harness Agent with multi-turn session memory:
```bash
go test -tags=e2e -v -run TestInProcessA2AHarnessAgent ./test/e2e/...
```

#### 4. Orchestrator-to-Harness Delegation & Synthesis Test:
Verifies that the Orchestrator discovers the Harness agent, delegates a task, and synthesizes the findings:
```bash
go test -tags=e2e -v -run TestLiveOrchestratorToHarnessIntegration ./test/e2e/...
```
