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

variable "auth_handler_lambda_invoke_arn" {
  description = "Auth handler Lambda invoke ARN (single Lambda for all /auth/* routes)"
  type        = string
}

variable "auth_handler_lambda_arn" {
  description = "Auth handler Lambda ARN"
  type        = string
}

variable "allowed_origins" {
  description = "Allowed CORS origins"
  type        = list(string)
  default     = ["http://localhost:3000"]
}
