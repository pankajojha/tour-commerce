# Branding Update: Tour Agent → Explorer Agent

## ✅ Completed Changes

All references to "tour agent" have been updated to "explorer agent" across the entire codebase.

### Files Updated

**Terraform Infrastructure:**
- `terraform/main.tf` - Project tags
- `terraform/modules/auth/main.tf` - Cognito User Pool, domain, clients
- `terraform/modules/auth-lambdas/main.tf` - Lambda function names, IAM roles
- `terraform/modules/security/main.tf` - Secret names
- `terraform/modules/database/main.tf` - Database cluster name
- `terraform/modules/network/main.tf` - VPC and security group names
- `terraform/modules/cache/main.tf` - Redis cluster name
- `terraform/modules/storage/main.tf` - S3 bucket names
- `terraform/modules/monitoring/main.tf` - CloudWatch log group names

**Go Services:**
- `services/auth/go.mod` - Module path: `github.com/explorer-agent/auth`
- All Lambda function code (no code changes needed, just module reference)

**Documentation:**
- `README.md`
- `DEPLOYMENT_READY.md`
- `TASK3_PROGRESS.md`
- `services/auth/README.md`
- `infrastructure/README.md`

**Configuration:**
- `package.json` - Project name
- `infrastructure/package.json` - Project name

### Resource Name Changes

**Before → After:**

**Cognito:**
- `tour-agent-user-pool` → `explorer-agent-user-pool`
- `tour-agent-web-client` → `explorer-agent-web-client`
- `tour-agent-mobile-client` → `explorer-agent-mobile-client`
- `tour-agent-${env}` (domain) → `explorer-agent-${env}`

**Lambda Functions:**
- `tour-agent-auth-signup` → `explorer-agent-auth-signup`
- `tour-agent-auth-login` → `explorer-agent-auth-login`
- `tour-agent-auth-passwordless-request` → `explorer-agent-auth-passwordless-request`
- `tour-agent-auth-passwordless-verify` → `explorer-agent-auth-passwordless-verify`
- `tour-agent-auth-token-refresh` → `explorer-agent-auth-token-refresh`

**IAM:**
- `tour-agent-auth-lambda-role` → `explorer-agent-auth-lambda-role`
- `tour-agent-auth-lambda-policy` → `explorer-agent-auth-lambda-policy`

**Secrets Manager:**
- `tour-agent/oauth/credentials` → `explorer-agent/oauth/credentials`
- `tour-agent/stripe/api-key` → `explorer-agent/stripe/api-key`
- `tour-agent/redis/auth-token` → `explorer-agent/redis/auth-token`
- `tour-agent/database/credentials` → `explorer-agent/database/credentials`

**Database:**
- `tour-agent-database` → `explorer-agent-database`
- `tour-agent-db-subnet-group` → `explorer-agent-db-subnet-group`

**Network:**
- `tour-agent-vpc` → `explorer-agent-vpc`
- `tour-agent-lambda-sg` → `explorer-agent-lambda-sg`
- `tour-agent-database-sg` → `explorer-agent-database-sg`
- `tour-agent-cache-sg` → `explorer-agent-cache-sg`

**Cache:**
- `tour-agent-redis` → `explorer-agent-redis`
- `tour-agent-redis-subnet-group` → `explorer-agent-redis-subnet-group`

**Storage:**
- `tour-agent-vouchers-{account}` → `explorer-agent-vouchers-{account}`
- `tour-agent-assets-{account}` → `explorer-agent-assets-{account}`
- `tour-agent-cloudfront-logs-{account}` → `explorer-agent-cloudfront-logs-{account}`

**CloudWatch Log Groups:**
- `/aws/apigateway/tour-agent` → `/aws/apigateway/explorer-agent`
- `/aws/lambda/tour-agent` → `/aws/lambda/explorer-agent`
- `/aws/lambda/tour-agent-auth-*` → `/aws/lambda/explorer-agent-auth-*`
- `/aws/rds/cluster/tour-agent-database/*` → `/aws/rds/cluster/explorer-agent-database/*`
- `/aws/elasticache/tour-agent-redis` → `/aws/elasticache/explorer-agent-redis`
- `/aws/xray/tour-agent` → `/aws/xray/explorer-agent`
- `/aws/events/tour-agent` → `/aws/events/explorer-agent`
- `/aws/sqs/tour-agent` → `/aws/sqs/explorer-agent`

**KMS:**
- `tour-agent-platform` (alias) → `explorer-agent-platform`

**Tags:**
- `Project: TourAgentPlatform` → `Project: ExplorerAgentPlatform`

## ✅ Verification

- ✅ Lambda functions rebuilt successfully
- ✅ Terraform configuration validated
- ✅ Go module path updated
- ✅ All documentation updated

## 📝 Next Steps

The rebranding is complete. When you deploy, all new resources will be created with the "explorer-agent" naming convention.

**Important Notes:**

1. **Fresh Deployment:** Since this is a new deployment, all resources will be created with the new names.

2. **If Resources Already Exist:** If you had previously deployed with "tour-agent" names, Terraform will want to:
   - Destroy old resources (tour-agent-*)
   - Create new resources (explorer-agent-*)
   
   Review the plan carefully before applying!

3. **Email Verification:** Update your SES email verification:
   ```bash
   aws ses verify-email-identity --email-address noreply@explorerAgent.com
   ```

4. **Domain Updates:** If you have a custom domain, update:
   - `from_email` variable in terraform.tfvars
   - `app_url` variable in terraform.tfvars

## 🚀 Ready to Deploy

All code has been updated and validated. Proceed with deployment when ready:

```bash
cd terraform
terraform plan -out=tfplan
terraform apply tfplan
```
