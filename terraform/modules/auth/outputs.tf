output "user_pool_id" {
  description = "Cognito User Pool ID"
  value       = aws_cognito_user_pool.main.id
}

output "user_pool_arn" {
  description = "Cognito User Pool ARN"
  value       = aws_cognito_user_pool.main.arn
}

output "user_pool_endpoint" {
  description = "Cognito User Pool endpoint"
  value       = aws_cognito_user_pool.main.endpoint
}

output "user_pool_domain" {
  description = "Cognito User Pool domain"
  value       = aws_cognito_user_pool_domain.main.domain
}

output "web_client_id" {
  description = "Web client ID"
  value       = aws_cognito_user_pool_client.web.id
}

output "mobile_client_id" {
  description = "Mobile client ID"
  value       = aws_cognito_user_pool_client.mobile.id
}

output "oauth_credentials_secret_arn" {
  description = "OAuth credentials secret ARN"
  value       = var.enable_social_login ? aws_secretsmanager_secret.oauth_credentials[0].arn : ""
}
