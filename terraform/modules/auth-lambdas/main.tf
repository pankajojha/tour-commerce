# IAM Role for auth Lambda
resource "aws_iam_role" "auth_lambda_role" {
  name = "explorer-agent-auth-lambda-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_basic" {
  role       = aws_iam_role.auth_lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy_attachment" "lambda_vpc" {
  role       = aws_iam_role.auth_lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

resource "aws_iam_role_policy_attachment" "lambda_xray" {
  role       = aws_iam_role.auth_lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/AWSXRayDaemonWriteAccess"
}

resource "aws_iam_role_policy" "auth_lambda_policy" {
  name = "explorer-agent-auth-lambda-policy"
  role = aws_iam_role.auth_lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["cognito-idp:*"]
        Resource = var.user_pool_arn
      },
      {
        Effect   = "Allow"
        Action   = ["ses:SendEmail", "ses:SendRawEmail"]
        Resource = "*"
      },
      {
        Effect   = "Allow"
        Action   = ["secretsmanager:GetSecretValue"]
        Resource = compact([var.database_secret_arn, var.oauth_secret_arn])
      },
      {
        Effect   = "Allow"
        Action   = ["kms:Decrypt"]
        Resource = var.kms_key_arn
      }
    ]
  })
}

# =============================================================================
# Single auth handler Lambda — serves all /auth/* API endpoints
# =============================================================================

resource "aws_lambda_function" "auth_handler" {
  filename         = "${path.root}/../services/auth/dist/handler.zip"
  function_name    = "explorer-agent-auth-handler"
  role             = aws_iam_role.auth_lambda_role.arn
  handler          = "bootstrap"
  source_code_hash = filebase64sha256("${path.root}/../services/auth/dist/handler.zip")
  runtime          = "provided.al2"
  timeout          = 30
  memory_size      = 256

  environment {
    variables = {
      USER_POOL_ID = var.user_pool_id
      CLIENT_ID    = var.web_client_id
      DB_HOST      = var.database_endpoint
      DB_NAME      = var.database_name
      DB_SECRET_ARN = var.database_secret_arn
      FROM_EMAIL   = var.from_email
      APP_URL      = var.app_url
    }
  }

  vpc_config {
    subnet_ids         = var.private_subnet_ids
    security_group_ids = [var.lambda_security_group_id]
  }

  tracing_config {
    mode = "Active"
  }
}

resource "aws_cloudwatch_log_group" "auth_handler" {
  name              = "/aws/lambda/${aws_lambda_function.auth_handler.function_name}"
  retention_in_days = 30
}

# =============================================================================
# Pre-Signup Trigger Lambda (Cognito trigger — separate event type)
# =============================================================================

resource "aws_lambda_function" "pre_signup" {
  filename         = "${path.root}/../services/auth/dist/pre-signup.zip"
  function_name    = "explorer-agent-auth-pre-signup"
  role             = aws_iam_role.auth_lambda_role.arn
  handler          = "bootstrap"
  source_code_hash = filebase64sha256("${path.root}/../services/auth/dist/pre-signup.zip")
  runtime          = "provided.al2"
  timeout          = 10
  memory_size      = 128

  tracing_config {
    mode = "Active"
  }
}

# OAuth Start Lambda
resource "aws_lambda_function" "oauth_start" {
  filename         = "${path.root}/../services/auth/dist/oauth-start.zip"
  function_name    = "explorer-agent-auth-oauth-start"
  role             = aws_iam_role.auth_lambda_role.arn
  handler          = "bootstrap"
  source_code_hash = filebase64sha256("${path.root}/../services/auth/dist/oauth-start.zip")
  runtime          = "provided.al2"
  timeout          = 30
  memory_size      = 256

  environment {
    variables = {
      CLIENT_ID          = var.web_client_id
      COGNITO_DOMAIN     = var.cognito_domain
      OAUTH_CALLBACK_URL = var.oauth_callback_url
      OAUTH_SCOPE        = "openid email profile"
    }
  }

  vpc_config {
    subnet_ids         = var.private_subnet_ids
    security_group_ids = [var.lambda_security_group_id]
  }

  tracing_config {
    mode = "Active"
  }
}

# OAuth Callback Lambda
resource "aws_lambda_function" "oauth_callback" {
  filename         = "${path.root}/../services/auth/dist/oauth-callback.zip"
  function_name    = "explorer-agent-auth-oauth-callback"
  role             = aws_iam_role.auth_lambda_role.arn
  handler          = "bootstrap"
  source_code_hash = filebase64sha256("${path.root}/../services/auth/dist/oauth-callback.zip")
  runtime          = "provided.al2"
  timeout          = 30
  memory_size      = 256

  environment {
    variables = {
      FRONTEND_AUTH_CALLBACK_URL = var.frontend_auth_callback_url
      FRONTEND_AUTH_CANCEL_URL   = var.frontend_auth_cancel_url
    }
  }

  vpc_config {
    subnet_ids         = var.private_subnet_ids
    security_group_ids = [var.lambda_security_group_id]
  }

  tracing_config {
    mode = "Active"
  }
}

resource "aws_cloudwatch_log_group" "pre_signup" {
  name              = "/aws/lambda/${aws_lambda_function.pre_signup.function_name}"
  retention_in_days = 30
}

resource "aws_cloudwatch_log_group" "oauth_start" {
  name              = "/aws/lambda/${aws_lambda_function.oauth_start.function_name}"
  retention_in_days = 30
}

resource "aws_cloudwatch_log_group" "oauth_callback" {
  name              = "/aws/lambda/${aws_lambda_function.oauth_callback.function_name}"
  retention_in_days = 30
}
