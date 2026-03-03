# Database Credentials Secret
resource "aws_secretsmanager_secret" "database_credentials" {
  name        = "tour-agent/database/credentials"
  description = "Aurora Serverless v2 PostgreSQL credentials"
}

resource "aws_secretsmanager_secret_version" "database_credentials" {
  secret_id = aws_secretsmanager_secret.database_credentials.id
  secret_string = jsonencode({
    username = "postgres"
    password = random_password.database_password.result
  })
}

resource "random_password" "database_password" {
  length  = 32
  special = false
}

# DB Subnet Group
resource "aws_db_subnet_group" "main" {
  name       = "tour-agent-db-subnet-group"
  subnet_ids = var.isolated_subnet_ids

  tags = {
    Name = "tour-agent-db-subnet-group"
  }
}

# RDS Cluster Parameter Group
resource "aws_rds_cluster_parameter_group" "main" {
  name        = "tour-agent-aurora-postgres15"
  family      = "aurora-postgresql15"
  description = "Aurora PostgreSQL 15 cluster parameter group"

  parameter {
    name  = "log_statement"
    value = "all"
  }

  parameter {
    name  = "log_min_duration_statement"
    value = "1000"
  }
}

# Aurora Serverless v2 Cluster
resource "aws_rds_cluster" "main" {
  cluster_identifier      = "tour-agent-database"
  engine                  = "aurora-postgresql"
  engine_version          = "15.8"
  database_name           = var.database_name
  master_username         = jsondecode(aws_secretsmanager_secret_version.database_credentials.secret_string)["username"]
  master_password         = jsondecode(aws_secretsmanager_secret_version.database_credentials.secret_string)["password"]
  db_subnet_group_name    = aws_db_subnet_group.main.name
  vpc_security_group_ids  = [var.database_security_group_id]
  storage_encrypted       = true
  backup_retention_period = 7
  preferred_backup_window = "03:00-04:00"
  preferred_maintenance_window = "sun:04:00-sun:05:00"
  enabled_cloudwatch_logs_exports = ["postgresql"]
  deletion_protection     = true
  skip_final_snapshot     = false
  final_snapshot_identifier = "tour-agent-database-final-snapshot"
  
  db_cluster_parameter_group_name = aws_rds_cluster_parameter_group.main.name

  serverlessv2_scaling_configuration {
    min_capacity = var.aurora_min_capacity
    max_capacity = var.aurora_max_capacity
  }

  tags = {
    Name = "tour-agent-database"
  }
}

# Aurora Serverless v2 Writer Instance
resource "aws_rds_cluster_instance" "writer" {
  identifier              = "tour-agent-database-writer"
  cluster_identifier      = aws_rds_cluster.main.id
  instance_class          = "db.serverless"
  engine                  = aws_rds_cluster.main.engine
  engine_version          = aws_rds_cluster.main.engine_version
  auto_minor_version_upgrade = true
  publicly_accessible     = false

  tags = {
    Name = "tour-agent-database-writer"
  }
}

# Aurora Serverless v2 Reader Instance
resource "aws_rds_cluster_instance" "reader" {
  identifier              = "tour-agent-database-reader1"
  cluster_identifier      = aws_rds_cluster.main.id
  instance_class          = "db.serverless"
  engine                  = aws_rds_cluster.main.engine
  engine_version          = aws_rds_cluster.main.engine_version
  auto_minor_version_upgrade = true
  publicly_accessible     = false

  tags = {
    Name = "tour-agent-database-reader1"
  }
}
