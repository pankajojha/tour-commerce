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
      Project   = "ExplorerAgentPlatform"
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

# Auth Module - Cognito User Pool
module "auth" {
  source = "./modules/auth"

  environment         = var.environment
  kms_key_id          = module.security.kms_key_id
  enable_social_login = var.enable_social_login
  
  # Callback URLs
  web_callback_urls    = var.web_callback_urls
  web_logout_urls      = var.web_logout_urls
  mobile_callback_urls = var.mobile_callback_urls
  mobile_logout_urls   = var.mobile_logout_urls
  
  # OAuth credentials (optional - set in terraform.tfvars)
  google_client_id     = var.google_client_id
  google_client_secret = var.google_client_secret
  facebook_app_id      = var.facebook_app_id
  facebook_app_secret  = var.facebook_app_secret
  apple_client_id      = var.apple_client_id
  apple_team_id        = var.apple_team_id
  apple_key_id         = var.apple_key_id
  apple_private_key    = var.apple_private_key
  
  # Lambda ARNs for custom auth triggers
  define_auth_challenge_lambda_arn          = ""
  create_auth_challenge_lambda_arn          = ""
  verify_auth_challenge_lambda_arn          = ""
  post_confirmation_lambda_arn              = ""
  pre_token_generation_lambda_arn           = ""
}

# Auth Lambda Functions
module "auth_lambdas" {
  source = "./modules/auth-lambdas"

  user_pool_id              = module.auth.user_pool_id
  user_pool_arn             = module.auth.user_pool_arn
  web_client_id             = module.auth.web_client_id
  database_endpoint         = module.database.cluster_endpoint
  database_name             = module.database.database_name
  database_secret_arn       = module.database.database_credentials_secret_arn
  oauth_secret_arn          = module.auth.oauth_credentials_secret_arn
  kms_key_arn               = module.security.kms_key_arn
  private_subnet_ids        = module.network.private_subnet_ids
  lambda_security_group_id  = module.network.lambda_security_group_id
  from_email                = var.from_email
  app_url                   = var.app_url
  cognito_domain            = format("https://%s.auth.%s.amazoncognito.com", module.auth.user_pool_domain, var.aws_region)
  oauth_callback_url        = var.oauth_callback_url
  frontend_auth_callback_url = var.frontend_auth_callback_url
  frontend_auth_cancel_url   = var.frontend_auth_cancel_url
}

# API Gateway
module "api_gateway" {
  source = "./modules/api-gateway"

  environment = var.environment
  aws_region  = var.aws_region

  user_pool_arn = module.auth.user_pool_arn

  auth_handler_lambda_invoke_arn = module.auth_lambdas.auth_handler_lambda_invoke_arn
  auth_handler_lambda_arn        = module.auth_lambdas.auth_handler_lambda_arn
}
