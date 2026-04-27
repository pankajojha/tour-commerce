# AI Proxy service Makefile
#
# This Lambda forwards all UI-facing AI service calls to the tourplannerai-api
# (the AWS AI runtime). It enforces an explicit route allowlist and passes the
# caller JWT through to the upstream Cognito authorizer.
#
# Environment variables required at runtime:
#   AI_GATEWAY_URL   Base URL of the tourplannerai-api gateway
#                    e.g. https://<id>.execute-api.<region>.amazonaws.com/production

BINARY_NAME   := bootstrap
HANDLER_DIR   := handler
DIST_DIR      := dist
FUNCTION_NAME ?= ai-proxy-handler

.PHONY: all build clean deps test zip deploy

all: build

deps:
	go mod download
	go mod tidy

build: deps
	mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o $(DIST_DIR)/$(BINARY_NAME) ./$(HANDLER_DIR)/...

test:
	go test ./...

zip: build
	cd $(DIST_DIR) && zip -j ai-proxy.zip $(BINARY_NAME)

clean:
	rm -rf $(DIST_DIR)

deploy: zip
	aws lambda update-function-code \
		--function-name $(FUNCTION_NAME) \
		--zip-file fileb://$(DIST_DIR)/ai-proxy.zip \
		--architectures arm64
