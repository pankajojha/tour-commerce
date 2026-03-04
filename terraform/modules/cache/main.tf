# ElastiCache Subnet Group
resource "aws_elasticache_subnet_group" "main" {
  name       = "explorer-agent-cache-subnet-group"
  subnet_ids = var.private_subnet_ids

  tags = {
    Name = "explorer-agent-cache-subnet-group"
  }
}

# ElastiCache Parameter Group
resource "aws_elasticache_parameter_group" "main" {
  name   = "explorer-agent-redis7"
  family = "redis7"

  parameter {
    name  = "maxmemory-policy"
    value = "allkeys-lru"
  }
}

# Get Redis auth token from Secrets Manager
data "aws_secretsmanager_secret_version" "redis_auth_token" {
  secret_id = var.redis_auth_token_secret_arn
}

# ElastiCache Redis Replication Group
resource "aws_elasticache_replication_group" "main" {
  replication_group_id       = "explorer-agent-redis"
  description                = "Redis cluster for inventory locking"
  engine                     = "redis"
  engine_version             = "7.0"
  node_type                  = "cache.t4g.micro"
  num_cache_clusters         = 2
  parameter_group_name       = aws_elasticache_parameter_group.main.name
  port                       = 6379
  subnet_group_name          = aws_elasticache_subnet_group.main.name
  security_group_ids         = [var.cache_security_group_id]
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  auth_token                 = jsondecode(data.aws_secretsmanager_secret_version.redis_auth_token.secret_string)["token"]
  kms_key_id                 = var.kms_key_arn
  automatic_failover_enabled = true
  multi_az_enabled           = true
  snapshot_retention_limit   = 5
  snapshot_window            = "03:00-05:00"
  maintenance_window         = "sun:05:00-sun:07:00"
  auto_minor_version_upgrade = true

  tags = {
    Name = "explorer-agent-redis"
  }
}
