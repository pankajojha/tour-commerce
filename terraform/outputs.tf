output "vpc_id" {
  description = "VPC ID"
  value       = module.network.vpc_id
}

output "database_endpoint" {
  description = "Aurora cluster endpoint"
  value       = module.database.cluster_endpoint
}

output "database_read_endpoint" {
  description = "Aurora cluster read endpoint"
  value       = module.database.cluster_read_endpoint
}

output "kms_key_id" {
  description = "KMS key ID"
  value       = module.security.kms_key_id
}

output "redis_endpoint" {
  description = "Redis cluster endpoint"
  value       = module.cache.redis_endpoint
}

output "cloudfront_distribution_domain" {
  description = "CloudFront distribution domain"
  value       = module.storage.cloudfront_domain
}

# Authentication Outputs
output "user_pool_id" {
  description = "Cognito User Pool ID"
  value       = module.auth.user_pool_id
}

output "user_pool_domain" {
  description = "Cognito User Pool domain"
  value       = module.auth.user_pool_domain
}

output "web_client_id" {
  description = "Web client ID"
  value       = module.auth.web_client_id
}

output "mobile_client_id" {
  description = "Mobile client ID"
  value       = module.auth.mobile_client_id
}

output "signup_lambda_arn" {
  description = "Signup Lambda ARN"
  value       = module.auth_lambdas.signup_lambda_arn
}

output "login_lambda_arn" {
  description = "Login Lambda ARN"
  value       = module.auth_lambdas.login_lambda_arn
}

output "passwordless_request_lambda_arn" {
  description = "Passwordless request Lambda ARN"
  value       = module.auth_lambdas.passwordless_request_lambda_arn
}

output "passwordless_verify_lambda_arn" {
  description = "Passwordless verify Lambda ARN"
  value       = module.auth_lambdas.passwordless_verify_lambda_arn
}

output "token_refresh_lambda_arn" {
  description = "Token refresh Lambda ARN"
  value       = module.auth_lambdas.token_refresh_lambda_arn
}

output "oauth_start_lambda_arn" {
  description = "OAuth start Lambda ARN"
  value       = module.auth_lambdas.oauth_start_lambda_arn
}

output "oauth_callback_lambda_arn" {
  description = "OAuth callback Lambda ARN"
  value       = module.auth_lambdas.oauth_callback_lambda_arn
}
