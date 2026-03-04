# Cognito User Pool
resource "aws_cognito_user_pool" "main" {
  name = "explorer-agent-user-pool"

  # Username configuration
  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]

  # Password policy
  password_policy {
    minimum_length                   = 8
    require_lowercase                = true
    require_uppercase                = true
    require_numbers                  = true
    require_symbols                  = true
    temporary_password_validity_days = 7
  }

  # Email configuration
  email_configuration {
    email_sending_account = "COGNITO_DEFAULT"
  }

  # Account recovery
  account_recovery_setting {
    recovery_mechanism {
      name     = "verified_email"
      priority = 1
    }
  }

  # User attributes
  schema {
    name                = "email"
    attribute_data_type = "String"
    required            = true
    mutable             = false

    string_attribute_constraints {
      min_length = 5
      max_length = 255
    }
  }

  schema {
    name                = "phone_number"
    attribute_data_type = "String"
    required            = false
    mutable             = true

    string_attribute_constraints {
      min_length = 10
      max_length = 20
    }
  }

  schema {
    name                = "name"
    attribute_data_type = "String"
    required            = false
    mutable             = true

    string_attribute_constraints {
      min_length = 1
      max_length = 255
    }
  }

  # Custom auth flow for passwordless
  lambda_config {
    define_auth_challenge          = var.define_auth_challenge_lambda_arn
    create_auth_challenge          = var.create_auth_challenge_lambda_arn
    verify_auth_challenge_response = var.verify_auth_challenge_lambda_arn
    pre_sign_up                    = var.pre_signup_lambda_arn
    post_confirmation              = var.post_confirmation_lambda_arn
    pre_token_generation           = var.pre_token_generation_lambda_arn
  }

  # MFA configuration (optional)
  mfa_configuration = "OPTIONAL"

  software_token_mfa_configuration {
    enabled = true
  }

  # Device tracking
  device_configuration {
    challenge_required_on_new_device      = false
    device_only_remembered_on_user_prompt = true
  }

  # User pool add-ons
  user_pool_add_ons {
    advanced_security_mode = "ENFORCED"
  }

  tags = {
    Name = "explorer-agent-user-pool"
  }
}

# Cognito User Pool Domain
resource "aws_cognito_user_pool_domain" "main" {
  domain       = "explorer-agent-${var.environment}"
  user_pool_id = aws_cognito_user_pool.main.id
}

# Cognito User Pool Client - Web
resource "aws_cognito_user_pool_client" "web" {
  name         = "explorer-agent-web-client"
  user_pool_id = aws_cognito_user_pool.main.id

  generate_secret = false

  # OAuth configuration
  allowed_oauth_flows_user_pool_client = true
  allowed_oauth_flows                  = ["code", "implicit"]
  allowed_oauth_scopes                 = ["email", "openid", "profile"]
  callback_urls                        = var.web_callback_urls
  logout_urls                          = var.web_logout_urls

  # Supported identity providers
  supported_identity_providers = concat(
    ["COGNITO"],
    var.enable_social_login ? [
      "Google",
      "Facebook",
      "SignInWithApple"
    ] : []
  )

  # Token validity
  id_token_validity      = 60  # 60 minutes
  access_token_validity  = 60  # 60 minutes
  refresh_token_validity = 30  # 30 days

  token_validity_units {
    id_token      = "minutes"
    access_token  = "minutes"
    refresh_token = "days"
  }

  # Auth flows
  explicit_auth_flows = [
    "ALLOW_USER_PASSWORD_AUTH",
    "ALLOW_REFRESH_TOKEN_AUTH",
    "ALLOW_CUSTOM_AUTH",
    "ALLOW_USER_SRP_AUTH"
  ]

  # Prevent user existence errors
  prevent_user_existence_errors = "ENABLED"

  # Read/write attributes
  read_attributes = [
    "email",
    "email_verified",
    "name",
    "phone_number",
    "phone_number_verified"
  ]

  write_attributes = [
    "email",
    "name",
    "phone_number"
  ]
}

# Cognito User Pool Client - Mobile
resource "aws_cognito_user_pool_client" "mobile" {
  name         = "explorer-agent-mobile-client"
  user_pool_id = aws_cognito_user_pool.main.id

  generate_secret = false

  # OAuth configuration
  allowed_oauth_flows_user_pool_client = true
  allowed_oauth_flows                  = ["code"]
  allowed_oauth_scopes                 = ["email", "openid", "profile"]
  callback_urls                        = var.mobile_callback_urls
  logout_urls                          = var.mobile_logout_urls

  # Supported identity providers
  supported_identity_providers = concat(
    ["COGNITO"],
    var.enable_social_login ? [
      "Google",
      "Facebook",
      "SignInWithApple"
    ] : []
  )

  # Token validity
  id_token_validity      = 60  # 60 minutes
  access_token_validity  = 60  # 60 minutes
  refresh_token_validity = 30  # 30 days

  token_validity_units {
    id_token      = "minutes"
    access_token  = "minutes"
    refresh_token = "days"
  }

  # Auth flows
  explicit_auth_flows = [
    "ALLOW_USER_PASSWORD_AUTH",
    "ALLOW_REFRESH_TOKEN_AUTH",
    "ALLOW_CUSTOM_AUTH",
    "ALLOW_USER_SRP_AUTH"
  ]

  # Prevent user existence errors
  prevent_user_existence_errors = "ENABLED"

  # Read/write attributes
  read_attributes = [
    "email",
    "email_verified",
    "name",
    "phone_number",
    "phone_number_verified"
  ]

  write_attributes = [
    "email",
    "name",
    "phone_number"
  ]
}

# Google Identity Provider (conditional)
resource "aws_cognito_identity_provider" "google" {
  count = var.enable_social_login && var.google_client_id != "" ? 1 : 0

  user_pool_id  = aws_cognito_user_pool.main.id
  provider_name = "Google"
  provider_type = "Google"

  provider_details = {
    authorize_scopes = "email profile openid"
    client_id        = var.google_client_id
    client_secret    = var.google_client_secret
  }

  attribute_mapping = {
    email    = "email"
    name     = "name"
    username = "sub"
  }
}

# Facebook Identity Provider (conditional)
resource "aws_cognito_identity_provider" "facebook" {
  count = var.enable_social_login && var.facebook_app_id != "" ? 1 : 0

  user_pool_id  = aws_cognito_user_pool.main.id
  provider_name = "Facebook"
  provider_type = "Facebook"

  provider_details = {
    authorize_scopes = "email,public_profile"
    client_id        = var.facebook_app_id
    client_secret    = var.facebook_app_secret
  }

  attribute_mapping = {
    email    = "email"
    name     = "name"
    username = "id"
  }
}

# Apple Identity Provider (conditional)
resource "aws_cognito_identity_provider" "apple" {
  count = var.enable_social_login && var.apple_client_id != "" ? 1 : 0

  user_pool_id  = aws_cognito_user_pool.main.id
  provider_name = "SignInWithApple"
  provider_type = "SignInWithApple"

  provider_details = {
    authorize_scopes = "email name"
    client_id        = var.apple_client_id
    team_id          = var.apple_team_id
    key_id           = var.apple_key_id
    private_key      = var.apple_private_key
  }

  attribute_mapping = {
    email    = "email"
    name     = "name"
    username = "sub"
  }
}

# Store OAuth secrets in Secrets Manager
resource "aws_secretsmanager_secret" "oauth_credentials" {
  count = var.enable_social_login ? 1 : 0

  name        = "explorer-agent/oauth/credentials"
  description = "OAuth provider credentials for social login"
  kms_key_id  = var.kms_key_id
}

resource "aws_secretsmanager_secret_version" "oauth_credentials" {
  count = var.enable_social_login ? 1 : 0

  secret_id = aws_secretsmanager_secret.oauth_credentials[0].id
  secret_string = jsonencode({
    google = {
      client_id     = var.google_client_id
      client_secret = var.google_client_secret
    }
    facebook = {
      app_id     = var.facebook_app_id
      app_secret = var.facebook_app_secret
    }
    apple = {
      client_id   = var.apple_client_id
      team_id     = var.apple_team_id
      key_id      = var.apple_key_id
      private_key = var.apple_private_key
    }
  })
}
