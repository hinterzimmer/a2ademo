variable "harness_name" {
  description = "Name for the Bedrock AgentCore Harness"
  type        = string
  default     = "a2a_security_auditor"
}

variable "model_id" {
  description = "Bedrock foundation model ID"
  type        = string
  default     = "eu.anthropic.claude-haiku-4-5-20251001-v1:0"
}

variable "system_prompt" {
  description = "System prompt defining the autonomous agent persona"
  type        = string
  default     = "You are the SecurityAuditorAgent, an autonomous Cloud & Protocol Security Auditor operating in an A2A protocol network. Your mission is to perform rigorous zero-trust threat modeling, evaluate protocol boundaries and isolation, identify vulnerabilities, and formulate remediation controls."
}

variable "max_iterations" {
  description = "Maximum iterations for agent loops"
  type        = number
  default     = 10
}

variable "max_tokens" {
  description = "Max response tokens"
  type        = number
  default     = 4096
}

variable "timeout_seconds" {
  description = "Harness execution timeout"
  type        = number
  default     = 300
}

variable "tags" {
  description = "Tags applied to all harness resources"
  type        = map(string)
  default     = {
    Environment = "demo"
    ManagedBy   = "terraform"
    Protocol    = "a2a"
  }
}
