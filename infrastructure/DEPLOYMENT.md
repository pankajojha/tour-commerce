# Database Schema Deployment Guide

## Overview

The Tour Excursion Booking Platform uses an automated database schema deployment system that:
- Deploys SQL migrations automatically via AWS Lambda
- Tracks applied migrations to prevent re-running
- Supports versioned migrations for schema evolution
- Provides rollback safety through transactions

## Architecture

### Components

1. **DatabaseStack**: Creates the Aurora Serverless v2 PostgreSQL cluster
2. **DatabaseSchemaStack**: Deploys schema migrations to the database
3. **Schema Deployer Lambda**: Executes SQL migrations
4. **Custom Resource**: Triggers Lambda during CDK deployment

### Migration Flow

```
CDK Deploy → Custom Resource → Lambda Function → Read Migrations → Apply to Database
```

## Deployment Steps

### Initial Deployment

```bash
# 1. Deploy database cluster (creates empty "touragent" database)
cd infrastructure
cdk deploy ExplorerAgentDatabaseStack

# 2. Deploy schema (creates tables, indexes, functions, etc.)
cdk deploy ExplorerAgentDatabaseSchemaStack
```

### Subsequent Schema Changes

```bash
# 1. Create new migration file
# infrastructure/database/migrations/V002__add_feature.sql

# 2. Deploy schema stack
cdk deploy ExplorerAgentDatabaseSchemaStack

# Lambda automatically detects and applies new migration
```

## Migration File Format

### Naming Convention

```
V{version}__{description}.sql
```

- **Version**: 3-digit number (001, 002, 003, ...)
- **Description**: Snake_case description
- **Example**: `V001__initial_schema.sql`, `V002__add_reviews_table.sql`

### Example Migration

```sql
-- V002__add_notifications_table.sql

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_read ON notifications(read) WHERE read = FALSE;
```

## Migration Tracking

### schema_migrations Table

The Lambda function creates and maintains a `schema_migrations` table:

```sql
CREATE TABLE schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    checksum VARCHAR(64) NOT NULL
);
```

### Checking Applied Migrations

```bash
# Connect to database
psql -h <endpoint> -U postgres -d touragent

# View applied migrations
SELECT * FROM schema_migrations ORDER BY version;
```

## How It Works

### Lambda Function Logic

1. **Connect to Database**: Uses credentials from Secrets Manager
2. **Ensure Migrations Table**: Creates `schema_migrations` if not exists
3. **Read Migration Files**: Scans `/var/task/migrations/` directory
4. **Check Applied Migrations**: Queries `schema_migrations` table
5. **Apply Pending Migrations**: Executes new migrations in order
6. **Record Migration**: Inserts record with version, description, checksum
7. **Transaction Safety**: Each migration runs in a transaction (rollback on error)

### Idempotency

The system is idempotent - safe to run multiple times:
- Already applied migrations are skipped
- Only new migrations are executed
- No duplicate execution

## Troubleshooting

### Migration Failed

Check Lambda logs in CloudWatch:

```bash
aws logs tail /aws/lambda/ExplorerAgentDatabaseSchemaStack-SchemaDeployerFunction --follow
```

### Rollback Migration

If a migration fails:
1. Fix the SQL in the migration file
2. Manually delete the failed migration record:
   ```sql
   DELETE FROM schema_migrations WHERE version = '002';
   ```
3. Redeploy: `cdk deploy ExplorerAgentDatabaseSchemaStack`

### Force Redeployment

To force Lambda to re-run (even if no new migrations):

Edit `infrastructure/lib/stacks/database-schema-stack.ts`:
```typescript
properties: {
  Version: '1.0.1', // Increment version
  Timestamp: Date.now(),
}
```

Then deploy:
```bash
cdk deploy ExplorerAgentDatabaseSchemaStack
```

## Best Practices

### Migration Guidelines

1. **One-Way Migrations**: Migrations should only move forward, never modify past migrations
2. **Backward Compatible**: Design migrations to be backward compatible when possible
3. **Test Locally**: Test migrations on local database before deploying
4. **Small Changes**: Keep migrations focused on single logical changes
5. **Descriptive Names**: Use clear, descriptive migration names

### Schema Changes

- **Adding Columns**: Use `ALTER TABLE ADD COLUMN` with defaults
- **Removing Columns**: First deploy code that doesn't use column, then remove
- **Renaming**: Create new column, migrate data, remove old column (3 migrations)
- **Indexes**: Add indexes in separate migrations for large tables

### Example: Safe Column Removal

```sql
-- V003__deprecate_old_column.sql
-- Mark column as deprecated, update code to not use it

-- V004__remove_old_column.sql (deploy after code update)
ALTER TABLE users DROP COLUMN old_column;
```

## Security

- **Credentials**: Stored in AWS Secrets Manager
- **VPC**: Lambda runs in private subnet
- **Encryption**: Database encrypted at rest with KMS
- **TLS**: All connections use TLS 1.2+
- **IAM**: Lambda has minimal permissions (read secrets, VPC access)

## Monitoring

### CloudWatch Logs

Lambda logs include:
- Migration execution progress
- SQL errors
- Applied migration count
- Database connection status

### Alarms

Consider setting up CloudWatch alarms for:
- Lambda execution failures
- Lambda timeout (15 minutes)
- Database connection errors

## Cost

- **Lambda**: Pay per execution (~$0.0001 per deployment)
- **CloudWatch Logs**: ~$0.50/GB ingested
- **Secrets Manager**: ~$0.40/month per secret

Typical cost: < $1/month for schema deployments

## Additional Resources

- [AWS Lambda Documentation](https://docs.aws.amazon.com/lambda/)
- [Aurora PostgreSQL Documentation](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/)
- [CDK Custom Resources](https://docs.aws.amazon.com/cdk/api/v2/docs/aws-cdk-lib.custom_resources-readme.html)

