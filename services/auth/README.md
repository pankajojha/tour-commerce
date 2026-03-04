# Authentication Service

Go-based Lambda functions for user authentication using AWS Cognito.

## Functions

### 1. Signup (`signup/`)
- Registers new users with email/password
- Validates email format and password strength
- Sends email verification code via Cognito
- **Endpoint**: `POST /auth/signup`
- **Request**:
  ```json
  {
    "email": "user@example.com",
    "password": "SecurePass123!",
    "name": "John Doe",
    "phone": "+1234567890"
  }
  ```

### 2. Login (`login/`)
- Authenticates users with email/password
- Returns JWT tokens (access, ID, refresh)
- **Endpoint**: `POST /auth/login`
- **Request**:
  ```json
  {
    "email": "user@example.com",
    "password": "SecurePass123!"
  }
  ```

### 3. Passwordless Request (`passwordless-request/`)
- Generates magic link or 6-digit code
- Stores hashed token in database with 10-minute expiration
- Sends email via SES
- Rate limiting: 3 attempts per 15 minutes
- **Endpoint**: `POST /auth/passwordless/request`
- **Request**:
  ```json
  {
    "email": "user@example.com",
    "type": "magic_link"
  }
  ```

### 4. Passwordless Verify (`passwordless-verify/`)
- Verifies magic link or code
- Returns JWT tokens on success
- Max 3 verification attempts per token
- **Endpoint**: `POST /auth/passwordless/verify`
- **Request**:
  ```json
  {
    "tokenId": "uuid-here",
    "token": "abc123..."
  }
  ```

### 5. Token Refresh (`token-refresh/`)
- Refreshes access and ID tokens using refresh token
- **Endpoint**: `POST /auth/token/refresh`
- **Request**:
  ```json
  {
    "refreshToken": "refresh-token-here"
  }
  ```

## Building

```bash
# Install dependencies
make deps

# Build all Lambda functions
make build

# Clean build artifacts
make clean

# Run tests
make test
```

## Deployment

1. Build the Lambda functions:
   ```bash
   cd services/auth
   make build
   ```

2. Deploy with Terraform:
   ```bash
   cd terraform
   terraform apply
   ```

## Environment Variables

Each Lambda function requires specific environment variables:

- `USER_POOL_ID`: Cognito User Pool ID
- `CLIENT_ID`: Cognito App Client ID
- `DB_HOST`: Aurora PostgreSQL endpoint
- `DB_NAME`: Database name
- `DB_USER`: Database username
- `DB_PASSWORD`: Database password
- `FROM_EMAIL`: SES verified sender email
- `APP_URL`: Application URL for magic links

## Database Schema

The passwordless authentication requires the `passwordless_tokens` table:

```sql
CREATE TABLE passwordless_tokens (
    token_id UUID PRIMARY KEY,
    user_email VARCHAR(255) NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    token_type VARCHAR(20) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    attempts INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_passwordless_tokens_email ON passwordless_tokens(user_email);
CREATE INDEX idx_passwordless_tokens_expires ON passwordless_tokens(expires_at);
```

## Testing

### Manual Testing with AWS CLI

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
```

### Testing with Postman

Import the provided Postman collection (coming soon) for comprehensive API testing.

## Security Features

- Password policy: 8+ characters, uppercase, lowercase, numbers, symbols
- Email verification required
- Rate limiting on passwordless authentication
- Bcrypt hashing for tokens
- JWT tokens with 60-minute expiration
- Refresh tokens with 30-day expiration
- Advanced security mode enabled in Cognito

## Error Handling

All functions return standardized error responses:

```json
{
  "error": "ErrorType",
  "message": "Human-readable error message"
}
```

Common HTTP status codes:
- `200`: Success
- `201`: Created (signup)
- `400`: Bad request (validation error)
- `401`: Unauthorized (invalid credentials)
- `403`: Forbidden (email not verified)
- `404`: Not found (invalid token)
- `409`: Conflict (user already exists)
- `429`: Too many requests (rate limited)
- `500`: Internal server error
