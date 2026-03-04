# IAM Role for Lambda functions
resource "aws_iam_role" "auth_lambda_role" {
  name = "explorer-agent-auth-lambda-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

# Attach basic Lambda execution policy
resource "aws_iam_role_policy_attachment" "lambda_basic" {
  role       = aws_iam_role.auth_lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

# Attach VPC execution policy
resource "aws_iam_role_policy_attachment" "lambda_vpc" {
  role       = aws_iam_role.auth_lambda_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

# Custom policy for Cognito, SES, Secrets Manager, and RDS
resource "aws_iam_role_policy" "auth_lambda_policy" {
  name = "explorer-agent-auth-lambda-policy"
  role = aws_iam_role.auth_lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "cognito-idp:*"
        ]
        Resource = var.user_pool_arn
      },
      {
        Effect = "Allow"
        Action = [
          "ses:SendEmail",
          "ses:SendRawEmail"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue"
        ]
        Resource = [
          var.database_secret_arn,
          var.oauth_secret_arn
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "kms:Decrypt"
        ]
        Resource = var.kms_key_arn
      }
    ]
  })
}

# Signup Lambda
resource "aws_lambda_function" "signup" {
  filename         = "${path.root}/../services/auth/dist/signup.zip"
  function_name    = "explorer-agent-auth-signup"
  role            = aws_iam_role.auth_lambda_role.arn
  handler         = "bootstrap"
  source_code_hash = filebase64sha256("${path.root}/../services/auth/dist/signup.zip")
  runtime         = "provided.al2"
  timeout         = 30
  memory_size     = 256

  environment {
    variables = {
      USER_POOL_ID = var.user_pool_id
      CLIENT_ID    = var.web_client_id
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

# Login Lambda
resource "aws_lambda_function" "login" {
  filename         = "${path.root}/../services/auth/dist/login.zip"
  function_name    = "explorer-agent-auth-login"
  role            = aws_iam_role.auth_lambda_role.arn
  handler         = "bootstrap"
  source_code_hash = filebase64sha256("${path.root}/../services/auth/dist/login.zip")
  runtime         = "provided.al2"
  timeout         = 30
  memory_size     = 256

  environment {
    variables = {
      CLIENT_ID = var.web_client_id
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

# Passwordless Request Lambda
resource "aws_lambda_function" "passwordless_request" {
  filename         = "${path.root}/../services/auth/dist/passwordless-request.zip"
  function_name    = "explorer-agent-auth-passwordless-request"
  role            = aws_iam_role.auth_lambda_role.arn
  handler         = "bootstrap"
  source_code_hash = filebase64sha256("${path.root}/../services/auth/dist/passwordless-request.zip")
  runtime         = "provided.al2"
  timeout         = 30
  memory_size     = 256

  environment {
    variables = {
      DB_HOST             = var.database_endpoint
      DB_NAME             = var.database_name
      DB_SECRET_ARN       = var.database_secret_arn
      FROM_EMAIL          = var.from_email
      APP_URL             = var.app_url
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

# Passwordless Verify Lambda
resource "aws_lambda_function" "passwordless_verify" {
  filename         = "${path.root}/../services/auth/dist/passwordless-verify.zip"
  function_name    = "explorer-agent-auth-passwordless-verify"
  role            = aws_iam_role.auth_lambda_role.arn
  handler         = "bootstrap"
  source_code_hash = filebase64sha256("${path.root}/../services/auth/dist/passwordless-verify.zip")
  runtime         = "provided.al2"
  timeout         = 30
  memory_size     = 256

  environment {
    variables = {
      DB_HOST       = var.database_endpoint
      DB_NAME       = var.database_name
      DB_SECRET_ARN = var.database_secret_arn
      CLIENT_ID     = var.web_client_id
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

# Token Refresh Lambda
resource "aws_lambda_function" "token_refresh" {
  filename         = "${path.root}/../services/auth/dist/token-refresh.zip"
  function_name    = "explorer-agent-auth-token-refresh"
  role            = aws_iam_role.auth_lambda_role.arn
  handler         = "bootstrap"
  source_code_hash = filebase64sha256("${path.root}/../services/auth/dist/token-refresh.zip")
  runtime         = "provided.al2"
  timeout         = 30
  memory_size     = 256

  environment {
    variables = {
      CLIENT_ID = var.web_client_id
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

# CloudWatch Log Groups
resource "aws_cloudwatch_log_group" "signup" {
  name              = "/aws/lambda/${aws_lambda_function.signup.function_name}"
  retention_in_days = 30
}

resource "aws_cloudwatch_log_group" "login" {
  name              = "/aws/lambda/${aws_lambda_function.login.function_name}"
  retention_in_days = 30
}

resource "aws_cloudwatch_log_group" "passwordless_request" {
  name              = "/aws/lambda/${aws_lambda_function.passwordless_request.function_name}"
  retention_in_days = 30
}

resource "aws_cloudwatch_log_group" "passwordless_verify" {
  name              = "/aws/lambda/${aws_lambda_function.passwordless_verify.function_name}"
  retention_in_days = 30
}

resource "aws_cloudwatch_log_group" "token_refresh" {
  name              = "/aws/lambda/${aws_lambda_function.token_refresh.function_name}"
  retention_in_days = 30
}
