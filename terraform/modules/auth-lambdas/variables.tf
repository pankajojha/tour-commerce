variable "user_pool_id" {
  description = "Cognito User Pool ID"
  type        = string
}

variable "user_pool_arn" {
  description = "Cognito User Pool ARN"
  type        = string
}

variable "web_client_id" {
  description = "Web client ID"
  type        = string
}

variable "database_endpoint" {
  description = "Database endpoint"
  type        = string
}

variable "database_name" {
  description = "Database name"
  type        = string
}

variable "database_secret_arn" {
  description = "Database credentials secret ARN"
  type        = string
}

variable "oauth_secret_arn" {
  description = "OAuth credentials secret ARN"
  type        = string
}

variable "kms_key_arn" {
  description = "KMS key ARN"
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs for Lambda"
  type        = list(string)
}

variable "lambda_security_group_id" {
  description = "Security group ID for Lambda functions"
  type        = string
}

variable "from_email" {
  description = "From email address for SES"
  type        = string
  default     = "noreply@exploreragent.com"
}

variable "app_url" {
  description = "Application URL for magic links"
  type        = string
  default     = "https://app.exploreragent.com"
}
