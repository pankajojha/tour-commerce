# Developer Skills Reference

Practical patterns for working in this monorepo. Read `design.md` for architecture context and `CLAUDE.md` for commands.

---

## Go Lambda Services

### Adding a new endpoint to an existing service

1. Add request/response structs at the top of `handler/main.go`.
2. Add a route entry to the `routes` map in `handler()`.
3. Add a case to `matchRoute()` for path-pattern matching.
4. Implement the handler function `handleXxx(ctx, req)`.
5. Extract caller identity with `subFromRequest(req)` (returns `uuid.UUID`).
6. Use `jsonOK(v)` / `jsonError(code, msg)` for consistent responses.

### Adding a new Go Lambda service (tour-agent or tour-commerce)

1. `mkdir tour-agent/services/<name>` and create `go.mod` with module `github.com/explorer-agent/<name>`.
2. Create `handler/main.go` following the router pattern in any existing service.
3. Create `<pkg>/db.go` with `Open()` reading `DATABASE_URL` from env.
4. Copy a `Makefile` from an existing service and update `FUNCTION_NAME`.
5. Add a CDK `NodejsFunction` / `GoFunction` construct in `tour-agent-infra/infrastructure/`.
6. Wire the Lambda to API Gateway with Cognito authorizer.

### Auth sub-handlers

Each subdirectory under `tour-agent/services/auth/` (`login/`, `signup/`, `oauth-start/`, `oauth-callback/`, `passwordless-request/`, `passwordless-verify/`, `pre-signup/`, `token-refresh/`) is an independent Lambda entry point with its own `main.go`. Add a new auth flow by creating a new subdirectory following the same structure.

---

## CDK Infrastructure

### Deploying a stack change

```bash
cd tour-agent-infra/infrastructure
npm --workspace=infrastructure run build   # compile TypeScript
npx cdk diff                               # review changes
npx cdk deploy ExplorerAgent-<StackName>   # deploy one stack
npx cdk deploy --all                       # deploy all stacks
```

### Stack dependency order

`SecurityStack` → `NetworkStack` → `DatabaseStack` + `CacheStack` + `StorageStack` → Lambdas → `DatabaseSchemaStack`

Always deploy security and network stacks before dependent stacks.

### Adding a new S3 bucket

Add a `new s3.Bucket(...)` in `StorageStack`. For sensitive content use `BucketEncryption.KMS` with `props.kmsKey`. Always set `blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL`. Export the bucket ARN via `CfnOutput`.

---

## Database Migrations

Migrations live in `tour-agent-infra/infrastructure/database/migrations/` and are applied by the `DatabaseSchemaStack` custom resource Lambda using Flyway on CDK deploy.

### Adding a migration

1. Create `V00N__description.sql` (next sequential version number).
2. Write idempotent SQL (`CREATE TABLE IF NOT EXISTS`, `CREATE UNIQUE INDEX IF NOT EXISTS`, etc.).
3. For RLS policies, always enable RLS on the table before creating policies.
4. Test locally with the Docker Compose Postgres stack: `local/postgres/docker-compose.yml`.
5. Deploy via CDK — `DatabaseSchemaStack` will run Flyway automatically.

### Local Postgres

```bash
cd tour-agent-infra/local/postgres
docker-compose up -d
# Migrations applied automatically by initdb/10_apply_migrations.sh
```

---

## AI Agents (tour-planner-ai)

### Adding a new specialist agent

1. Create `src/agents/<name>_agent.py` with an `async def handle_<name>_query(prompt, context)` function.
2. Import and register it in `src/agents/orchestrator.py` `__init__` and `run()`.
3. Add context-key overrides and/or keyword patterns to `_classify_intent()`.
4. Add mock tool endpoints to `src/mock_api.py` for local testing.
5. Define tool schemas in `src/agents/tools.py`.

### Privacy guardrail pipeline

Always call `prepare_model_input(raw_text, system_prompt)` before any model invocation. It returns `(redacted_text, safe_system_prompt, report)`. Never pass raw user input directly to a model.

```python
from .privacy_guard import prepare_model_input

redacted, system, report = prepare_model_input(user_text, my_system_prompt)
# pass redacted + system to model
if report.has_redactions():
    log.info("redacted %d emails, %d cards …", report.emails, report.cards)
```

### System prompt lifecycle

1. Edit the template in `prompt_templates/<agent_name>.md`.
2. Publish to DB: `POST /prompt-templates/<agent_name>/publish` (via `template_store.py`).
3. Activate: call `activate_system_prompt('<uuid>')` in PostgreSQL or via the admin API.
4. Cache invalidates automatically after 5 minutes, or call `prompt_store.invalidate('<agent_name>')` in tests.

---

## Policy Engine (tour-agent/services/policy)

### Checking permissions in a handler

```go
import "github.com/explorer-agent/policy/rbac"

ok, err := rbac.HasPermission(ctx, db, callerID, "policy:edit", companyID)
if err != nil || !ok {
    return jsonError(http.StatusForbidden, "insufficient permissions"), nil
}
```

### Role hierarchy (level, lower = more privilege)

| Level | Role |
|---|---|
| 1 | platform_admin |
| 2 | company_admin |
| 3 | travel_manager |
| 4 | dept_manager |
| 5 | traveler |

Privilege escalation is blocked: a caller cannot assign a role with a lower level number than their own.

### Adding a new permission

Add a new permission key to the `role_permissions` table seed data in V003 migration. Pass the key string to `rbac.HasPermission()` in the handler.

---

## Trip Lifecycle

### FSM states

`draft → active → shared → finalized → consumed`

### Extending the FSM

1. Insert rows into `trip_status_transitions` for the new transition.
2. Insert rows into `trip_operation_rules` to control what operations are allowed in the new status.
3. Insert a row into `trip_ui_hints` with the display text for the new status.
4. No code changes needed — the handler reads these tables at runtime.

### Business rules enforced in code (not just DB)

- `trip_end_date` must be set before a trip can leave `draft` (`lifecycle.ValidateTripDates`).
- One active (non-consumed) trip per user; POST /trips returns 409 if one exists.

---

## Commerce Flow (tour-commerce)

### Checkout sequence

```
POST /commerce/checkout       ← cart_id required
  ├── load cart summary
  ├── call tour-planner-ai: budget+conflict validation
  ├── snapshot AI result into commerce_checkout_sessions
  └── return session_id to client

POST /commerce/checkout/{session_id}/confirm
  ├── re-validate session not expired
  ├── create bookings in tour-agent (one per item)
  ├── record order + order_items in commerce DB
  └── initiate Stripe payment
```

### Adding a new checkout validation

Add a new AI tool call in `tour-commerce/services/checkout/checkout/clients.go`, then record the result in the `commerce_ai_interactions` table via the `ai-commerce` service.

---

## Frontend (tour-planner-ui)

### Adding a new page

1. Add route in `src/App.tsx`.
2. Create `src/pages/<Name>Page.tsx` + `<Name>Page.css`.
3. Add nav link in `src/Navigation.tsx` if needed.

### Adding a new API service

1. Create `src/<feature>/service.ts` with typed async functions calling `fetch`.
2. Add types to `src/<feature>/types.ts`.
3. Consume via hook or direct call in the component.

### Environment variables

All env vars are `VITE_` prefixed. Copy `.env.example` to `.env` and fill in API Gateway URLs before running `npm run dev`.

---

## Local Development

### Full local stack (no AWS required)

```bash
# 1. Start Postgres with migrations
cd tour-agent-infra/local/postgres && docker-compose up -d

# 2. Start tour-planner-ai mock API
cd tour-planner-ai/tourplannerai-api
uvicorn src.mock_api:app --host 127.0.0.1 --port 8001

# 3. Start tour-planner-ai local AgentCore server
python -m src.agentcore_local          # listens on :8080

# 4. Start UI dev server
cd tour-planner-ui && npm run dev      # listens on :5173
```

### AWS login (account 727974078184, us-east-1)

```bash
aws sso login --profile AdministratorAccess-727974078184
export AWS_PROFILE=AdministratorAccess-727974078184
export CDK_DEFAULT_ACCOUNT=727974078184
export CDK_DEFAULT_REGION=us-east-1
```

---

## Testing

| Workspace | Command |
|---|---|
| Any Go service | `go test ./...` (run from service root) |
| tour-planner-ai | `python -m src.main` (runs orchestrator against mock API) |
| tour-planner-ui | `npm run lint` (ESLint) |
| CDK | `npm --workspace=infrastructure run build` (TypeScript compile) |
