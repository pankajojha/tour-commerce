# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## Repository Layout

This is a monorepo for a cruise shore-excursion booking platform. Six top-level workspaces plus a business-docs folder:

| Directory | Language / Runtime | Role |
|---|---|---|
| `tour-agent/` | Go 1.21 | Core transactional backend services — auth, trips, policy, AI proxy |
| `tour-agent-infra/` | Terraform + AWS CDK (TypeScript) | All tour-agent infrastructure and database code — Terraform roots, CDK stacks, Flyway migrations, local Postgres |
| `tour-planner-ai/` | Python 3.11 + AWS CDK (TypeScript) | Bedrock AgentCore AI orchestration |
| `tour-planner-ui/` | React 19 + TypeScript + Vite | Frontend SPA |
| `tour-commerce/` | Go 1.21 + AWS CDK (TypeScript) | Commerce layer — cart, checkout, orders, payments |
| `tour-planner-cms/` | Strapi (Node.js) | Headless CMS |
| `tour-agent-docs/` | Markdown | Business / go-to-market documentation (not code) |

> **IaC note**: `tour-agent-infra/terraform/` is the **live** infrastructure-as-code for the deployed AWS environment (local state file, API Gateway, auth Lambdas, Cognito) — deploy with targeted plans (`terraform plan -target=… -out=…` then `terraform apply <plan>`). `tour-agent-infra/infrastructure/` holds the AWS CDK stacks (fuller design, not the currently deployed path). `tour-agent-infra/findmytour-tf/` manages the findmytour.ai Cloudflare deployment.

---

## Commands

### tour-agent — Go Lambda services

Each service under `tour-agent/services/<name>/` is an independent Go module with its own `Makefile`.

The **auth** service is special: it is a single Go module (`services/auth/`) that contains **multiple sub-Lambda handlers**, each in its own subdirectory:
`login/`, `oauth-start/`, `oauth-callback/`, `passwordless-request/`, `passwordless-verify/`, `pre-signup/`, `signup/`, `token-refresh/`
Each sub-handler has its own `main.go` and is built/deployed independently.

```bash
# Build any service
cd tour-agent/services/trips && make build       # produces dist/bootstrap
cd tour-agent/services/auth  && make build       # produces dist/handler.zip (all sub-handlers)
cd tour-agent/services/policy && make build
cd tour-agent/services/ai-proxy && make build

# Run tests for a service
cd tour-agent/services/trips && go test ./...
cd tour-agent/services/auth  && go test ./... -v

# Download/tidy deps
cd tour-agent/services/trips && make deps

# Build + zip + deploy a single Lambda
cd tour-agent/services/trips && make deploy FUNCTION_NAME=my-function
```

Infrastructure (CDK):
```bash
cd tour-agent-infra/infrastructure
npm install
npm run build      # use this, NOT bare `tsc` — see note below
npx cdk synth
npx cdk deploy --all
```

> **tsc note**: a stray `tsc` npm package can shadow the TypeScript compiler. Always use `npm exec -- tsc` or `npm run build`, not a bare `tsc` invocation.

AWS identity (account `727974078184`, region `us-east-1`):
```bash
export CDK_DEFAULT_ACCOUNT=727974078184
export CDK_DEFAULT_REGION=us-east-1
export AWS_PROFILE=AdministratorAccess-727974078184
aws sso login --profile AdministratorAccess-727974078184
```

### tour-commerce — Go Lambda services

Same pattern as `tour-agent` services:
```bash
cd tour-commerce/services/cart     && make build
cd tour-commerce/services/checkout && make build
cd tour-commerce/services/orders   && make build
cd tour-commerce/services/payments && make build
cd tour-commerce/services/ai-commerce && make build

# Build all services at once
cd tour-commerce && npm run build

# CDK infrastructure
cd tour-commerce/infrastructure && npm install && npx cdk deploy --all
```

### tour-planner-ai — Python/AgentCore

```bash
cd tour-planner-ai/tourplannerai-api

# Install (including toolkit dev extras)
pip install -e '.[dev]'

# Run the orchestrator test suite against the mock API
python -m src.main

# Start AgentCore-compatible local server (GET /ping, POST /invocations)
python -m src.agentcore_local

# Start mock tool API (excursion search, destination context, cruise schedule, etc.)
uvicorn src.mock_api:app --host 127.0.0.1 --port 8001

# AgentCore toolkit dev flow
agentcore dev
agentcore invoke --dev '{"prompt":"Plan a 6-hour Cozumel cruise stop with beach + food"}'
```

Infrastructure:
```bash
cd tour-planner-ai/tourplannerai-infra
npm install && npm run build
npm run cdk synth
npm run cdk:deploy

# Deploy Public API stack with Cognito context params
npm run cdk -- deploy TourPlannerAI-PublicApiStack \
  -c aiRuntimeBaseUrl=http://<host>:<port> \
  -c cognitoUserPoolArn=arn:aws:cognito-idp:... \
  -c requiredOAuthScope=<resource>/<scope> \
  -c allowedVpceIds=vpce-abc123
```

### tour-planner-ui — React/Vite

```bash
cd tour-planner-ui
npm install
npm run dev        # starts on :5173
npm run build      # tsc -b && vite build
npm run lint       # eslint
```

Environment: copy `.env.example` to `.env` and set:
- `VITE_AUTH_BASE_URL` — auth Lambda API Gateway URL
- `VITE_AI_PROXY_URL` — ai-proxy Lambda API Gateway URL
- `VITE_TRIPS_API_URL` — trips Lambda API Gateway URL
- `VITE_ITINERARY_SHARE_API_URL` (default: `http://127.0.0.1:8001/share_itinerary`)
- `VITE_SHARED_TRIP_SYNTHESIS_API_URL` (default: `http://127.0.0.1:8001/shared_trip_synthesis`)
- `VITE_TRIP_INTAKE_API_URL` (default: `http://127.0.0.1:8001/trip_intake_submit`)

### tour-planner-cms — Strapi

```bash
cd tour-planner-cms
npm run develop    # dev mode with auto-reload
npm run build      # build admin panel
npm run start      # production start
```

---

## Architecture

### Service boundaries

```
tour-planner-ui  ──────────────────────────────────────────────────┐
                                                                   │
                 VITE_AUTH_BASE_URL      VITE_AI_PROXY_URL         │
                        │                       │                   │
                        ▼                       ▼                   │
              tour-agent/services/      tour-agent/services/       │ VITE_TRIPS_API_URL
                   auth/                   ai-proxy/               │        │
                                               │                   │        ▼
                                               ▼                   │  tour-agent/services/
                                    tour-planner-ai/               │      trips/
                                    tourplannerai-api/             │
                                    (BedrockAgentCore)             │
                                               │                   │
                                   TOUR_PLANNER_AI_URL             │
                                               ▲                   │
                                               │                   │
                          tour-commerce/services/checkout/ ────────┘
                          tour-commerce/services/ai-commerce/
```

**tour-agent** is the system of record for all transactional state: users, excursion catalogue, inventory, bookings, trips, payments. **tour-planner-ai** must never own transactional state — it calls tour-agent APIs via MCP tools.

**tour-commerce** is the commerce layer: shopping cart, checkout (which calls both tour-agent to create bookings and tour-planner-ai for budget/conflict validation), order lifecycle, and Stripe payments.

### tour-agent database schema (PostgreSQL / Aurora Serverless v2)

Six versioned migrations in `tour-agent-infra/infrastructure/database/migrations/`:
- **V001** — Core schema: users, social profiles, sessions, destinations, categories, suppliers, excursions, schedules, bookings, payments, reviews, inventory, pricing
- **V002** — AI orchestration: ai_runs, webhooks, trip collaboration tables (trip_plans, trip_category_entries, contributor blocks), ownership/RLS scaffolding
- **V003** — Corporate policy engine: 5-tier RBAC (platform_admin → company_admin → travel_manager → dept_manager → traveler), policy rules, approval workflows
- **V004** — System prompts: versioned prompt storage for sanitizer agents

**Trip lifecycle FSM** is data-driven: `trip_status_transitions`, `trip_operation_rules`, and `trip_ui_hints` tables control what operations are allowed in each status (`draft → active → shared → finalized → consumed`). The `trip_plans` table enforces one active trip per user via a unique index (`uq_one_active_trip_per_owner`).

### tour-commerce database schema

Separate Aurora Serverless v2 instance (not shared with tour-agent). Migration in `tour-commerce/infrastructure/database/migrations/V001__commerce_schema.sql`:
- `commerce_carts` / `commerce_cart_items` — shopping cart
- `commerce_checkout_sessions` — short-lived checkout state with AI validation snapshot
- `commerce_orders` / `commerce_order_items` — order records; `tour_agent_booking_id` on each item links back to tour-agent
- `commerce_payments` — Stripe payment/refund tracking
- `commerce_ai_interactions` — audit log of every tour-planner-ai call made during commerce

### tour-planner-ai agent routing

`TourPlannerOrchestrator` classifies prompts and routes to one of nine specialist agents. Context keys take precedence over keyword matching:

| Context key | Routed to |
|---|---|
| `track_budget`, `cost_items`, `budget_items` | BudgetAssistant |
| `summarize_uploaded_itinerary`, `uploaded_itinerary_text` | ItinerarySummaryAssistant |
| `analyze_shared_trip`, `shared_trip_payload`, `trip_intake_payload` | SharedTripSynthesisAssistant |
| `flight_booking_text` | FlightAssistant |
| `hotel_booking_text` | HotelAssistant |
| `cruise_booking_text` | CruiseAssistant |
| keyword-based | Excursion / Restaurant / Sharing |

The mock tool API (port 8001) must be running for the full agent → tool pipeline. Call `POST /invocations` on port 8080 for the AgentCore interface.

### CDK Infrastructure Stacks (tour-agent)

Seven CDK stacks in `tour-agent-infra/infrastructure/lib/stacks/`:

| Stack | What it provisions |
|---|---|
| `NetworkStack` | VPC (3 AZs, 1 NAT GW), Lambda/DB/cache security groups, S3+DynamoDB gateway endpoints |
| `SecurityStack` | KMS customer-managed key, Stripe API key secret, Redis auth token secret |
| `DatabaseStack` | Aurora Serverless v2 PostgreSQL 15.4 (writer + 1 reader), credentials secret, KMS encrypted |
| `CacheStack` | ElastiCache Redis 7.0 replication group (primary + 1 replica, TLS, KMS at-rest) for inventory locking |
| `StorageStack` | S3 vouchers bucket (KMS-encrypted), S3 static-assets bucket + CloudFront distribution (HTTPS, OAI) |
| `MonitoringStack` | CloudWatch log groups for API Gateway, Lambda, Aurora, Redis, X-Ray, EventBridge, SQS |
| `DatabaseSchemaStack` | Custom resource Lambda that runs Flyway migrations on deploy |

### Go Lambda pattern

Every Go Lambda service follows the same structure:
```
services/<name>/
  go.mod              # independent module (github.com/explorer-agent/<name>)
  handler/main.go     # Lambda entrypoint + router + HTTP helpers
  <pkg>/types.go      # domain types
  <pkg>/db.go         # DB helpers (Open() reads DATABASE_URL)
  Makefile            # build / zip / deploy targets
```

`DATABASE_URL` is the only DB configuration used at runtime — it is resolved from Secrets Manager at startup. All handlers extract the caller identity from `req.RequestContext.Authorizer["sub"]` (Cognito JWT sub claim as UUID).

### AI proxy route allowlist

`tour-agent/services/ai-proxy/handler/main.go` is a **strict allowlist-based reverse proxy**. It only forwards the following METHOD+path combinations to `AI_GATEWAY_URL`; everything else returns 404. This is an intentional security boundary — never add a wildcard or catch-all rule.

Allowed routes: `GET /health`, `POST /excursions/search`, `GET /excursions/{id}`, `POST /excursions/availability`, `POST /excursions/quote`, `GET /destination/context`, `GET /cruise/schedule`, `POST /planning/recommend`, `POST /trips/intake`, `POST /trips/share`, `GET /trips/bookings/{id}`, `POST /synthesis/shared-trip`, `OPTIONS *`

### Privacy guardrails and system prompts (tour-planner-ai)

`tour-planner-ai/tourplannerai-api/src/agents/privacy_guard.py` runs **before every model call**:
1. Redacts PII/PCI from user input (emails, phones, cards via Luhn check, CVVs, passports, SSNs, booking refs, addresses).
2. Appends a non-negotiable privacy guardrail block to every system prompt.

System prompts for `hotel_booking_sanitizer`, `flight_booking_sanitizer`, and `cruise_booking_sanitizer` agents are stored in the `system_prompts` DB table (V004 migration) and resolved at runtime via `prompt_store.py` with a 5-minute in-process cache. To update a prompt: insert a new row, then call `activate_system_prompt(id)`.

### UI module structure

`tour-planner-ui/src/` is divided by feature boundary: `auth/`, `intake/`, `trips/`, `excursions/`, `ai/`, `policy/`, `pages/`. Adding a new auth flow requires touching `types.ts`, a new `services/` module, `useAuthController.ts`, a component, and `AuthPage.tsx`. Adding a new intake category requires updating `types.ts`, `categories.ts`, `payload.ts`, and `TripIntakePage.tsx`.

Trip lifecycle UI state is driven entirely from the DB via `GET /trips/{id}/operations` and `GET /trips/ui-hints/{status}` — do not hardcode status logic in frontend components.
