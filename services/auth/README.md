# Auth Service

Single Lambda serving all authentication endpoints via API Gateway.

## Architecture

**User Pool First** — Cognito User Pool is the central identity authority. Social providers (Google, Facebook, Apple) federate into the pool. A pre-signup trigger auto-links social accounts to existing native accounts by email.

## Structure

```
auth/
├── handler/main.go      # Single Lambda — all /auth/* API routes
├── pre-signup/main.go   # Cognito pre-signup trigger (account linking)
├── go.mod
├── Makefile
└── dist/                # Build artifacts (gitignored)
```

## Endpoints

All served by `handler/main.go`, routed by API Gateway resource path:

| Method | Path                         | Auth     | Description              |
|--------|------------------------------|----------|--------------------------|
| POST   | /auth/signup                 | None     | Register with email/pass |
| POST   | /auth/login                  | None     | Email/password login     |
| POST   | /auth/passwordless/request   | None     | Request magic link/code  |
| POST   | /auth/passwordless/verify    | None     | Verify token, get JWT    |
| POST   | /auth/refresh                | None     | Refresh access token     |

## Environment Variables

| Variable       | Required | Used by          | Description                    |
|----------------|----------|------------------|--------------------------------|
| USER_POOL_ID   | Yes      | handler          | Cognito User Pool ID           |
| CLIENT_ID      | Yes      | handler          | Cognito App Client ID          |
| DB_HOST        | No*      | handler          | PostgreSQL host (passwordless) |
| DB_NAME        | No*      | handler          | Database name                  |
| DB_USER        | No*      | handler          | Database user                  |
| DB_PASSWORD    | No*      | handler          | Database password              |
| FROM_EMAIL     | No       | handler          | SES sender address             |
| APP_URL        | No       | handler          | App URL for magic links        |

*Required only for passwordless flows.

## Build

```bash
make build    # Produces dist/handler.zip + dist/pre-signup.zip
make clean    # Remove dist/
make deps     # go mod tidy
make test     # go test ./...
```

## Infrastructure

Terraform modules in `tour-agent/terraform/modules/`:
- `auth/` — Cognito User Pool, social IdPs, pre-signup trigger Lambda
- `auth-lambdas/` — Auth handler Lambda, IAM roles
- `api-gateway/` — REST API with routes pointing to the single handler
