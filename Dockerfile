# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /mysql-mcp-server ./cmd/mysql-mcp-server

# Final stage - minimal image
FROM scratch

# Copy CA certificates for HTTPS connections
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /mysql-mcp-server /mysql-mcp-server

# Environment variables (can be overridden)
ENV MYSQL_DSN=""
ENV MYSQL_MAX_ROWS="200"
ENV MYSQL_QUERY_TIMEOUT_SECONDS="30"
ENV MYSQL_MCP_EXTENDED="0"
ENV MYSQL_MCP_JSON_LOGS="0"

# The MCP server uses stdio, so no ports to expose
# Run the server
ENTRYPOINT ["/mysql-mcp-server"]

