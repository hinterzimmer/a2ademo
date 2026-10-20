output "harness_arn" {
  description = "ARN of the Bedrock AgentCore Harness"
  value       = module.bedrock_harness.harness_arn
}

output "harness_name" {
  description = "Name of the Bedrock AgentCore Harness"
  value       = module.bedrock_harness.harness_name
}

output "lambda_function_url" {
  description = "Base URL of the A2A Lambda Function URL"
  value       = module.a2a_lambda.lambda_function_url
}

output "agent_card_url" {
  description = "Standard A2A AgentCard discovery endpoint"
  value       = "${trimsuffix(module.a2a_lambda.lambda_function_url, "/")}/.well-known/agent-card.json"
}

output "rpc_endpoint_url" {
  description = "Standard A2A JSON-RPC 2.0 communication endpoint"
  value       = "${trimsuffix(module.a2a_lambda.lambda_function_url, "/")}/rpc"
}
