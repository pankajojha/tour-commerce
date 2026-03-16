output "api_id" {
  description = "API Gateway REST API ID"
  value       = aws_api_gateway_rest_api.main.id
}

output "api_execution_arn" {
  description = "API Gateway execution ARN"
  value       = aws_api_gateway_rest_api.main.execution_arn
}

output "production_invoke_url" {
  description = "Production stage invoke URL"
  value       = aws_api_gateway_stage.production.invoke_url
}

output "dev_invoke_url" {
  description = "Dev stage invoke URL"
  value       = aws_api_gateway_stage.dev.invoke_url
}

output "api_root_resource_id" {
  description = "API Gateway root resource ID"
  value       = aws_api_gateway_rest_api.main.root_resource_id
}

output "v1_resource_id" {
  description = "v1 resource ID for future service integrations"
  value       = aws_api_gateway_resource.v1.id
}

output "destinations_resource_id" {
  description = "Destinations resource ID"
  value       = aws_api_gateway_resource.destinations.id
}

output "excursions_resource_id" {
  description = "Excursions resource ID"
  value       = aws_api_gateway_resource.excursions.id
}

output "bookings_resource_id" {
  description = "Bookings resource ID"
  value       = aws_api_gateway_resource.bookings.id
}

output "cognito_authorizer_id" {
  description = "Cognito authorizer ID for protected endpoints"
  value       = aws_api_gateway_authorizer.cognito.id
}
