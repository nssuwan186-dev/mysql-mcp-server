# ----------------------------------------
# MySQL MCP Server – Makefile
# ----------------------------------------

APP_NAME = mysql-mcp-server
BIN_DIR = bin
BIN = $(BIN_DIR)/$(APP_NAME)
PKG = ./cmd/mysql-mcp-server

# Colors
YELLOW=\033[1;33m
GREEN=\033[1;32m
BLUE=\033[1;34m
CYAN=\033[1;36m
RED=\033[1;31m
RESET=\033[0m

# Default target
.DEFAULT_GOAL := help

# ----------------------------------------
# Build / Run
# ----------------------------------------

build:
	@echo "$(CYAN)🔨 Building $(APP_NAME)...$(RESET)"
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN) $(PKG)
	@echo "$(GREEN)✔ Build complete: $(BIN)$(RESET)"

run: build
	@echo "$(CYAN)🚀 Running $(APP_NAME)...$(RESET)"
	@$(BIN)

clean:
	@echo "$(YELLOW)🧹 Cleaning project...$(RESET)"
	rm -rf $(BIN_DIR)
	@echo "$(GREEN)✔ Clean complete$(RESET)"

# ----------------------------------------
# Testing
# ----------------------------------------

test:
	@echo "$(BLUE)🧪 Running unit tests...$(RESET)"
	go test ./... -v

integration:
	@echo "$(BLUE)🐋 Running integration tests (Docker required)...$(RESET)"
	go test -tags=integration ./internal/mysql -v

# ----------------------------------------
# Code Quality
# ----------------------------------------

fmt:
	@echo "$(CYAN)🎨 Formatting Go code...$(RESET)"
	go fmt ./...
	@echo "$(GREEN)✔ Code formatted$(RESET)"

lint:
	@echo "$(CYAN)🔍 Running linter...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)⚠ golangci-lint not installed, running go vet instead$(RESET)"; \
		go vet ./...; \
	fi
	@echo "$(GREEN)✔ Lint complete$(RESET)"

# ----------------------------------------
# Dependencies
# ----------------------------------------

deps:
	@echo "$(CYAN)📦 Downloading Go dependencies...$(RESET)"
	go mod tidy
	@echo "$(GREEN)✔ Dependencies updated$(RESET)"

# ----------------------------------------
# Docker Build
# ----------------------------------------

docker:
	@echo "$(CYAN)🐳 Building Docker image '$(APP_NAME)'...$(RESET)"
	docker build -t $(APP_NAME):latest .
	@echo "$(GREEN)✔ Docker image built$(RESET)"

# ----------------------------------------
# Release Build
# ----------------------------------------

release:
	@echo "$(CYAN)📦 Creating production release binary...$(RESET)"
	GOOS=linux GOARCH=amd64 go build -o $(BIN).linux $(PKG)
	GOOS=darwin GOARCH=arm64 go build -o $(BIN).mac $(PKG)
	@echo "$(GREEN)✔ Release artifacts ready in $(BIN_DIR)/$(RESET)"

# ----------------------------------------
# Help
# ----------------------------------------

help:
	@echo ""
	@echo "$(YELLOW)Available Make targets:$(RESET)"
	@echo ""
	@echo "$(CYAN)  make build        $(RESET)- Build the server"
	@echo "$(CYAN)  make run          $(RESET)- Build + run the server"
	@echo "$(CYAN)  make clean        $(RESET)- Remove build artifacts"
	@echo "$(CYAN)  make test         $(RESET)- Run unit tests"
	@echo "$(CYAN)  make integration  $(RESET)- Run integration tests (Docker)"
	@echo "$(CYAN)  make fmt          $(RESET)- Format Go code"
	@echo "$(CYAN)  make lint         $(RESET)- Run linter"
	@echo "$(CYAN)  make deps         $(RESET)- Download and tidy modules"
	@echo "$(CYAN)  make docker       $(RESET)- Build Docker image"
	@echo "$(CYAN)  make release      $(RESET)- Build multi-platform binaries"
	@echo ""
