# Multi-stage build for Go URL Shortener
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build lightweight statically-linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./CMD/server

# Final minimal runtime image
FROM alpine:3.20

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/server /app/server
COPY --from=builder /app/schema.sql /app/schema.sql

EXPOSE 8080

ENTRYPOINT ["/app/server"]

