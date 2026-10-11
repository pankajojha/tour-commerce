module github.com/explorer-agent/commerce-payments

go 1.24.0

toolchain go1.24.2

require (
	github.com/aws/aws-lambda-go v1.47.0
	github.com/google/uuid v1.6.0
	github.com/stripe/stripe-go/v74 v74.30.0
)

require golang.org/x/text v0.29.0 // indirect

replace dbproxy => ../../../tour-agent/services/db-proxy
