output "signup_lambda_arn" {
  description = "Signup Lambda ARN"
  value       = aws_lambda_function.signup.arn
}

output "signup_lambda_invoke_arn" {
  description = "Signup Lambda invoke ARN"
  value       = aws_lambda_function.signup.invoke_arn
}

output "login_lambda_arn" {
  description = "Login Lambda ARN"
  value       = aws_lambda_function.login.arn
}

output "login_lambda_invoke_arn" {
  description = "Login Lambda invoke ARN"
  value       = aws_lambda_function.login.invoke_arn
}

output "passwordless_request_lambda_arn" {
  description = "Passwordless request Lambda ARN"
  value       = aws_lambda_function.passwordless_request.arn
}

output "passwordless_request_lambda_invoke_arn" {
  description = "Passwordless request Lambda invoke ARN"
  value       = aws_lambda_function.passwordless_request.invoke_arn
}

output "passwordless_verify_lambda_arn" {
  description = "Passwordless verify Lambda ARN"
  value       = aws_lambda_function.passwordless_verify.arn
}

output "passwordless_verify_lambda_invoke_arn" {
  description = "Passwordless verify Lambda invoke ARN"
  value       = aws_lambda_function.passwordless_verify.invoke_arn
}

output "token_refresh_lambda_arn" {
  description = "Token refresh Lambda ARN"
  value       = aws_lambda_function.token_refresh.arn
}

output "token_refresh_lambda_invoke_arn" {
  description = "Token refresh Lambda invoke ARN"
  value       = aws_lambda_function.token_refresh.invoke_arn
}
