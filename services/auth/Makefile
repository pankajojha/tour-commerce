.PHONY: build clean deploy test deps

FUNCTIONS := signup login passwordless-request passwordless-verify token-refresh
DIST_DIR := dist

build:
	@echo "Building Lambda functions..."
	@mkdir -p $(DIST_DIR)
	@for func in $(FUNCTIONS); do \
		echo "Building $$func..."; \
		cd $$func && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go && \
		zip -j ../$(DIST_DIR)/$$func.zip bootstrap && \
		rm bootstrap && \
		cd ..; \
	done
	@echo "Build complete! Artifacts in $(DIST_DIR)/"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(DIST_DIR)
	@for func in $(FUNCTIONS); do \
		rm -f $$func/bootstrap; \
	done
	@echo "Clean complete!"

test:
	@echo "Running tests..."
	@go test ./... -v

deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
