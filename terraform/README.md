# Tour Agent Platform - Terraform Infrastructure

This directory contains Terraform configuration for deploying the Tour Agent Platform infrastructure on AWS.

## Architecture

The infrastructure is organized into modular components:

- **Network Module**: VPC, subnets, NAT gateway, security groups, VPC endpoints
- **Security Module**: KMS keys, Secrets Manager secrets
- **Database Module**: Aurora Serverless v2 PostgreSQL cluster
- **Cache Module**: ElastiCache Redis cluster
- **Storage Module**: S3 buckets and CloudFront distribution
- **Monitoring Module**: CloudWatch log groups and alarms

## Prerequisites

- Terraform >= 1.0
- AWS CLI configured with appropriate credentials
- AWS account with necessary permissions

## Usage

### Initialize Terraform

```bash
cd terraform
terraform init
```

### Plan Infrastructure

```bash
terraform plan
```

### Deploy Infrastructure

Deploy all modules:

```bash
terraform apply
```

Deploy specific modules:

```bash
terraform apply -target=module.network
terraform apply -target=module.security
terraform apply -target=module.database
```

### Destroy Infrastructure

```bash
terraform destroy
```

## Configuration

### Variables

Key variables can be configured in `terraform.tfvars`:

```hcl
aws_region         = "us-east-1"
environment        = "production"
vpc_cidr           = "10.0.0.0/16"
availability_zones = ["us-east-1a", "us-east-1b", "us-east-1c"]
database_name      = "touragent"
aurora_min_capacity = 0.5
aurora_max_capacity = 128
```

### Outputs

After deployment, Terraform outputs important resource identifiers:

- VPC ID
- Database endpoints
- KMS key ID
- Redis endpoint
- CloudFront distribution domain

## Module Dependencies

```
network (independent)
security (independent)
database -> network, security
cache -> network, security
storage -> security
monitoring (independent)
```

## Cost Optimization

- Single NAT Gateway (instead of one per AZ)
- Aurora Serverless v2 with auto-scaling (0.5-128 ACU)
- ElastiCache t4g.micro instances
- CloudFront PriceClass_100 (US, Canada, Europe)

## Security Features

- VPC with isolated subnets for database
- Security groups with least privilege
- KMS encryption for data at rest
- Secrets Manager for credentials
- CloudFront with HTTPS redirect
- S3 bucket encryption and versioning

## Monitoring

- CloudWatch log groups for API Gateway, Lambda, ECS
- CloudWatch alarms for database CPU and connections
- CloudWatch alarms for cache CPU and memory
- 7-30 day log retention

## Comparison with CDK

This Terraform configuration is equivalent to the AWS CDK infrastructure in the `infrastructure/` directory. Key differences:

- **State Management**: Terraform uses state files; CDK uses CloudFormation
- **Language**: Terraform uses HCL; CDK uses TypeScript
- **Modularity**: Both support modular architecture
- **Deployment**: Terraform uses `terraform apply`; CDK uses `cdk deploy`

Choose based on your team's preferences and existing tooling.
