variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs for cache"
  type        = list(string)
}

variable "cache_security_group_id" {
  description = "Cache security group ID"
  type        = string
}

variable "kms_key_arn" {
  description = "KMS key ARN for encryption"
  type        = string
}

variable "redis_auth_token_secret_arn" {
  description = "Redis auth token secret ARN"
  type        = string
}
