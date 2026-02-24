# Database Schema Management

## Directory Structure

```
database/
├── migrations/              # SQL migration files
│   └── V001__initial_schema.sql
├── schema-deployer-lambda/  # Lambda function for deployment
│   ├── index.ts
│   ├── package.json
│   └── tsconfig.json
└── README.md
```

## Migration Files

### Location
`infrastructure/database/migrations/`

### Current Migrations

- **V001__initial_schema.sql**: Initial database schema
  - Creates all tables (users, destinations, excursions, bookings, etc.)
  - Creates indexes for query optimization
  - Creates triggers for automatic updates
  - Creates utility functions
  - Creates views for reporting
  - Configures pg_cron for session cleanup
  - Inserts sample category data

## Schema Deployer Lambda

### Purpose
Automatically deploys SQL migrations to the Aurora PostgreSQL database during CDK deployment.

### Features
- **Idempotent**: Safe to run multiple times
- **Versioned**: Tracks applied migrations
- **Transactional**: Rolls back on error
- **Automated**: Triggered by CDK Custom Resource

### How It Works
1. Reads migration files from bundled directory
2. Connects to database using Secrets Manager credentials
3. Creates `schema_migrations` tracking table
4. Applies pending migrations in order
5. Records each migration with checksum

### Environment Variables
- `DB_SECRET_ARN`: ARN of database credentials secret
- `MIGRATIONS_DIR`: Path to migrations directory (default: `/var/task/migrations`)

## Adding New Migrations

### Step 1: Create Migration File

Create a new file in `infrastructure/database/migrations/`:

```bash
# Example: V002__add_notifications.sql
touch infrastructure/database/migrations/V002__add_notifications.sql
```

### Step 2: Write SQL

```sql
-- V002__add_notifications.sql

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
```

### Step 3: Deploy

```bash
cd infrastructure
cdk deploy TourAgentDatabaseSchemaStack
```

The Lambda function will automatically detect and apply the new migration.

## Migration Naming Convention

Format: `V{version}__{description}.sql`

- **Version**: 3-digit number (001, 002, 003, ...)
- **Description**: Snake_case description
- **Examples**:
  - `V001__initial_schema.sql`
  - `V002__add_notifications_table.sql`
  - `V003__add_booking_indexes.sql`

## Schema Migrations Table

The Lambda function creates and maintains a tracking table:

```sql
CREATE TABLE schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    description VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    checksum VARCHAR(64) NOT NULL
);
```

### Viewing Applied Migrations

```sql
SELECT * FROM schema_migrations ORDER BY version;
```

Example output:
```
 version |     description      |         applied_at         |                           checksum
---------+----------------------+----------------------------+--------------------------------------------------------------
 001     | initial schema       | 2024-01-15 10:30:00+00     | a1b2c3d4e5f6...
 002     | add notifications    | 2024-01-20 14:15:00+00     | f6e5d4c3b2a1...
```

## Troubleshooting

### Check Lambda Logs

```bash
aws logs tail /aws/lambda/TourAgentDatabaseSchemaStack-SchemaDeployerFunction --follow
```

### Manually Rollback Failed Migration

If a migration fails and needs to be fixed:

1. Connect to database:
   ```bash
   psql -h <endpoint> -U postgres -d touragent
   ```

2. Delete failed migration record:
   ```sql
   DELETE FROM schema_migrations WHERE version = '002';
   ```

3. Fix the SQL file and redeploy:
   ```bash
   cdk deploy TourAgentDatabaseSchemaStack
   ```

### Force Redeployment

To force Lambda to re-run (even without new migrations):

Edit `infrastructure/lib/stacks/database-schema-stack.ts`:
```typescript
properties: {
  Version: '1.0.1', // Increment this
  Timestamp: Date.now(),
}
```

## Best Practices

1. **Test Locally First**: Test migrations on local PostgreSQL before deploying
2. **One Change Per Migration**: Keep migrations focused
3. **Never Modify Applied Migrations**: Always create new migrations
4. **Backward Compatible**: Design migrations to be backward compatible
5. **Use Transactions**: Wrap complex migrations in BEGIN/COMMIT
6. **Add Comments**: Document why changes are being made

## Security

- Database credentials stored in AWS Secrets Manager
- Lambda runs in private VPC subnet
- TLS encryption for database connections
- IAM permissions follow least privilege principle

## Performance

- Lambda timeout: 15 minutes
- Lambda memory: 512 MB
- Typical execution time: < 1 minute for most migrations
- Large migrations (data backfills) may take longer

## Cost

- Lambda execution: ~$0.0001 per deployment
- CloudWatch Logs: ~$0.50/GB
- Typical monthly cost: < $1

