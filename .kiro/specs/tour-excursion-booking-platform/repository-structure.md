# Repository Structure Plan: Tour Excursion Booking Platform

## Overview

This document outlines the recommended Git repository structure and code organization for the tour excursion booking platform. The goal is to start small while maintaining scalability and clear separation of concerns.

---

## Recommended Approach: Monorepo

**Repository Name:** `tour-booking-platform`

### Why Monorepo?

- Single source of truth for all code
- Easy to share code between services
- Simplified dependency management
- Easier to maintain consistency across services
- Good for small teams starting out (1-5 developers)
- Single CI/CD pipeline with clear stages
- Can evolve to multi-repo later if needed

---

## Repository Structure

```
tour-booking-platform/
├── .github/
│   └── workflows/              # CI/CD pipelines
│       ├── infrastructure.yml  # Deploy CDK stacks
│       ├── services.yml        # Deploy Lambda functions
│       └── frontend.yml        # Deploy frontend (future)
│
├── infrastructure/             # AWS CDK Infrastructure as Code
│   ├── bin/
│   │   └── app.ts             # CDK app entry point
│   ├── lib/
│   │   ├── stacks/
│   │   │   ├── network-stack.ts        # VPC, subnets, security groups
│   │   │   ├── database-stack.ts       # Aurora Serverless v2 PostgreSQL
│   │   │   ├── cache-stack.ts          # ElastiCache Redis
│   │   │   ├── search-stack.ts         # OpenSearch
│   │   │   ├── storage-stack.ts        # S3, CloudFront
│   │   │   ├── auth-stack.ts           # Cognito User Pool
│   │   │   ├── api-stack.ts            # API Gateway
│   │   │   ├── messaging-stack.ts      # EventBridge, SQS, SNS
│   │   │   ├── monitoring-stack.ts     # CloudWatch, X-Ray
│   │   │   ├── security-stack.ts       # WAF, KMS, Secrets Manager
│   │   │   ├── ai-stack.ts             # AWS Bedrock, DynamoDB for memory
│   │   │   └── analytics-stack.ts      # Analytics database, reporting
│   │   └── constructs/                 # Reusable CDK constructs
│   │       ├── lambda-function.ts
│   │       ├── api-endpoint.ts
│   │       └── database-table.ts
│   ├── cdk.json
│   ├── package.json
│   ├── tsconfig.json
│   └── README.md
│
├── services/                   # Backend Lambda functions
│   ├── shared/                 # Shared utilities and types
│   │   ├── types/
│   │   │   ├── booking.ts
│   │   │   ├── excursion.ts
│   │   │   ├── user.ts
│   │   │   ├── payment.ts
│   │   │   ├── itinerary.ts
│   │   │   ├── restaurant.ts
│   │   │   ├── shopping.ts
│   │   │   ├── place.ts
│   │   │   ├── recommendation.ts
│   │   │   └── index.ts
│   │   ├── utils/
│   │   │   ├── timezone.ts
│   │   │   ├── validation.ts
│   │   │   ├── logger.ts
│   │   │   ├── error-handler.ts
│   │   │   └── index.ts
│   │   ├── database/
│   │   │   ├── connection.ts
│   │   │   ├── repositories/
│   │   │   │   ├── booking-repository.ts
│   │   │   │   ├── excursion-repository.ts
│   │   │   │   ├── user-repository.ts
│   │   │   │   ├── itinerary-repository.ts
│   │   │   │   ├── restaurant-repository.ts
│   │   │   │   ├── shopping-repository.ts
│   │   │   │   ├── place-repository.ts
│   │   │   │   └── recommendation-repository.ts
│   │   │   └── index.ts
│   │   ├── cache/
│   │   │   ├── redis-client.ts
│   │   │   ├── inventory-cache.ts
│   │   │   └── index.ts
│   │   ├── external/
│   │   │   ├── google-places-client.ts
│   │   │   ├── google-trends-client.ts
│   │   │   ├── bedrock-client.ts
│   │   │   └── index.ts
│   │   ├── middleware/
│   │   │   ├── auth.ts
│   │   │   ├── error-handler.ts
│   │   │   └── index.ts
│   │   └── package.json
│   │
│   ├── auth/                   # Authentication service
│   │   ├── handlers/
│   │   │   ├── signup.ts
│   │   │   ├── login.ts
│   │   │   ├── social-login.ts
│   │   │   ├── refresh-token.ts
│   │   │   └── password-reset.ts
│   │   ├── tests/
│   │   │   ├── signup.test.ts
│   │   │   ├── login.test.ts
│   │   │   └── social-login.test.ts
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── destinations/           # Destination service
│   │   ├── handlers/
│   │   │   ├── search-destinations.ts
│   │   │   ├── get-destination.ts
│   │   │   ├── get-by-port-code.ts
│   │   │   └── get-by-airport-code.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── excursions/            # Excursion service
│   │   ├── handlers/
│   │   │   ├── search-excursions.ts
│   │   │   ├── get-excursion.ts
│   │   │   ├── get-schedule.ts
│   │   │   ├── create-excursion.ts
│   │   │   └── update-excursion.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── cruise-schedules/      # Cruise schedule service
│   │   ├── handlers/
│   │   │   ├── get-port-call.ts
│   │   │   ├── search-port-calls.ts
│   │   │   ├── calculate-time-window.ts
│   │   │   ├── get-excursions-for-cruise.ts
│   │   │   └── import-schedules.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── flight-schedules/      # Flight schedule service
│   │   ├── handlers/
│   │   │   ├── get-flight-arrival.ts
│   │   │   ├── search-flight-arrivals.ts
│   │   │   ├── calculate-earliest-start.ts
│   │   │   ├── get-excursions-for-flight.ts
│   │   │   └── import-flight-schedules.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── inventory/             # Inventory management
│   │   ├── handlers/
│   │   │   ├── check-availability.ts
│   │   │   ├── lock-inventory.ts
│   │   │   ├── release-lock.ts
│   │   │   ├── confirm-booking.ts
│   │   │   ├── get-inventory-status.ts
│   │   │   ├── update-capacity.ts
│   │   │   └── set-blackout-date.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── pricing/               # Pricing engine
│   │   ├── handlers/
│   │   │   ├── calculate-price.ts
│   │   │   ├── get-pricing-tiers.ts
│   │   │   ├── set-pricing-rule.ts
│   │   │   └── set-seasonal-pricing.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── payments/              # Payment processing
│   │   ├── handlers/
│   │   │   ├── create-payment-intent.ts
│   │   │   ├── confirm-payment.ts
│   │   │   ├── process-refund.ts
│   │   │   ├── get-payment-status.ts
│   │   │   └── webhook-handler.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── bookings/              # Booking service
│   │   ├── handlers/
│   │   │   ├── create-booking.ts
│   │   │   ├── get-booking.ts
│   │   │   ├── list-user-bookings.ts
│   │   │   ├── cancel-booking.ts
│   │   │   ├── modify-booking.ts
│   │   │   └── get-voucher.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── vouchers/              # Voucher generation
│   │   ├── handlers/
│   │   │   ├── generate-voucher.ts
│   │   │   └── scan-qr-code.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── notifications/         # Notification service
│   │   ├── handlers/
│   │   │   ├── send-email.ts
│   │   │   ├── send-sms.ts
│   │   │   ├── booking-confirmation.ts
│   │   │   ├── supplier-notification.ts
│   │   │   ├── booking-reminder.ts
│   │   │   └── cancellation-notification.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── admin/                 # Admin dashboards
│   │   ├── handlers/
│   │   │   ├── supplier-dashboard.ts
│   │   │   ├── operations-dashboard.ts
│   │   │   ├── manage-excursion.ts
│   │   │   ├── get-booking-manifest.ts
│   │   │   ├── export-manifest.ts
│   │   │   └── generate-reports.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── reviews/               # Review system
│   │   ├── handlers/
│   │   │   ├── submit-review.ts
│   │   │   ├── get-reviews.ts
│   │   │   ├── vote-helpful.ts
│   │   │   └── add-supplier-response.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── recommendations/       # Enhanced recommendation engine
│   │   ├── handlers/
│   │   │   ├── get-recommendations.ts
│   │   │   ├── calculate-recommendation-score.ts
│   │   │   ├── get-cross-sell.ts
│   │   │   ├── get-upsell.ts
│   │   │   ├── get-trending.ts
│   │   │   ├── get-user-segment.ts
│   │   │   └── update-user-behavior.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── dining/                # Dining & restaurant service
│   │   ├── handlers/
│   │   │   ├── search-restaurants.ts
│   │   │   ├── get-restaurant.ts
│   │   │   ├── get-restaurants-by-destination.ts
│   │   │   ├── get-near-port.ts
│   │   │   ├── create-restaurant.ts
│   │   │   ├── update-restaurant.ts
│   │   │   └── generate-affiliate-link.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── shopping/              # Shopping & local products service
│   │   ├── handlers/
│   │   │   ├── search-shopping-locations.ts
│   │   │   ├── get-shopping-location.ts
│   │   │   ├── get-local-products.ts
│   │   │   ├── get-trending-products.ts
│   │   │   ├── create-shopping-location.ts
│   │   │   ├── update-shopping-location.ts
│   │   │   └── generate-affiliate-link.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── places/                # Google Places integration
│   │   ├── handlers/
│   │   │   ├── sync-google-places.ts
│   │   │   ├── get-place-details.ts
│   │   │   ├── search-places.ts
│   │   │   ├── update-popularity-score.ts
│   │   │   └── import-places-batch.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── search-integration/    # Search integration service
│   │   ├── handlers/
│   │   │   ├── sync-google-trends.ts
│   │   │   ├── get-trending-keywords.ts
│   │   │   ├── aggregate-ratings.ts
│   │   │   ├── get-popular-searches.ts
│   │   │   └── analyze-seasonal-trends.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── itinerary/             # Itinerary management service
│   │   ├── handlers/
│   │   │   ├── create-itinerary.ts
│   │   │   ├── get-itinerary.ts
│   │   │   ├── update-itinerary.ts
│   │   │   ├── delete-itinerary.ts
│   │   │   ├── list-user-itineraries.ts
│   │   │   ├── add-item.ts
│   │   │   ├── update-item.ts
│   │   │   ├── delete-item.ts
│   │   │   ├── detect-conflicts.ts
│   │   │   ├── share-itinerary.ts
│   │   │   ├── revoke-share.ts
│   │   │   ├── update-share-permissions.ts
│   │   │   └── import-booking.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   ├── ai-assistant/          # AI assistant service (AWS Bedrock)
│   │   ├── handlers/
│   │   │   ├── process-message.ts
│   │   │   ├── get-recommendations-ai.ts
│   │   │   ├── detect-conflicts-ai.ts
│   │   │   ├── summarize-itinerary.ts
│   │   │   ├── get-conversation-history.ts
│   │   │   ├── manage-memory.ts
│   │   │   └── mcp-integration.ts
│   │   ├── prompts/
│   │   │   ├── system-prompt.ts
│   │   │   ├── recommendation-prompt.ts
│   │   │   ├── conflict-detection-prompt.ts
│   │   │   └── guardrails.ts
│   │   ├── tests/
│   │   ├── package.json
│   │   └── README.md
│   │
│   └── user-profile/          # User profile management
│       ├── handlers/
│       │   ├── get-user-profile.ts
│       │   ├── update-user-profile.ts
│       │   ├── update-email.ts
│       │   ├── get-booking-history.ts
│       │   ├── update-notification-preferences.ts
│       │   ├── disconnect-social-account.ts
│       │   ├── delete-account.ts
│       │   └── export-user-data.ts
│       ├── tests/
│       ├── package.json
│       └── README.md
│
├── database/                   # Database migrations and seeds
│   ├── migrations/
│   │   ├── 001_initial_schema.sql
│   │   ├── 002_add_indexes.sql
│   │   ├── 003_add_cruise_schedules.sql
│   │   ├── 004_add_flight_schedules.sql
│   │   ├── 005_add_itinerary_tables.sql
│   │   ├── 006_add_dining_tables.sql
│   │   ├── 007_add_shopping_tables.sql
│   │   ├── 008_add_places_tables.sql
│   │   ├── 009_add_recommendation_tables.sql
│   │   └── ...
│   ├── seeds/
│   │   ├── destinations.sql
│   │   ├── categories.sql
│   │   ├── sample_excursions.sql
│   │   ├── sample_restaurants.sql
│   │   ├── sample_shopping_locations.sql
│   │   └── sample_places.sql
│   ├── flyway.conf
│   └── README.md
│
├── frontend/                   # Web application (optional - future)
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   ├── services/
│   │   └── utils/
│   ├── public/
│   ├── package.json
│   └── README.md
│
├── tests/                      # Integration and E2E tests
│   ├── integration/
│   │   ├── booking-flow.test.ts
│   │   ├── cruise-integration.test.ts
│   │   ├── payment-flow.test.ts
│   │   ├── inventory-locking.test.ts
│   │   ├── itinerary-management.test.ts
│   │   ├── conflict-detection.test.ts
│   │   ├── recommendation-engine.test.ts
│   │   ├── dining-integration.test.ts
│   │   ├── shopping-integration.test.ts
│   │   ├── ai-assistant.test.ts
│   │   └── event-driven-workflows.test.ts
│   ├── e2e/
│   │   ├── complete-booking.test.ts
│   │   ├── user-journey.test.ts
│   │   ├── itinerary-creation.test.ts
│   │   └── multi-revenue-stream.test.ts
│   ├── property-based/
│   │   ├── inventory-consistency.test.ts
│   │   ├── no-overbooking.test.ts
│   │   ├── time-window-validity.test.ts
│   │   ├── price-positivity.test.ts
│   │   ├── recommendation-score-bounds.test.ts
│   │   └── conflict-detection-accuracy.test.ts
│   ├── package.json
│   └── README.md
│
├── docs/                       # Documentation
│   ├── api/
│   │   ├── openapi.yaml
│   │   └── postman-collection.json
│   ├── architecture/
│   │   ├── diagrams/
│   │   │   ├── system-architecture.png
│   │   │   ├── data-flow.png
│   │   │   └── deployment.png
│   │   └── decisions/
│   │       ├── 001-use-redis-for-inventory.md
│   │       └── 002-monorepo-structure.md
│   ├── runbooks/
│   │   ├── deployment.md
│   │   ├── rollback.md
│   │   ├── disaster-recovery.md
│   │   └── troubleshooting.md
│   └── README.md
│
├── scripts/                    # Utility scripts
│   ├── deploy.sh              # Deploy all services
│   ├── deploy-infrastructure.sh
│   ├── deploy-services.sh
│   ├── seed-data.sh           # Seed database with sample data
│   ├── run-tests.sh           # Run all tests
│   ├── run-integration-tests.sh
│   └── setup-local-env.sh     # Setup local development environment
│
├── .github/                    # GitHub specific files
│   ├── workflows/
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── ISSUE_TEMPLATE/
│
├── package.json               # Root package.json (workspace)
├── tsconfig.json              # Root TypeScript config
├── .gitignore
├── .eslintrc.js
├── .prettierrc
├── jest.config.js             # Root Jest config
├── .nvmrc                     # Node version
├── LICENSE
└── README.md
```

---

## Package Management: NPM Workspaces

### Root `package.json`

```json
{
  "name": "tour-booking-platform",
  "version": "1.0.0",
  "private": true,
  "workspaces": [
    "infrastructure",
    "services/shared",
    "services/*",
    "tests"
  ],
  "scripts": {
    "install:all": "npm install",
    "build": "npm run build --workspaces",
    "test": "npm run test --workspaces",
    "test:integration": "npm run test --workspace=tests",
    "lint": "eslint . --ext .ts",
    "format": "prettier --write \"**/*.{ts,json,md}\"",
    "deploy:infra": "cd infrastructure && npm run deploy",
    "deploy:services": "./scripts/deploy-services.sh",
    "deploy:all": "npm run deploy:infra && npm run deploy:services"
  },
  "devDependencies": {
    "@types/node": "^20.0.0",
    "@typescript-eslint/eslint-plugin": "^6.0.0",
    "@typescript-eslint/parser": "^6.0.0",
    "eslint": "^8.0.0",
    "prettier": "^3.0.0",
    "typescript": "^5.0.0",
    "jest": "^29.0.0",
    "ts-jest": "^29.0.0"
  }
}
```

### Benefits:
- Single `npm install` at root installs all dependencies
- Shared dependencies at root level (TypeScript, ESLint, Jest)
- Service-specific dependencies in each service folder
- Easy to run commands across all workspaces

---

## Service Structure Example

### Example: `services/bookings/`

```
services/bookings/
├── handlers/
│   ├── create-booking.ts       # Lambda handler
│   ├── get-booking.ts
│   ├── list-user-bookings.ts
│   ├── cancel-booking.ts
│   └── modify-booking.ts
├── tests/
│   ├── create-booking.test.ts  # Unit tests
│   ├── cancel-booking.test.ts
│   └── booking-validation.test.ts
├── package.json
└── README.md
```

### Example Handler: `create-booking.ts`

```typescript
import { APIGatewayProxyHandler } from 'aws-lambda';
import { BookingRepository } from '../../shared/database/repositories/booking-repository';
import { validateBooking } from '../../shared/utils/validation';
import { logger } from '../../shared/utils/logger';

export const handler: APIGatewayProxyHandler = async (event) => {
  try {
    const booking = JSON.parse(event.body || '{}');
    
    // Validate
    validateBooking(booking);
    
    // Create booking
    const bookingRepo = new BookingRepository();
    const result = await bookingRepo.create(booking);
    
    logger.info('Booking created', { bookingId: result.id });
    
    return {
      statusCode: 201,
      body: JSON.stringify(result)
    };
  } catch (error) {
    logger.error('Error creating booking', { error });
    return {
      statusCode: 500,
      body: JSON.stringify({ error: 'Internal server error' })
    };
  }
};
```

---

## Infrastructure Structure

### CDK Stack Organization

Each stack is responsible for a specific AWS service category:

1. **network-stack.ts** - VPC, subnets, NAT Gateway, security groups
2. **database-stack.ts** - Aurora Serverless v2 PostgreSQL
3. **cache-stack.ts** - ElastiCache Redis cluster
4. **search-stack.ts** - OpenSearch domain
5. **storage-stack.ts** - S3 buckets, CloudFront distribution
6. **auth-stack.ts** - Cognito User Pool, Identity Providers
7. **api-stack.ts** - API Gateway, Lambda integrations
8. **messaging-stack.ts** - EventBridge, SQS queues, SNS topics
9. **monitoring-stack.ts** - CloudWatch dashboards, alarms, X-Ray
10. **security-stack.ts** - WAF, KMS keys, Secrets Manager
11. **ai-stack.ts** - AWS Bedrock, DynamoDB for AI memory stores, SageMaker endpoints
12. **analytics-stack.ts** - Separate analytics database, reporting infrastructure

### Example Stack: `api-stack.ts`

```typescript
import * as cdk from 'aws-cdk-lib';
import * as apigateway from 'aws-cdk-lib/aws-apigateway';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import { Construct } from 'constructs';

export class ApiStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // Create REST API
    const api = new apigateway.RestApi(this, 'TourBookingApi', {
      restApiName: 'Tour Booking API',
      description: 'API for tour excursion booking platform',
    });

    // Add resources and methods
    const bookings = api.root.addResource('bookings');
    // ... add Lambda integrations
  }
}
```

---

## CI/CD Pipeline Strategy

### GitHub Actions Workflow

```yaml
# .github/workflows/deploy.yml
name: Deploy Tour Booking Platform

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '20'
      - run: npm install
      - run: npm run lint
      - run: npm run test
      - run: npm run test:integration

  deploy-infrastructure:
    needs: test
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - run: npm install
      - run: npm run deploy:infra
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}

  deploy-services:
    needs: deploy-infrastructure
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - run: npm install
      - run: npm run deploy:services
        env:
          AWS_ACCESS_KEY_ID: ${{ secrets.AWS_ACCESS_KEY_ID }}
          AWS_SECRET_ACCESS_KEY: ${{ secrets.AWS_SECRET_ACCESS_KEY }}

  smoke-tests:
    needs: deploy-services
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: npm run test:e2e
```

### Deployment Stages

1. **Test** - Run linting, unit tests, integration tests
2. **Deploy Infrastructure** - Deploy CDK stacks (VPC, databases, etc.)
3. **Deploy Services** - Deploy Lambda functions
4. **Smoke Tests** - Run E2E tests against deployed environment

---

## Alternative: Multi-Repo Structure

If you decide to scale to multiple repositories later:

### Repository 1: `tour-booking-infrastructure`
- Contains only CDK infrastructure code
- Deployed independently
- Versioned separately

### Repository 2: `tour-booking-services`
- Contains all Lambda services
- Depends on infrastructure outputs
- Can deploy services independently

### Repository 3: `tour-booking-database`
- Contains database migrations and seeds
- Deployed before services

### Repository 4: `tour-booking-frontend`
- Contains web application
- Deployed independently
- Consumes API from services

---

## Development Workflow

### Local Development

1. **Clone repository**
   ```bash
   git clone https://github.com/your-org/tour-booking-platform.git
   cd tour-booking-platform
   ```

2. **Install dependencies**
   ```bash
   npm install
   ```

3. **Setup local environment**
   ```bash
   ./scripts/setup-local-env.sh
   ```

4. **Run tests**
   ```bash
   npm run test
   ```

5. **Deploy to dev environment**
   ```bash
   npm run deploy:infra
   npm run deploy:services
   ```

### Branch Strategy

- `main` - Production environment
- `develop` - Development environment
- `feature/*` - Feature branches
- `bugfix/*` - Bug fix branches
- `hotfix/*` - Production hotfixes

### Pull Request Process

1. Create feature branch from `develop`
2. Implement changes
3. Run tests locally
4. Create pull request to `develop`
5. CI/CD runs tests automatically
6. Code review
7. Merge to `develop`
8. Deploy to dev environment
9. QA testing
10. Merge to `main` for production deployment

---

## Technology Stack Summary

### Infrastructure
- **IaC**: AWS CDK (TypeScript)
- **Compute**: AWS Lambda (Node.js 20.x)
- **API**: API Gateway REST API
- **Database**: Aurora Serverless v2 PostgreSQL
- **Cache**: ElastiCache Redis
- **Search**: OpenSearch / Elasticsearch
- **Storage**: S3, CloudFront
- **Auth**: Cognito
- **Messaging**: EventBridge, SQS, SNS
- **Monitoring**: CloudWatch, X-Ray
- **Security**: WAF, KMS, Secrets Manager
- **AI/ML**: AWS Bedrock (AgentCore), SageMaker, DynamoDB (AI memory)
- **Analytics**: Separate analytics database for reporting

### Application
- **Language**: TypeScript
- **Runtime**: Node.js 20.x
- **Testing**: Jest, fast-check (property-based testing)
- **Linting**: ESLint
- **Formatting**: Prettier
- **Package Manager**: npm (workspaces)

### External Integrations
- **Google Places API** - Restaurant and shopping location data
- **Google Trends API** - Trending search keywords and seasonal analysis
- **Stripe/Adyen** - Payment processing
- **Cruise Schedule Feeds** - Port call data
- **Flight Schedule APIs** - Flight arrival data

### New Services & Features
- **Dining Service** - Restaurant recommendations with affiliate links
- **Shopping Service** - Shopping locations and local products
- **Places Service** - Google Places integration and synchronization
- **Search Integration** - Google Trends and popularity scoring
- **Itinerary Management** - Trip planning with conflict detection
- **AI Assistant** - AWS Bedrock-powered conversational AI
- **Enhanced Recommendations** - Multi-factor scoring (context, behavior, popularity, personalization)

### Revenue Streams
1. **Excursion Commissions** (15-25%)
2. **Restaurant Affiliate Commissions** (5-15%)
3. **Shopping Affiliate Commissions** (3-10%)

### CI/CD
- **Pipeline**: GitHub Actions
- **Deployment**: AWS CDK, AWS SAM (optional)
- **Environments**: dev, staging, production

---

## Key Architecture Decisions

### 1. Multi-Revenue Stream Architecture
The platform now supports three revenue streams (excursions, dining, shopping) with a unified recommendation engine that personalizes suggestions across all categories.

### 2. AI-Powered Features
- **AWS Bedrock** for conversational AI and trip planning assistance
- **DynamoDB** for short-term and long-term AI memory
- **SageMaker** for ML-based recommendation models
- **MCP Integration** for external tool access

### 3. Enhanced Recommendation Engine
Multi-factor scoring algorithm:
- 30% Context-based (destination, time, budget)
- 25% Behavioral (bookings, searches, clicks)
- 25% Popularity (ratings, trends, velocity)
- 20% Personalization (ML-based user segmentation)

### 4. User Segmentation
Five traveler segments:
- Luxury Traveler
- Family Traveler
- Budget Traveler
- Food-Focused Traveler
- Adventure Traveler

### 5. Itinerary Management
- Multi-item type support (excursions, flights, hotels, cruises, leisure)
- Automatic conflict detection
- Sharing with view/edit permissions
- Import functionality for external bookings

---

## Questions to Validate

Before proceeding with implementation, please confirm:

1. **Monorepo vs Multi-Repo**: Is the monorepo approach acceptable for your team size and workflow?

2. **Package Manager**: Is npm workspaces acceptable, or do you prefer yarn/pnpm?

3. **CI/CD Tool**: Is GitHub Actions acceptable, or do you prefer GitLab CI, AWS CodePipeline, or another tool?

4. **Node.js Version**: Is Node.js 20.x acceptable for Lambda runtime?

5. **Testing Framework**: Is Jest acceptable for unit/integration tests?

6. **Frontend**: Should the frontend be in the same repo or separate? (Recommend same repo initially)

7. **Database Migrations**: Is Flyway acceptable, or do you prefer another tool (Liquibase, TypeORM migrations)?

8. **Deployment Strategy**: Deploy all services together or independently?

9. **Environment Strategy**: How many environments do you need? (dev, staging, production)

10. **Naming Conventions**: Any preferences for naming conventions (kebab-case, camelCase, PascalCase)?

---

## Next Steps

Once you validate this structure:

1. Create the Git repository
2. Set up the initial folder structure
3. Configure npm workspaces
4. Set up TypeScript, ESLint, Prettier
5. Create initial CDK project
6. Set up CI/CD pipeline
7. Begin implementing Task 1 from tasks.md (Infrastructure setup)

Please review and provide feedback on this structure before we proceed with implementation.
