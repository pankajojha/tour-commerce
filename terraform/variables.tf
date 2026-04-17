variable "aws_region" {
  description = "AWS region for resources"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "availability_zones" {
  description = "Availability zones"
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b", "us-east-1c"]
}

variable "database_name" {
  description = "Default database name"
  type        = string
  default     = "exploreragent"
}

variable "aurora_min_capacity" {
  description = "Aurora Serverless v2 minimum capacity (ACU)"
  type        = number
  default     = 0.5
}

variable "aurora_max_capacity" {
  description = "Aurora Serverless v2 maximum capacity (ACU)"
  type        = number
  default     = 128
}

# Authentication Variables
variable "enable_social_login" {
  description = "Enable social login providers"
  type        = bool
  default     = false
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

variable "from_email" {
  description = "From email address for SES"
  type        = string
  default     = "noreply@exploreragent.com"
}

variable "app_url" {
  description = "Application URL for magic links"
  type        = string
  default     = "http://localhost:3000"
}

variable "oauth_callback_url" {
  description = "Backend OAuth callback URL registered in Cognito app client"
  type        = string
  default     = "http://localhost:8080/auth/oauth/callback"
}

variable "frontend_auth_callback_url" {
  description = "Frontend URL for successful social login return"
  type        = string
  default     = "http://localhost:5173/auth/callback"
}

variable "frontend_auth_cancel_url" {
  description = "Frontend URL for cancelled/failed social login return"
  type        = string
  default     = "http://localhost:5173/auth/cancel"
}

# OAuth Provider Credentials (optional)
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
