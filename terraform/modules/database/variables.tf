variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "isolated_subnet_ids" {
  description = "Isolated subnet IDs for database"
  type        = list(string)
}

variable "database_security_group_id" {
  description = "Database security group ID"
  type        = string
}

variable "database_name" {
  description = "Default database name"
  type        = string
}

variable "aurora_min_capacity" {
  description = "Aurora Serverless v2 minimum capacity"
  type        = number
}

variable "aurora_max_capacity" {
  description = "Aurora Serverless v2 maximum capacity"
  type        = number
}
