module github.com/explorer-agent/commerce-orders

go 1.24.0

toolchain go1.24.2

require (
	github.com/aws/aws-lambda-go v1.47.0
	github.com/google/uuid v1.6.0
)

replace dbproxy => ../../../tour-agent/services/db-proxy
