# Trips service Makefile
BINARY_NAME   := bootstrap
HANDLER_DIR   := handler
DIST_DIR      := dist
FUNCTION_NAME ?= trips-lifecycle-handler

.PHONY: all build clean deps test zip

all: build

deps:
	go mod download
	go mod tidy

build: deps
	GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o $(DIST_DIR)/$(BINARY_NAME) ./$(HANDLER_DIR)/...

test:
	go test ./...

zip: build
	cd $(DIST_DIR) && zip -j trips.zip $(BINARY_NAME)

clean:
	rm -rf $(DIST_DIR)

deploy: zip
	aws lambda update-function-code \
		--function-name $(FUNCTION_NAME) \
		--zip-file fileb://$(DIST_DIR)/trips.zip \
		--architectures arm64
