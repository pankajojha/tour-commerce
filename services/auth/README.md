# Auth Service

Single Lambda serving all authentication endpoints via API Gateway.

## Architecture

**User Pool First** — Cognito User Pool is the central identity authority. Social providers (Google, Facebook, Apple) federate into the pool. A pre-signup trigger auto-links social accounts to existing native accounts by email.

## Structure

## Service Layout

Single handler and pre-signup trigger are deployed as Lambda artifacts:

```
auth/
├── handler/main.go      # Single Lambda — all /auth/* API routes
├── pre-signup/main.go   # Cognito pre-signup trigger (account linking)
├── go.mod
├── Makefile
└── dist/                # Build artifacts (gitignored)
```

Auth endpoints are implemented with modular code under feature folders.

- `main.go`: Lambda entrypoint only.
- `handler.go`: request parsing and main control flow.
- `config.go`: SDK clients, env loading, and startup validation.
- `types.go`: request/response structs.
- `responses.go`: shared success/error HTTP response helpers.
- Additional files by concern (for example `errors.go`, `store.go`, `email.go`, `tokens.go`, `helpers.go`, `providers.go`, `url.go`, `state.go`).

Current folders:

- `signup/`
- `login/`
- `passwordless-request/`
- `passwordless-verify/`
- `token-refresh/`
- `oauth-start/`
- `oauth-callback/`

Feature behavior highlights:

### Passwordless Request (`passwordless-request/`)
- Generates magic link or 6-digit code
- Stores hashed token in database with 10-minute expiration
- Sends email via SES
- Rate limiting: 3 attempts per 15 minutes
- Endpoint: `POST /auth/passwordless/request`

### Passwordless Verify (`passwordless-verify/`)
- Verifies magic link or code
- Returns JWT tokens on success
- Max 3 verification attempts per token
- Endpoint: `POST /auth/passwordless/verify`

### Token Refresh (`token-refresh/`)
- Refreshes access and ID tokens using refresh token
- Endpoint: `POST /auth/token/refresh`

### OAuth Start (`oauth-start/`)
- Redirects to Cognito Hosted UI authorize endpoint
- Providers: `google`, `facebook`, `microsoft`, `linkedin`
- Endpoint: `GET /auth/oauth/{provider}`

### OAuth Callback (`oauth-callback/`)
- Handles Cognito/provider callback
- Redirects back to frontend callback/cancel URL
- Endpoint: `GET /auth/oauth/callback`

## Endpoints

All served by `handler/main.go`, routed by API Gateway resource path:

| Method | Path                         | Auth     | Description              |
|--------|------------------------------|----------|--------------------------|
| POST   | /auth/signup                 | None     | Register with email/pass |
| POST   | /auth/login                  | None     | Email/password login     |
| POST   | /auth/passwordless/request   | None     | Request magic link/code  |
| POST   | /auth/passwordless/verify    | None     | Verify token, get JWT    |
| POST   | /auth/token/refresh          | None     | Refresh access token     |

## Environment Variables

Environment variables by function:

- `signup`: `USER_POOL_ID`, `CLIENT_ID`
- `login`: `CLIENT_ID`
- `passwordless-request`: `DB_HOST`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `FROM_EMAIL`, `APP_URL`
- `passwordless-verify`: `CLIENT_ID`, `DB_HOST`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`
- `token-refresh`: `CLIENT_ID`
- `oauth-start`: `CLIENT_ID`, `COGNITO_DOMAIN`, `OAUTH_CALLBACK_URL`, optional `OAUTH_SCOPE`
- `oauth-callback`: `FRONTEND_AUTH_CALLBACK_URL`, `FRONTEND_AUTH_CANCEL_URL`

Variable reference:

- `USER_POOL_ID`: Cognito User Pool ID
- `CLIENT_ID`: Cognito App Client ID
- `DB_HOST`: Aurora PostgreSQL endpoint
- `DB_NAME`: Database name
- `DB_USER`: Database username
- `DB_PASSWORD`: Database password
- `FROM_EMAIL`: SES verified sender email
- `APP_URL`: Base app URL used to construct magic links
- `COGNITO_DOMAIN`: Cognito hosted UI domain used by OAuth start
- `OAUTH_CALLBACK_URL`: Backend callback URL registered in Cognito app client
- `OAUTH_SCOPE`: Optional OAuth scope override (defaults to `openid email profile`)
- `FRONTEND_AUTH_CALLBACK_URL`: Frontend success return URL
- `FRONTEND_AUTH_CANCEL_URL`: Frontend cancel/error return URL

## Build

```bash
# Test signup
aws lambda invoke \
  --function-name explorer-agent-auth-signup \
  --payload '{"body":"{\"email\":\"test@example.com\",\"password\":\"Test123!\"}"}' \
  response.json

# Test login
aws lambda invoke \
  --function-name explorer-agent-auth-login \
  --payload '{"body":"{\"email\":\"test@example.com\",\"password\":\"Test123!\"}"}' \
  response.json

# Test OAuth start (google)
aws lambda invoke \
  --function-name explorer-agent-auth-oauth-start \
  --payload '{"httpMethod":"GET","pathParameters":{"provider":"google"}}' \
  response.json

# Test OAuth callback success redirect
aws lambda invoke \
  --function-name explorer-agent-auth-oauth-callback \
  --payload '{"httpMethod":"GET","queryStringParameters":{"code":"sample-code","state":"eyJub25jZSI6IngifQ"}}' \
  response.json
```

Or use Makefile shortcuts:

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
