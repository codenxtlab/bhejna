# Stage 1: Builder
FROM golang:1.25-bookworm AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO enabled (Required for mattn/go-sqlite3)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o bhejna ./cmd/bhejna

# Stage 2: Runner
FROM debian:bookworm-slim

# Install SSL certs (for n8n webhooks) and tzdata (for timezones)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    tzdata \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*

# Create a non-root user (Force exactly UID 10001)
RUN groupadd -g 10001 appgroup && useradd -u 10001 -g appgroup -s /bin/sh appuser

WORKDIR /app

# Create data dir and set ownership
RUN mkdir -p /app/data && chown -R appuser:appgroup /app

# Bring in the binary from the builder
COPY --from=builder --chown=appuser:appgroup /app/bhejna .

# Run as non-root
USER appuser

ENV DB_PATH=/app/data/bhejna.db

EXPOSE 8080

ENTRYPOINT ["/app/bhejna"]
