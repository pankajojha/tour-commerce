# KMS Key
resource "aws_kms_key" "main" {
  description             = "KMS key for tour agent platform encryption"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = {
    Name = "tour-agent-kms-key"
  }
}

resource "aws_kms_alias" "main" {
  name          = "alias/tour-agent-platform"
  target_key_id = aws_kms_key.main.key_id
}

# Stripe API Key Secret
resource "aws_secretsmanager_secret" "stripe_api_key" {
  name        = "tour-agent/stripe/api-key"
  description = "Stripe API key for payment processing"
  kms_key_id  = aws_kms_key.main.id
}

resource "aws_secretsmanager_secret_version" "stripe_api_key" {
  secret_id     = aws_secretsmanager_secret.stripe_api_key.id
  secret_string = "placeholder-update-after-deployment"
}

# Redis Auth Token Secret
resource "aws_secretsmanager_secret" "redis_auth_token" {
  name        = "tour-agent/redis/auth-token"
  description = "ElastiCache Redis authentication token"
  kms_key_id  = aws_kms_key.main.id
}

resource "aws_secretsmanager_secret_version" "redis_auth_token" {
  secret_id = aws_secretsmanager_secret.redis_auth_token.id
  secret_string = jsonencode({
    token = random_password.redis_auth_token.result
  })
}

resource "random_password" "redis_auth_token" {
  length  = 32
  special = false
}
