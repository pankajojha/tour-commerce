# Tour Excursion Booking Platform

A comprehensive AWS-based serverless platform for discovering, booking, and managing shore excursions and tours. The platform supports destination-based selling with intelligent time-window filtering for cruise passengers and flight arrivals, dynamic pricing, real-time inventory management, and multi-channel authentication.

## Features

- **Destination Discovery**: Search and filter excursions by destination, category, price, duration, and difficulty
- **Cruise Integration**: Automatic time-window filtering based on cruise ship arrival/departure schedules
- **Flight Integration**: Excursion filtering based on flight arrival times with transfer buffers
- **Real-Time Inventory**: Prevent overbooking with Redis-based inventory locking (5-minute holds)
- **Dynamic Pricing**: Demand-based, seasonal, and cruise line premium pricing
- **Secure Payments**: Stripe/Adyen integration with PCI DSS compliance
- **Multi-Channel Auth**: Email/password and social media login (Facebook, Google, Instagram, TikTok)
- **Voucher Generation**: PDF vouchers with QR codes for easy check-in
- **Event-Driven Architecture**: Asynchronous processing with EventBridge, SQS, and SNS
- **AI Recommendations**: Personalized excursion suggestions using SageMaker

## Architecture

The platform uses AWS serverless architecture:

- **Compute**: AWS Lambda (Node.js 20.x)
- **API**: API Gateway REST API
- **Database**: Aurora Serverless v2 PostgreSQL (auto-scaling 0.5-128 ACU)
- **Cache**: ElastiCache Redis (inventory locking)
- **Search**: OpenSearch (full-text search)
- **Storage**: S3 + CloudFront CDN
- **Auth**: Amazon Cognito User Pools
- **Messaging**: EventBridge, SQS, SNS
- **Monitoring**: CloudWatch, X-Ray
- **Security**: KMS, Secrets Manager, WAF

## Repository Structure

```
explorer-agent-platform/
├── infrastructure/          # AWS CDK infrastructure code
│   ├── bin/                # CDK app entry point
│   ├── lib/stacks/         # CDK stack definitions
│   └── README.md
├── services/               # Lambda functions (future)
├── database/               # Database schema and migrations (future)
├── docs/                   # Documentation (future)
└── README.md
```

## Getting Started

### Prerequisites

- Node.js 20.x or later
- AWS CLI configured with credentials
- AWS CDK CLI: `npm install -g aws-cdk`

### Installation

```bash
# Install all dependencies
npm install
```

### Deploy Infrastructure

```bash
# Deploy all infrastructure stacks
npm run deploy:infra
```

This will deploy:
1. Network Stack (VPC, subnets, security groups)
2. Security Stack (KMS, Secrets Manager)
3. Database Stack (Aurora Serverless v2 PostgreSQL)
4. Cache Stack (ElastiCache Redis)
5. Storage Stack (S3, CloudFront)
6. Monitoring Stack (CloudWatch log groups)

### Post-Deployment

1. **Update Stripe API Key**:
   ```bash
   aws secretsmanager update-secret \
     --secret-id explorer-agent/stripe/api-key \
     --secret-string "your-stripe-api-key"
   ```

2. **Initialize Database**:
   - Connect to Aurora using credentials from Secrets Manager
   - Run `database/database-schema.sql`

## Infrastructure Components

### Network Stack
- VPC with public, private, and isolated subnets across 3 AZs
- NAT Gateway for private subnet internet access
- Security groups for Lambda, database, and cache
- VPC endpoints for S3 and DynamoDB

### Security Stack
- KMS key with automatic rotation
- Secrets Manager for database credentials, API keys, and Redis auth token

### Database Stack
- Aurora Serverless v2 PostgreSQL 15.4
- Auto-scaling from 0.5 to 128 ACU
- 1 writer + 1 reader instance
- Encryption at rest with KMS
- Automated backups (7-day retention)

### Cache Stack
- ElastiCache Redis 7.0
- 2 nodes (primary + replica)
- Multi-AZ with automatic failover
- Encryption at rest and in transit
- Auth token authentication

### Storage Stack
- S3 bucket for vouchers (KMS encrypted, versioned)
- S3 bucket for static assets
- CloudFront distribution with HTTPS only

### Monitoring Stack
- CloudWatch log groups for all services
- X-Ray tracing enabled
- 30-day log retention

## Development

### Build

```bash
npm run build
```

## Troubleshooting: `tsc` / build issue

- Symptom: running `npm run build` for the `infrastructure` workspace printed:
   "This is not the tsc command you are looking for" and the build failed.

- Root cause: a deprecated/npm package named `tsc` was present in the repository's
   `node_modules` and provided a `tsc` binary that is not the official TypeScript
   compiler. That binary prints the message above when invoked.

- What was changed:
   - Removed the stray `tsc` package from the repo.
   - Updated `infrastructure/package.json` build scripts to use `npm exec -- tsc`
      which runs the `tsc` from installed `typescript` rather than invoking an
      external `tsc` package via `npx`.

- Commands to reproduce or verify the fix:

```bash
# refresh deps and run infra build
npm install
npm --workspace=infrastructure run build
```

- Alternatives / quick fixes:
   - Install TypeScript locally in the workspace: `npm install --workspace=infrastructure --save-dev typescript`
   - Remove any stray `tsc` package: `npm uninstall tsc` (repo root)

If the build fails after these steps, check `node_modules/.bin/tsc` to ensure it points
to the `typescript` binary and not the deprecated `tsc` package.
```

### Lint

```bash
npm run lint
```

### Format

```bash
npm run format
```

## Documentation

- [Infrastructure README](infrastructure/README.md) - Detailed infrastructure documentation
- [Design Document](.kiro/specs/tour-excursion-booking-platform/design.md) - System design and architecture
- [Requirements](.kiro/specs/tour-excursion-booking-platform/requirements.md) - Functional and non-functional requirements
- [Tasks](.kiro/specs/tour-excursion-booking-platform/tasks.md) - Implementation tasks

## Security

- All data encrypted at rest (KMS) and in transit (TLS 1.2+)
- Private subnets for databases and cache
- Security groups with least privilege
- Secrets stored in Secrets Manager
- PCI DSS compliant payment processing (delegated to Stripe/Adyen)
- GDPR compliant (right to access, erasure, data portability)

## Cost Optimization

- Aurora Serverless v2 scales down to 0.5 ACU when idle
- Single NAT Gateway
- S3 lifecycle policies
- CloudWatch log retention policies
- CloudFront caching

## License

ISC

## Support

For issues and questions, please refer to the documentation in the `.kiro/specs/` directory.


## AWS SSO
aws sts get-caller-identity --profile AdministratorAccess-727974078184
aws sso login  --profile AdministratorAccess-727974078184

https://d-906603d3bd.awsapps.com/start/#/
export CDK_DEFAULT_ACCOUNT=727974078184
export CDK_DEFAULT_REGION=us-east-1
export AWS_PROFILE=AdministratorAccess-727974078184


cd ~/explorer-agent
git add infrastructure/lib/stacks/database-schema-stack.ts
git commit -m "feat(cdk): enable local bundling for Lambda to avoid Docker permission issues"
npm --workspace=infrastructure run synth


rm -rf node_modules package-lock.json infrastructure/node_modules infrastructure/package-lock.json && npm install && npm --workspace=infrastructure run synth 2>&1 | grep -i "EBADENGINE\|deprecated" | head -10


React 18 with TypeScript, Vite, and Tailwind CSS