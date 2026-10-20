output "harness_arn" {
  description = "ARN of the Bedrock AgentCore Harness"
  value       = aws_bedrockagentcore_harness.this.arn
}

output "harness_id" {
  description = "ID of the Bedrock AgentCore Harness"
  value       = aws_bedrockagentcore_harness.this.harness_id
}

output "harness_name" {
  description = "Name of the Bedrock AgentCore Harness"
  value       = aws_bedrockagentcore_harness.this.harness_name
}
