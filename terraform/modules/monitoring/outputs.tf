output "api_gateway_log_group_name" {
  description = "API Gateway log group name"
  value       = aws_cloudwatch_log_group.api_gateway.name
}

output "lambda_log_group_name" {
  description = "Lambda log group name"
  value       = aws_cloudwatch_log_group.lambda.name
}

output "ecs_log_group_name" {
  description = "ECS log group name"
  value       = aws_cloudwatch_log_group.ecs.name
}
