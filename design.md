# System Design: Tour Excursion Booking Platform

## Overview

A cruise shore-excursion booking platform built on AWS serverless infrastructure. Five independently deployable workspace boundaries: **tour-agent** (transactional backend), **tour-planner-ai** (AI orchestration), **tour-planner-ui** (React SPA), **tour-commerce** (commerce layer), **tour-planner-cms** (Strapi headless CMS).

---

## AWS Infrastructure Topology

```
Internet
   │
   ▼
CloudFront (static assets CDN)  ←─── S3: explorer-agent-assets-{account}
                                       S3: explorer-agent-vouchers-{account} (KMS)
   │
API Gateway (REST) ─── Cognito JWT Authorizer
   │
   ├── Lambda: auth/* (8 sub-handlers in tour-agent/services/auth/)
   ├── Lambda: ai-proxy  ──────────────────────────────────────────┐
   ├── Lambda: trips                                               │
   ├── Lambda: policy                                              ▼
   │                                              API Gateway (private VPCE)
   └── tour-commerce services:                         │
       ├── Lambda: cart               tour-planner-ai/tourplannerai-api
       ├── Lambda: checkout ──────────    (Bedrock AgentCore Runtime)
       ├── Lambda: orders                               │
       ├── Lambda: payments                     AWS Bedrock (Claude)
       └── Lambda: ai-commerce

All Lambdas run inside VPC (private-with-egress subnets)
   │
   ├── Aurora Serverless v2 PostgreSQL 15.4 (isolated subnets)
   │     ├── tour-agent DB:    exploreragent
   │     └── tour-commerce DB: separate cluster
   │
   └── ElastiCache Redis 7.0 (isolated subnets, TLS + KMS)
         └── inventory locking

Secrets Manager: DB credentials, Stripe API key, Redis auth token
KMS: customer-managed key for Aurora, Redis, vouchers S3, Secrets
CloudWatch: log groups for API GW, Lambda, Aurora, Redis, X-Ray
```

**VPC**: 10.0.0.0/16, 3 AZs, 1 NAT Gateway, gateway endpoints for S3 and DynamoDB.

---

## Service Boundaries

### tour-agent (Go 1.21, AWS Lambda)

System of record for all transactional state. Never let tour-planner-ai write directly to this DB.

| Sub-service | Endpoints |
|---|---|
| `auth/login` | `POST /auth/login` |
| `auth/signup` | `POST /auth/signup` |
| `auth/oauth-start` | `GET /auth/oauth/start` |
| `auth/oauth-callback` | `GET /auth/oauth/callback` |
| `auth/passwordless-request` | `POST /auth/passwordless/request` |
| `auth/passwordless-verify` | `POST /auth/passwordless/verify` |
| `auth/pre-signup` | Cognito pre-signup Lambda trigger |
| `auth/token-refresh` | `POST /auth/token/refresh` |
| `trips` | `POST /trips`, `GET /trips/owner/active`, `GET /trips/{id}/operations`, `POST /trips/{id}/transition`, `PATCH /trips/{id}`, `GET /trips/ui-hints/{status}` |
| `policy` | `POST /policy/evaluate`, `GET /policy/limits`, policy CRUD, rule management, override workflow, compliance report, role assignment |
| `ai-proxy` | Strict allowlist reverse proxy → `AI_GATEWAY_URL` |

### tour-planner-ai (Python 3.11, Bedrock AgentCore)

Stateless AI orchestration. Reads trip/excursion data from tour-agent via MCP tools. Never writes transactional state directly.

**Entry point**: `POST /invocations` (AgentCore interface, port 8080)

**Routing**: `TourPlannerOrchestrator` classifies prompts → 9 specialist agents via context-key overrides first, then keyword matching.

| Agent | Routed when |
|---|---|
| ExcursionAssistant | keywords: excursion, shore tour, beach, activity |
| FlightAssistant | context: `flight_booking_text` / `sanitize_flight_booking` |
| HotelAssistant | context: `hotel_booking_text` / `sanitize_hotel_booking` |
| CruiseAssistant | context: `cruise_booking_text` / `sanitize_cruise_booking` |
| BudgetAssistant | context: `track_budget`, `cost_items`, `budget_limit` |
| ItinerarySummaryAssistant | context: `summarize_uploaded_itinerary`, `uploaded_itinerary_text` |
| SharedTripSynthesisAssistant | context: `analyze_shared_trip`, `shared_trip_payload`, `trip_intake_payload` |
| RestaurantAssistant | keywords: restaurant, dining, food, cuisine |
| SharingAssistant | keywords: save, share, draft, preferences |

**Privacy pipeline** (mandatory, runs before every model call):
1. `privacy_guard.redact_sensitive_text()` — strips emails, phones, cards (Luhn-validated), CVV, passports, SSNs, booking refs, addresses.
2. `privacy_guard.enforce_prompt_privacy_guardrails()` — appends NON-NEGOTIABLE PRIVACY GUARDRAILS block to system prompt.

**System prompts**: stored in `system_prompts` table (V004), resolved at runtime via `prompt_store.resolve_active_prompt()` with 5-minute in-process cache. Activate a new version with `activate_system_prompt(id)`.

### tour-commerce (Go 1.21, AWS Lambda)

| Sub-service | Endpoints |
|---|---|
| `cart` | Cart CRUD |
| `checkout` | `POST /commerce/checkout` (initiates AI budget+conflict check), `GET /commerce/checkout/{session_id}`, `POST /commerce/checkout/{session_id}/confirm` |
| `orders` | Order lifecycle |
| `payments` | Stripe payment / refund tracking |
| `ai-commerce` | AI interaction audit log |

Checkout flow: create session → call tour-planner-ai for budget/conflict validation → return session with AI snapshot → client confirms → bookings created in tour-agent.

### tour-planner-ui (React 19 + Vite)

SPA with feature-boundary module structure: `auth/`, `intake/`, `trips/`, `excursions/`, `ai/`, `policy/`, `pages/`, `components/`.

Trip lifecycle UI state comes from `GET /trips/{id}/operations` and `GET /trips/ui-hints/{status}` — no hardcoded status logic in components.

### tour-planner-cms (Strapi)

Headless CMS for content management. Runs on Node.js with PostgreSQL backend (separate from tour-agent DB).

---

## Database Schema (tour-agent)

Four Flyway migrations applied by `DatabaseSchemaStack` on deploy:

| Migration | Content |
|---|---|
| V001 | Core: `AppUsers`, `social_profiles`, `sessions`, `destinations`, `categories`, `suppliers`, `excursions`, `schedules`, `bookings`, `payments`, `reviews`, `inventory`, `pricing` |
| V002 | AI: `ai_runs`, `webhooks`, `trip_plans`, `trip_category_entries`, contributor blocks, `trip_status_transitions`, `trip_operation_rules`, `trip_ui_hints` |
| V003 | Policy: 5-tier RBAC (`roles`, `user_roles`), `company_policies`, `policy_rules`, `policy_evaluations`, `policy_overrides`, approval workflow |
| V004 | Prompts: `system_prompts`, `system_prompt_activations`, `activate_system_prompt()` function, `active_system_prompts` view, RLS |

**Trip lifecycle FSM**: `draft → active → shared → finalized → consumed`. Transitions are data-driven from `trip_status_transitions`. One active trip per user enforced by `uq_one_active_trip_per_owner` unique index.

**RBAC tiers** (lowest level number = highest privilege): `platform_admin (1)` → `company_admin (2)` → `travel_manager (3)` → `dept_manager (4)` → `traveler (5)`.

---

## Database Schema (tour-commerce)

Separate Aurora cluster. One migration `V001__commerce_schema.sql`:
- `commerce_carts` / `commerce_cart_items`
- `commerce_checkout_sessions` (short-lived, includes AI validation snapshot)
- `commerce_orders` / `commerce_order_items` (link back to tour-agent via `tour_agent_booking_id`)
- `commerce_payments` (Stripe payment/refund tracking)
- `commerce_ai_interactions` (audit log of every AI call during commerce)

---

## Authentication Flow

```
UI → POST /auth/login  (email+password or social OAuth)
  ├── Social: /auth/oauth/start → provider → /auth/oauth/callback → Cognito
  └── Passwordless: /auth/passwordless/request (send email OTP) → /auth/passwordless/verify
       └── Cognito User Pools issues JWT (ID + Access + Refresh tokens)

All Lambda routes read caller identity from:
  req.RequestContext.Authorizer["sub"]  ← Cognito JWT sub claim (UUID)
```

Supported providers: Email, Google, Facebook, Instagram, TikTok.

---

## Security Model

| Layer | Mechanism |
|---|---|
| Network | VPC private/isolated subnets, security groups (Lambda → DB port 5432, Lambda → Redis port 6379) |
| Encryption at rest | KMS CMK for Aurora, Redis, vouchers S3 |
| Encryption in transit | TLS on all services; Redis `transitEncryptionEnabled: true` |
| Secrets | Secrets Manager: DB creds, Stripe key, Redis auth token |
| Auth | Cognito JWT on all API Gateway routes |
| RBAC | 5-tier role hierarchy in DB; privilege escalation blocked at the Go layer |
| AI input | Regex PII/PCI redaction before every model call |
| AI output | Privacy guardrail system prompt appended to all agent prompts |
| CDN | CloudFront OAI for S3; HTTPS-only; TLS 1.2+ |

---

## Environment Variables (runtime)

| Variable | Consumer | Value |
|---|---|---|
| `DATABASE_URL` | All Go Lambda services | Resolved from Secrets Manager at init |
| `AI_GATEWAY_URL` | tour-agent/ai-proxy | Base URL of tourplannerai-api API Gateway |
| `TOUR_PLANNER_AI_URL` | tour-commerce/checkout | tour-planner-ai endpoint for AI validation |
| `ENABLE_GUARDRAILS` | tour-planner-ai | `true` (default) |
| `VITE_AUTH_BASE_URL` | tour-planner-ui | Auth Lambda API Gateway URL |
| `VITE_AI_PROXY_URL` | tour-planner-ui | ai-proxy Lambda API Gateway URL |
| `VITE_TRIPS_API_URL` | tour-planner-ui | trips Lambda API Gateway URL |

---

## Key Design Invariants

1. **tour-planner-ai never owns transactional state** — it reads via MCP tools, writes go through tour-agent APIs.
2. **One active trip per user** — enforced by DB unique index and application-layer 409 conflict response.
3. **Privacy first** — all text goes through `prepare_model_input()` before model invocation; no PII/PCI reaches the model.
4. **Strict AI proxy allowlist** — ai-proxy only forwards explicitly listed routes; no wildcard forwarding.
5. **Trip lifecycle is data-driven** — `trip_status_transitions` and `trip_operation_rules` tables own FSM logic; UI reads `operations` and `ui-hints` endpoints, never hardcodes state.
6. **Privilege escalation is blocked** — policy service checks caller role level before allowing role assignment.
