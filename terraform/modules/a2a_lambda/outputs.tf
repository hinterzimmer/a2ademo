output "lambda_function_url" {
  description = "Public Function URL of the A2A Agent Lambda"
  value       = aws_lambda_function_url.a2a_agent_url.function_url
}

output "function_name" {
  description = "Name of the A2A Agent Lambda function"
  value       = aws_lambda_function.a2a_agent.function_name
}

output "function_arn" {
  description = "ARN of the A2A Agent Lambda function"
  value       = aws_lambda_function.a2a_agent.arn
}
