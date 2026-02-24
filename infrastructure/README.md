# Tour Booking Platform - Infrastructure

This directory contains the AWS CDK infrastructure code for the tour excursion booking platform.

## Architecture Overview

The infrastructure is organized into the following stacks:

1. **NetworkStack** - VPC, subnets, NAT Gateway, security groups
2. **SecurityStack** - KMS keys, Secrets Manager secrets
3. **DatabaseStack** - Aurora Serverless v2 PostgreSQL cluster
4. **DatabaseSchemaStack** - Automated database schema deployment
5. **CacheStack** - ElastiCache Redis cluster
6. **StorageStack** - S3 buckets, CloudFront distribution
7. **MonitoringStack** - CloudWatch log groups, X-Ray tracing

## Prerequisites

- Node.js 20.x or later
- AWS CLI configured with appropriate credentials
- AWS CDK CLI installed (`npm install -g aws-cdk`)

## Installation

```bash
npm install
```

## Build

```bash
npm run build
```

## Deploy

### Deploy all stacks

```bash
npm run deploy
```

### Deploy specific stack

```bash
cdk deploy TourAgentNetworkStack
cdk deploy TourAgentSecurityStack
cdk deploy TourAgentDatabaseStack
cdk deploy TourAgentDatabaseSchemaStack
cdk deploy TourAgentCacheStack
cdk deploy TourAgentStorageStack
cdk deploy TourAgentMonitoringStack
```

### Recommended Deployment Order

For initial deployment, follow this order:

```bash
# Step 1: Deploy network infrastructure
cdk deploy TourAgentNetworkStack

# Step 2: Deploy security infrastructure
cdk deploy TourAgentSecurityStack

# Step 3: Deploy database cluster (creates empty database)
cdk deploy TourAgentDatabaseStack

# Step 4: Deploy database schema (creates tables, indexes, etc.)
cdk deploy TourAgentDatabaseSchemaStack

# Step 5: Deploy remaining stacks
cdk deploy TourAgentCacheStack
cdk deploy TourAgentStorageStack
cdk deploy TourAgentMonitoringStack
```

## Synthesize CloudFormation templates

```bash
npm run synth
```

## View differences

```bash
npm run diff
```

## Stack Dependencies

The stacks have the following dependencies:

- DatabaseStack depends on NetworkStack and SecurityStack
- DatabaseSchemaStack depends on DatabaseStack, NetworkStack, and SecurityStack
- CacheStack depends on NetworkStack and SecurityStack
- StorageStack depends on SecurityStack

**Important**: The DatabaseSchemaStack must be deployed AFTER the DatabaseStack to ensure the database cluster is available before schema deployment.

## Infrastructure Components

### Network Stack

- **VPC**: 10.0.0.0/16 CIDR with 3 availability zones
- **Subnets**: Public, Private (with NAT), and Isolated subnets
- **NAT Gateway**: Single NAT Gateway for cost optimization
- **Security Groups**: Lambda, Database, and Cache security groups
- **VPC Endpoints**: S3 and DynamoDB gateway endpoints

### Security Stack

- **KMS Key**: Customer-managed key with automatic rotation
- **Secrets**:
  - Database credentials (auto-generated)
  - Stripe API key (placeholder)
  - Redis auth token (auto-generated)

### Database Stack

- **Aurora Serverless v2 PostgreSQL**:
  - Engine version: 15.4
  - Auto-scaling: 0.5 - 128 ACU
  - 1 writer + 1 reader instance
  - Encryption at rest with KMS
  - Automated backups (7-day retention)
  - CloudWatch Logs integration
  - Deletion protection enabled

### Database Schema Stack

- **Automated Schema Deployment**:
  - Lambda function deploys SQL migrations automatically
  - Migration files stored in `database/migrations/` directory
  - Tracks applied migrations in `schema_migrations` table
  - Idempotent - safe to run multiple times
  - Supports versioned migrations (V001__, V002__, etc.)
  - Executes migrations in transaction for rollback safety
  - Custom Resource triggers deployment during CDK deploy

**Migration File Format**:
- Filename: `V{version}__{description}.sql` (e.g., `V001__initial_schema.sql`)
- Version: 3-digit number (001, 002, 003, etc.)
- Description: Snake_case description of the migration

**How It Works**:
1. Lambda function reads migration files from Lambda Layer
2. Checks `schema_migrations` table for applied migrations
3. Applies pending migrations in order
4. Records each migration with version, description, and checksum
5. Rolls back on error to maintain database consistency

**Modifying Schema**:
To add new migrations:
1. Create new SQL file: `infrastructure/database/migrations/V002__add_new_table.sql`
2. Deploy schema stack: `cdk deploy TourAgentDatabaseSchemaStack`
3. Lambda automatically applies new migration

### Cache Stack

- **ElastiCache Redis**:
  - Engine version: 7.0
  - Node type: cache.t4g.micro
  - 2 nodes (primary + replica)
  - Multi-AZ with automatic failover
  - Encryption at rest and in transit
  - Auth token authentication
  - Snapshot retention: 5 days

### Storage Stack

- **S3 Buckets**:
  - Vouchers bucket (KMS encryption, versioned)
  - Static assets bucket (S3-managed encryption)
  - CloudFront logs bucket
- **CloudFront Distribution**:
  - HTTPS only
  - Compression enabled
  - Optimized caching policy
  - Price class: US, Canada, Europe

### Monitoring Stack

- **CloudWatch Log Groups**:
  - API Gateway logs (30-day retention)
  - Lambda logs (30-day retention)
  - Database logs (30-day retention)
  - Redis logs (30-day retention)
  - X-Ray traces (7-day retention)
  - EventBridge logs (30-day retention)
  - SQS logs (30-day retention)

## Post-Deployment Steps

1. **Update Stripe API Key**:
   ```bash
   aws secretsmanager update-secret \
     --secret-id tour-agent/stripe/api-key \
     --secret-string "your-stripe-api-key"
   ```

2. **Verify Database Schema**:
   The database schema is automatically deployed by the DatabaseSchemaStack. To verify:
   ```bash
   # Get database credentials
   aws secretsmanager get-secret-value \
     --secret-id tour-agent/database/credentials \
     --query SecretString --output text
   
   # Connect to database and check tables
   psql -h <endpoint> -U postgres -d touragent -c "\dt"
   
   # Check applied migrations
   psql -h <endpoint> -U postgres -d touragent -c "SELECT * FROM schema_migrations;"
   ```

3. **Configure Redis**:
   - Redis is ready to use with the auth token from Secrets Manager

## Cost Optimization

- Aurora Serverless v2 scales down to 0.5 ACU when idle
- Single NAT Gateway instead of one per AZ
- S3 lifecycle policies for old data
- CloudWatch log retention policies
- CloudFront price class limited to US, Canada, Europe

## Security Features

- All data encrypted at rest (KMS)
- All data encrypted in transit (TLS 1.2+)
- Private subnets for databases and cache
- Security groups with least privilege
- Secrets stored in Secrets Manager
- Deletion protection on critical resources

## Outputs

Each stack exports outputs that can be referenced by other stacks or application code:

- VPC ID and security group IDs
- Database endpoints and credentials secret ARN
- Redis endpoints and auth token secret ARN
- S3 bucket names and ARNs
- CloudFront distribution domain name
- Log group names

## Cleanup

To destroy all stacks:

```bash
cdk destroy --all
```

**Warning**: Some resources have `RETAIN` removal policy and will not be deleted:
- KMS keys
- S3 buckets (vouchers and static assets)
- Database snapshots

## Troubleshooting

### Stack deployment fails

Check CloudFormation events in AWS Console for detailed error messages.

### Cannot connect to database

Ensure Lambda functions are in the same VPC and have the correct security group.

### Redis connection timeout

Verify security group rules allow traffic on port 6379 from Lambda security group.

## Additional Resources

- [AWS CDK Documentation](https://docs.aws.amazon.com/cdk/)
- [Aurora Serverless v2 Documentation](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/aurora-serverless-v2.html)
- [ElastiCache Redis Documentation](https://docs.aws.amazon.com/AmazonElastiCache/latest/red-ug/)
