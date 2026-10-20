variable "function_name" {
  description = "Name for the A2A Lambda function"
  type        = string
  default     = "a2a-security-auditor-agent"
}

variable "harness_arn" {
  description = "ARN of the Bedrock AgentCore Harness"
  type        = string
}

variable "model_id" {
  description = "Bedrock model ID for direct fallback invocations"
  type        = string
  default     = "eu.anthropic.claude-haiku-4-5-20251001-v1:0"
}

variable "lambda_source_dir" {
  description = "Path to directory containing Python Lambda source files"
  type        = string
}

variable "agent_name" {
  description = "Name of the A2A Agent"
  type        = string
  default     = "SecurityAuditorAgent"
}

variable "agent_description" {
  description = "Description of the A2A Agent"
  type        = string
  default     = "Autonomous cloud-deployed A2A agent running on AWS Bedrock Harness for zero-trust security auditing, protocol verification, and compliance."
}

variable "cors_allowed_origins" {
  description = "Allowed origins for CORS"
  type        = list(string)
  default     = ["*"]
}

variable "tags" {
  description = "Tags applied to all Lambda resources"
  type        = map(string)
  default     = {
    Environment = "demo"
    ManagedBy   = "terraform"
    Protocol    = "a2a"
  }
}
