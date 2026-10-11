# Go URL Shortener (100M-Systems)

A high-performance URL shortener REST API built with Go, PostgreSQL, and Redis.

## Features

- **Create short URLs:** Fast 8-character base64 URL-safe collision-handled code generation
- **Redirects:** Ultra-fast HTTP 302 redirection to original URLs
- **URL Metadata API:** Query URL creation and metadata by short code
- **Multi-Engine Storage:**
  - `memory`: Local in-memory store for testing and development
  - `postgres`: Persistent relational storage using PostgreSQL
  - `redis`: High-speed key-value storage using Redis
  - `cached`: PostgreSQL persistent source-of-truth with Redis cache-aside layer
- **Structured Logging:** Go `log/slog` structured JSON logging
- **Middleware:** Request logging and panic recovery
- **Graceful Shutdown:** SIGINT / SIGTERM signal handling with timeout

## Storage Modes

Configure via `STORAGE_MODE` in `.env`:
- `STORAGE_MODE=postgres`: Uses PostgreSQL (automatically utilizes Redis cache if `REDIS_URL` is available)
- `STORAGE_MODE=cached`: Explicit PostgreSQL + Redis cache-aside mode
- `STORAGE_MODE=redis`: Redis-only storage
- `STORAGE_MODE=memory`: In-memory map storage

## Running Locally

1. Set your database credentials in `.env` (pointing to your server, e.g. `192.168.0.113`):
   ```bash
   cp .env.example .env
   ```
2. Start the server directly:
   ```bash
   go run ./CMD/server
   ```

## Running with Docker

Since PostgreSQL and Redis run on your server (`192.168.0.113`), build and run just the application container:

```bash
docker compose up -d --build
```

To stop:
```bash
docker compose down
```

## API Endpoints

- **Shorten URL:**
  ```http
  POST /api/shorten
  Content-Type: application/json

  {
    "url": "https://example.com/very/long/url"
  }
  ```
- **Redirect:**
  ```http
  GET /{code}
  ```
- **Inspect URL:**
  ```http
  GET /api/urls/{code}
  ```