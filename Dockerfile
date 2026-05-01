# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies for CGO/SQLite
RUN apk add --no-cache git gcc musl-dev

# Copy fluxcore module (handle local replace directive)
# NOTE: When building, provide fluxcore via build-context: --build-context fluxcore-build=../fluxcore
COPY fluxcore-build ../fluxcore

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o tokrouter ./cmd/tokrouter

# Stage 2: Runtime
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk --no-cache add ca-certificates wget

# Copy binary from builder
COPY --from=builder /app/tokrouter .

# Create data directory
RUN mkdir -p /app/data

# Expose default port
EXPOSE 8765

# Healthcheck
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
  CMD wget -q -O- http://localhost:8765/health > /dev/null 2>&1 || exit 1

# Run tokrouter
CMD ["./tokrouter", "start"]