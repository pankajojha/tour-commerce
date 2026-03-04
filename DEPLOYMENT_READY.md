# Task 3: Authentication Service - Ready for Deployment

## ✅ Completed Work

### Infrastructure (Terraform)
- ✅ Cognito User Pool module (`terraform/modules/auth/`)
- ✅ Lambda functions module (`terraform/modules/auth-lambdas/`)
- ✅ Main Terraform configuration updated
- ✅ Variables and outputs configured
- ✅ Terraform validated successfully

### Lambda Functions (Go)
- ✅ Signup function
- ✅ Login function
- ✅ Passwordless request function
- ✅ Passwordless verify function
- ✅ Token refresh function
- ✅ All functions built and packaged in `services/auth/dist/`

### Build System
- ✅ Makefile with build, clean, test, deps targets
- ✅ Dist folder for organized artifacts
- ✅ .gitignore updated

## 📋 Pre-Deployment Checklist

### 1. Verify SES Email (Required for passwordless auth)
```bash
aws ses verify-email-identity \
  --email-address noreply@touragent.com \
  --profile AdministratorAccess-727974078184
```

Check verification status:
```bash
aws ses get-identity-verification-attributes \
  --identities noreply@touragent.com \
  --profile AdministratorAccess-727974078184
```

### 2. Review terraform.tfvars (Optional - for social login)
Create `terraform/terraform.tfvars` if you want to enable social login:

```hcl
# Basic configuration
environment = "dev"
from_email  = "noreply@touragent.com"
app_url     = "http://localhost:3000"

# Web callback URLs
web_callback_urls = ["http://localhost:3000/auth/callback"]
web_logout_urls   = ["http://localhost:3000"]

# Enable social login (optional)
enable_social_login = false

# OAuth credentials (only if enable_social_login = true)
# google_client_id     = "your-google-client-id"
# google_client_secret = "your-google-client-secret"
# facebook_app_id      = "your-facebook-app-id"
# facebook_app_secret  = "your-facebook-app-secret"
```

## 🚀 Deployment Steps

### Step 1: Plan the deployment
```bash
cd terraform
terraform plan -out=tfplan
```

Review the plan carefully. You should see:
- 1 Cognito User Pool
- 2 Cognito User Pool Clients (web, mobile)
- 1 Cognito User Pool Domain
- 5 Lambda functions
- 5 CloudWatch Log Groups
- IAM roles and policies
- Secrets Manager entries

### Step 2: Apply the changes
```bash
terraform apply tfplan
```

This will create:
- Cognito User Pool: `explorer-agent-user-pool`
- Lambda functions:
  - `explorer-agent-auth-signup`
  - `explorer-agent-auth-login`
  - `explorer-agent-auth-passwordless-request`
  - `explorer-agent-auth-passwordless-verify`
  - `explorer-agent-auth-token-refresh`

### Step 3: Save the outputs
```bash
terraform output > auth_outputs.txt
```

Important outputs:
- `user_pool_id` - For frontend configuration
- `web_client_id` - For frontend configuration
- `user_pool_domain` - For OAuth flows
- Lambda ARNs - For API Gateway integration

## 🧪 Post-Deployment Testing

### 1. Test Signup via AWS CLI
```bash
# Get the client ID from terraform output
CLIENT_ID=$(terraform output -raw web_client_id)

# Test signup
aws cognito-idp sign-up \
  --client-id $CLIENT_ID \
  --username test@example.com \
  --password "TestPass123!" \
  --user-attributes Name=email,Value=test@example.com \
  --profile AdministratorAccess-727974078184
```

### 2. Verify Email
Check your email for the verification code, then:
```bash
aws cognito-idp confirm-sign-up \
  --client-id $CLIENT_ID \
  --username test@example.com \
  --confirmation-code YOUR_CODE \
  --profile AdministratorAccess-727974078184
```

### 3. Test Login
```bash
aws cognito-idp initiate-auth \
  --client-id $CLIENT_ID \
  --auth-flow USER_PASSWORD_AUTH \
  --auth-parameters USERNAME=test@example.com,PASSWORD=TestPass123! \
  --profile AdministratorAccess-727974078184
```

### 4. Test Lambda Functions Directly
```bash
# Test signup Lambda
aws lambda invoke \
  --function-name explorer-agent-auth-signup \
  --payload '{"body":"{\"email\":\"test2@example.com\",\"password\":\"Test123!\"}"}' \
  --profile AdministratorAccess-727974078184 \
  response.json

cat response.json
```

## 📝 Next Steps (Task 3.4 - Manual Validation)

After deployment, you'll need to:

1. **Create API Gateway** to expose Lambda functions as REST endpoints
2. **Test with Postman** using the manual validation checklist in tasks.md
3. **Verify passwordless flow** works end-to-end
4. **Test rate limiting** on passwordless requests
5. **Verify database** token storage and cleanup

## 🔧 Troubleshooting

### Lambda can't connect to database
- Check VPC configuration
- Verify security groups allow Lambda → RDS traffic
- Check database credentials in Secrets Manager

### SES emails not sending
- Verify email address in SES console
- Check if account is in sandbox mode (can only send to verified addresses)
- Request production access if needed

### Cognito errors
- Check CloudWatch logs: `/aws/lambda/explorer-agent-auth-*`
- Verify user pool configuration
- Check client ID and user pool ID match

### Terraform errors
- Run `terraform init -upgrade` if modules changed
- Check AWS credentials: `aws sts get-caller-identity`
- Verify all required variables are set

## 📊 Monitoring

After deployment, monitor:
- CloudWatch Logs: `/aws/lambda/explorer-agent-auth-*`
- Cognito User Pool metrics in AWS Console
- Lambda function metrics (invocations, errors, duration)
- Database connections from Lambda

## 🔐 Security Notes

- All passwords are hashed by Cognito (not stored in our database)
- Passwordless tokens are bcrypt-hashed before storage
- JWT tokens expire after 60 minutes
- Refresh tokens expire after 30 days
- Rate limiting prevents brute force attacks
- Advanced security mode enabled in Cognito

## 💰 Cost Estimate

Approximate monthly costs (dev environment, low usage):
- Cognito: $0 (first 50,000 MAUs free)
- Lambda: ~$0.20 (1M requests free tier)
- CloudWatch Logs: ~$0.50
- Secrets Manager: ~$0.40 per secret
- **Total: ~$2-5/month for dev**

Production costs will scale with usage.

## 📚 Documentation

- Lambda functions: `services/auth/README.md`
- Task progress: `TASK3_PROGRESS.md`
- Terraform modules: `terraform/modules/auth/` and `terraform/modules/auth-lambdas/`

## ✅ Ready to Deploy!

All code is complete and validated. Run the deployment steps above when ready.
