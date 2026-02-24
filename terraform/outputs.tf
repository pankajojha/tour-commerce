output "vpc_id" {
  description = "VPC ID"
  value       = module.network.vpc_id
}

output "database_endpoint" {
  description = "Aurora cluster endpoint"
  value       = module.database.cluster_endpoint
}

output "database_read_endpoint" {
  description = "Aurora cluster read endpoint"
  value       = module.database.cluster_read_endpoint
}

output "kms_key_id" {
  description = "KMS key ID"
  value       = module.security.kms_key_id
}

output "redis_endpoint" {
  description = "Redis cluster endpoint"
  value       = module.cache.redis_endpoint
}

output "cloudfront_distribution_domain" {
  description = "CloudFront distribution domain"
  value       = module.storage.cloudfront_domain
}
