# ----------------------------------------
# MySQL MCP Server - Makefile
# ----------------------------------------

APP_NAME := mysql-mcp-server

.PHONY: all test integration build run tidy clean

# Default target
all: test

# --------------------------
# Unit Tests (fast, no Docker)
# --------------------------
test:
	@echo "🧪 Running unit tests..."
	go test ./...

# ----------------------------------------------
# Integration Tests (requires Docker/testcontainers)
# ----------------------------------------------
integration:
	@echo "🐳 Running integration tests (Docker required)..."
	go test -tags=integration ./internal/mysql -v

# --------------------------
# Build binary
# --------------------------
build:
	@echo "🔨 Building $(APP_NAME)..."
	go build -o bin/$(APP_NAME) ./cmd/mysql-mcp-server

# --------------------------
# Run server locally
# --------------------------
run: build
	@echo "🚀 Running $(APP_NAME)..."
	./bin/$(APP_NAME)

# --------------------------
# Go tidy cleanup
# --------------------------
tidy:
	@echo "🧹 Tidying go.mod..."
	go mod tidy

# --------------------------
# Clean artifacts
# --------------------------
clean:
	@echo "🗑️ Cleaning build artifacts..."
	rm -rf bin/
