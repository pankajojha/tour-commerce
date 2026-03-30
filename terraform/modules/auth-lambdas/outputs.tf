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
