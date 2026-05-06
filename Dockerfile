# Stage 1 (Builder)
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o bhejna ./cmd/bhejna

# Stage 2 (Runner)
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

RUN mkdir -p /app/data

WORKDIR /app

COPY --from=builder /app/bhejna .

ENV DB_PATH=/app/data/bhejna.db

EXPOSE 8080

ENTRYPOINT ["/app/bhejna"]
