variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "kms_key_id" {
  description = "KMS key ID for encryption"
  type        = string
}

variable "web_callback_urls" {
  description = "Callback URLs for web client"
  type        = list(string)
  default     = ["http://localhost:3000/auth/callback"]
}

variable "web_logout_urls" {
  description = "Logout URLs for web client"
  type        = list(string)
  default     = ["http://localhost:3000"]
}

variable "mobile_callback_urls" {
  description = "Callback URLs for mobile client"
  type        = list(string)
  default     = ["exploreragent://auth/callback"]
}

variable "mobile_logout_urls" {
  description = "Logout URLs for mobile client"
  type        = list(string)
  default     = ["exploreragent://"]
}

variable "enable_social_login" {
  description = "Enable social login providers"
  type        = bool
  default     = false
}

# Google OAuth
variable "google_client_id" {
  description = "Google OAuth client ID"
  type        = string
  default     = ""
  sensitive   = true
}

variable "google_client_secret" {
  description = "Google OAuth client secret"
  type        = string
  default     = ""
  sensitive   = true
}

# Facebook OAuth
variable "facebook_app_id" {
  description = "Facebook app ID"
  type        = string
  default     = ""
  sensitive   = true
}

variable "facebook_app_secret" {
  description = "Facebook app secret"
  type        = string
  default     = ""
  sensitive   = true
}

# Apple OAuth
variable "apple_client_id" {
  description = "Apple client ID (Service ID)"
  type        = string
  default     = ""
  sensitive   = true
}

variable "apple_team_id" {
  description = "Apple team ID"
  type        = string
  default     = ""
  sensitive   = true
}

variable "apple_key_id" {
  description = "Apple key ID"
  type        = string
  default     = ""
  sensitive   = true
}

variable "apple_private_key" {
  description = "Apple private key"
  type        = string
  default     = ""
  sensitive   = true
}

# Lambda ARNs for Cognito triggers
variable "define_auth_challenge_lambda_arn" {
  description = "ARN of Lambda function for define auth challenge"
  type        = string
  default     = ""
}

variable "create_auth_challenge_lambda_arn" {
  description = "ARN of Lambda function for create auth challenge"
  type        = string
  default     = ""
}

variable "verify_auth_challenge_lambda_arn" {
  description = "ARN of Lambda function for verify auth challenge"
  type        = string
  default     = ""
}

variable "pre_signup_lambda_arn" {
  description = "ARN of Lambda function for pre-signup trigger"
  type        = string
  default     = ""
}

variable "post_confirmation_lambda_arn" {
  description = "ARN of Lambda function for post-confirmation trigger"
  type        = string
  default     = ""
}

variable "pre_token_generation_lambda_arn" {
  description = "ARN of Lambda function for pre-token generation trigger"
  type        = string
  default     = ""
}
