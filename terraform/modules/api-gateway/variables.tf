variable "environment" {
  description = "Environment name"
  type        = string
}

variable "aws_region" {
  description = "AWS region"
  type        = string
}

variable "user_pool_arn" {
  description = "Cognito User Pool ARN for authorizer"
  type        = string
}

# Lambda invoke ARNs
variable "signup_lambda_invoke_arn" {
  description = "Signup Lambda invoke ARN"
  type        = string
}

variable "signup_lambda_arn" {
  description = "Signup Lambda ARN"
  type        = string
}

variable "login_lambda_invoke_arn" {
  description = "Login Lambda invoke ARN"
  type        = string
}

variable "login_lambda_arn" {
  description = "Login Lambda ARN"
  type        = string
}

variable "passwordless_request_lambda_invoke_arn" {
  description = "Passwordless request Lambda invoke ARN"
  type        = string
}

variable "passwordless_request_lambda_arn" {
  description = "Passwordless request Lambda ARN"
  type        = string
}

variable "passwordless_verify_lambda_invoke_arn" {
  description = "Passwordless verify Lambda invoke ARN"
  type        = string
}

variable "passwordless_verify_lambda_arn" {
  description = "Passwordless verify Lambda ARN"
  type        = string
}

variable "token_refresh_lambda_invoke_arn" {
  description = "Token refresh Lambda invoke ARN"
  type        = string
}

variable "token_refresh_lambda_arn" {
  description = "Token refresh Lambda ARN"
  type        = string
}

variable "allowed_origins" {
  description = "Allowed CORS origins"
  type        = list(string)
  default     = ["http://localhost:3000"]
}
