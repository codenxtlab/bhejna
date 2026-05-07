# Stage 1 (Builder)
FROM golang:1.21-bookworm AS builder

RUN apt-get update && apt-get install -y gcc libsqlite3-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o bhejna ./cmd/bhejna

# Stage 2 (Runner)
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates tzdata sqlite3 && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /app/data

WORKDIR /app

COPY --from=builder /app/bhejna .

ENV DB_PATH=/app/data/bhejna.db

EXPOSE 8080

ENTRYPOINT ["/app/bhejna"]
