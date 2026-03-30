.PHONY: build clean test deps

DIST_DIR := dist

build:
	@echo "Building auth handler Lambda..."
	@mkdir -p $(DIST_DIR)
	@cd handler && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go && \
		zip -j ../$(DIST_DIR)/handler.zip bootstrap && \
		rm bootstrap
	@echo "Building pre-signup trigger Lambda..."
	@cd pre-signup && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go && \
		zip -j ../$(DIST_DIR)/pre-signup.zip bootstrap && \
		rm bootstrap
	@echo "Build complete! Artifacts in $(DIST_DIR)/"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(DIST_DIR)
	@echo "Clean complete!"

test:
	@echo "Running tests..."
	@go test ./... -v

deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
