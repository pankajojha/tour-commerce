terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region  = var.aws_region
  profile = "AdministratorAccess-727974078184"

  default_tags {
    tags = {
      Project   = "TourAgentPlatform"
      ManagedBy = "Terraform"
    }
  }
}


# Network Module
module "network" {
  source = "./modules/network"

  vpc_cidr           = var.vpc_cidr
  availability_zones = var.availability_zones
  aws_region         = var.aws_region
}

# Security Module
module "security" {
  source = "./modules/security"
}

# Database Module
module "database" {
  source = "./modules/database"

  vpc_id                      = module.network.vpc_id
  isolated_subnet_ids         = module.network.isolated_subnet_ids
  database_security_group_id  = module.network.database_security_group_id
  database_name               = var.database_name
  aurora_min_capacity         = var.aurora_min_capacity
  aurora_max_capacity         = var.aurora_max_capacity
}

# Cache Module
module "cache" {
  source = "./modules/cache"

  vpc_id                       = module.network.vpc_id
  private_subnet_ids           = module.network.private_subnet_ids
  cache_security_group_id      = module.network.cache_security_group_id
  kms_key_arn                  = module.security.kms_key_arn
  redis_auth_token_secret_arn  = module.security.redis_auth_token_secret_arn
}

# Storage Module
module "storage" {
  source = "./modules/storage"

  kms_key_arn = module.security.kms_key_arn
}

# Monitoring Module
module "monitoring" {
  source = "./modules/monitoring"
}
