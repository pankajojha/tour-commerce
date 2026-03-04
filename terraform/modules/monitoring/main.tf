# CloudWatch Log Groups
resource "aws_cloudwatch_log_group" "api_gateway" {
  name              = "/aws/apigateway/explorer-agent"
  retention_in_days = 30

  tags = {
    Name = "explorer-agent-api-gateway-logs"
  }
}

resource "aws_cloudwatch_log_group" "lambda" {
  name              = "/aws/lambda/explorer-agent"
  retention_in_days = 7

  tags = {
    Name = "explorer-agent-lambda-logs"
  }
}

resource "aws_cloudwatch_log_group" "ecs" {
  name              = "/aws/ecs/explorer-agent"
  retention_in_days = 7

  tags = {
    Name = "explorer-agent-ecs-logs"
  }
}

# CloudWatch Alarms for Database
resource "aws_cloudwatch_metric_alarm" "database_cpu" {
  alarm_name          = "explorer-agent-database-high-cpu"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "CPUUtilization"
  namespace           = "AWS/RDS"
  period              = 300
  statistic           = "Average"
  threshold           = 80
  alarm_description   = "Database CPU utilization is too high"
  treat_missing_data  = "notBreaching"

  tags = {
    Name = "explorer-agent-database-cpu-alarm"
  }
}

resource "aws_cloudwatch_metric_alarm" "database_connections" {
  alarm_name          = "explorer-agent-database-high-connections"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "DatabaseConnections"
  namespace           = "AWS/RDS"
  period              = 300
  statistic           = "Average"
  threshold           = 80
  alarm_description   = "Database connections are too high"
  treat_missing_data  = "notBreaching"

  tags = {
    Name = "explorer-agent-database-connections-alarm"
  }
}

# CloudWatch Alarms for Cache
resource "aws_cloudwatch_metric_alarm" "cache_cpu" {
  alarm_name          = "explorer-agent-cache-high-cpu"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ElastiCache"
  period              = 300
  statistic           = "Average"
  threshold           = 75
  alarm_description   = "Cache CPU utilization is too high"
  treat_missing_data  = "notBreaching"

  tags = {
    Name = "explorer-agent-cache-cpu-alarm"
  }
}

resource "aws_cloudwatch_metric_alarm" "cache_memory" {
  alarm_name          = "explorer-agent-cache-high-memory"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "DatabaseMemoryUsagePercentage"
  namespace           = "AWS/ElastiCache"
  period              = 300
  statistic           = "Average"
  threshold           = 90
  alarm_description   = "Cache memory usage is too high"
  treat_missing_data  = "notBreaching"

  tags = {
    Name = "explorer-agent-cache-memory-alarm"
  }
}
