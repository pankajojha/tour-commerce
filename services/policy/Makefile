BINARY_NAME   := bootstrap
HANDLER_DIR   := handler
DIST_DIR      := dist
FUNCTION_NAME ?= policy-handler

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
	cd $(DIST_DIR) && zip -j policy.zip $(BINARY_NAME)

clean:
	rm -rf $(DIST_DIR)

deploy: zip
	aws lambda update-function-code \
		--function-name $(FUNCTION_NAME) \
		--zip-file fileb://$(DIST_DIR)/policy.zip \
		--architectures arm64
