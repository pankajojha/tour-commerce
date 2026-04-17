output "auth_handler_lambda_arn" {
  description = "Auth handler Lambda ARN"
  value       = aws_lambda_function.auth_handler.arn
}

output "auth_handler_lambda_invoke_arn" {
  description = "Auth handler Lambda invoke ARN"
  value       = aws_lambda_function.auth_handler.invoke_arn
}

output "pre_signup_lambda_arn" {
  description = "Pre-signup trigger Lambda ARN"
  value       = aws_lambda_function.pre_signup.arn
}

output "oauth_start_lambda_arn" {
  description = "OAuth start Lambda ARN"
  value       = aws_lambda_function.oauth_start.arn
}

output "oauth_start_lambda_invoke_arn" {
  description = "OAuth start Lambda invoke ARN"
  value       = aws_lambda_function.oauth_start.invoke_arn
}

output "oauth_callback_lambda_arn" {
  description = "OAuth callback Lambda ARN"
  value       = aws_lambda_function.oauth_callback.arn
}

output "oauth_callback_lambda_invoke_arn" {
  description = "OAuth callback Lambda invoke ARN"
  value       = aws_lambda_function.oauth_callback.invoke_arn
}
