variable "aws_region" {
  description = "AWS region for Bedrock and Lambda resources"
  type        = string
  default     = "eu-central-1"
}

variable "harness_name" {
  description = "Identifier for the Bedrock AgentCore Harness"
  type        = string
  default     = "a2a_security_auditor"
}

variable "model_id" {
  description = "Foundation model identifier for Bedrock Harness and fallback inference"
  type        = string
  default     = "eu.anthropic.claude-haiku-4-5-20251001-v1:0"
}

variable "system_prompt" {
  description = "System prompt defining the autonomous SecurityAuditorAgent"
  type        = string
  default     = "You are the SecurityAuditorAgent, an autonomous Cloud & Protocol Security Auditor operating in an A2A protocol network. Your mission is to perform rigorous zero-trust threat modeling, evaluate protocol boundaries and isolation, identify vulnerabilities, and formulate remediation controls."
}

variable "tags" {
  description = "Resource tags"
  type        = map(string)
  default     = {
    Project     = "a2ademo"
    Environment = "demo"
    ManagedBy   = "terraform"
    Protocol    = "a2a"
  }
}
