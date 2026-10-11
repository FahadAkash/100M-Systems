# Go URL Shortener (100M-Systems)

A high-performance URL shortener REST API built with Go, PostgreSQL, Redis, and React.

## Features

- **Create short URLs:** Fast 8-character base64 URL-safe collision-handled code generation
- **Redirects:** Ultra-fast HTTP 302 redirection to original URLs
- **URL Metadata & Analytics API:** Query URL creation metadata and real-time click streaming analytics by short code
- **Multi-Engine Storage:**
  - `memory`: Local in-memory store for testing and development
  - `postgres`: Persistent relational storage using PostgreSQL
  - `redis`: High-speed key-value storage using Redis
  - `cached`: PostgreSQL persistent source-of-truth with Redis cache-aside layer
- **Apache Kafka Streaming Analytics:** Non-blocking async event producer and micro-batching background consumer for high-throughput click tracking
- **Distributed Rate Limiting:** Redis-backed sliding window rate limiter with in-memory fallback
- **Prometheus Metrics:** Native `/metrics` endpoint measuring traffic, latency, rate limits, Kafka pipelines, and cache stats
- **Structured Logging:** Go `log/slog` structured JSON logging
- **CORS Enabled:** Seamless integration with React frontend
- **Graceful Shutdown:** SIGINT / SIGTERM signal handling with timeout

## Apache Kafka Event Streaming Pipeline

When a user accesses a short URL (`GET /{code}`):
1. **Zero-Latency Hot Path:** The HTTP redirect is resolved and returns `302 Found` within sub-milliseconds.
2. **Asynchronous Non-Blocking Producer:** The redirect handler publishes a `ClickEvent` (containing short code, target URL, IP address, user-agent, referer, and timestamp) into an in-memory buffered channel. The request is never delayed or blocked even under heavy loads or Kafka broker outages.
3. **Micro-Batch Consumer:** A background Kafka consumer group reads the stream and flushes events in transactions to PostgreSQL (`url_clicks` table) every 1 second or 100 messages, minimizing database write overhead.
4. **Prometheus Telemetry:** Monitored with `kafka_events_produced_total` and `kafka_events_consumed_total`.

### Configuration in `.env`:
```env
KAFKA_ENABLED=false                     # Toggle Kafka streaming on/off
KAFKA_BROKERS=192.168.0.113:9092        # Kafka broker address list (comma-separated)
KAFKA_TOPIC=url-clicks                  # Target topic name
KAFKA_GROUP_ID=url-shortener-analytics  # Consumer group ID
```

## Rate Limiting

Rate limiting protects endpoints against abuse and traffic spikes:
- **Distributed Mode:** When Redis is available, tracks request windows per IP using Redis atomic pipelines.
- **In-Memory Mode:** Gracefully falls back to per-IP token bucket limiters (`golang.org/x/time/rate`).
- **HTTP Response Headers:**
  - `X-RateLimit-Limit`: Maximum requests permitted per window.
  - `X-RateLimit-Remaining`: Requests remaining in the current window.
  - `X-RateLimit-Reset`: Unix timestamp when the current window resets.
  - `Retry-After`: Seconds to wait before retrying (on `HTTP 429 Too Many Requests`).
- **Configurable via `.env`:**
  - `RATE_LIMIT_ENABLED=true`
  - `RATE_LIMIT_MAX_REQUESTS=100` (e.g., 100 requests)
  - `RATE_LIMIT_WINDOW_SECONDS=60` (e.g., per 60 seconds)

## Prometheus Analytics Metrics

The service exports standard and custom metrics on `GET /metrics`:

| Metric Name | Type | Labels | Description |
| :--- | :--- | :--- | :--- |
| `http_requests_total` | Counter | `method`, `path`, `status` | Total HTTP requests by route and status code |
| `http_request_duration_seconds` | Histogram | `method`, `path`, `status` | Request latency distribution in seconds |
| `http_requests_in_flight` | Gauge | - | Active concurrent requests being processed |
| `http_rate_limited_total` | Counter | `path` | Total requests rejected by rate limiting (429) |
| `url_shortener_urls_created_total` | Counter | - | Total short URLs generated |
| `url_shortener_redirects_total` | Counter | `status` (`found`/`not_found`) | Total short code redirects executed |
| `url_shortener_cache_hits_total` | Counter | - | Redis cache hits (sub-millisecond lookups) |
| `url_shortener_cache_misses_total` | Counter | - | Redis cache misses (required database query) |
| `kafka_events_produced_total` | Counter | - | Total click events published to Kafka |
| `kafka_events_consumed_total` | Counter | - | Total click events consumed and batched to DB |

*Dynamic codes in route paths are normalized (e.g. `/api/urls/:code`, `/api/urls/:code/analytics`, `/:code`) to prevent Prometheus high-cardinality explosions.*

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
- **Inspect URL Metadata:**
  ```http
  GET /api/urls/{code}
  ```
- **Click Analytics:**
  ```http
  GET /api/urls/{code}/analytics
  ```
  *Response:*
  ```json
  {
    "code": "8vX1zL90",
    "original_url": "https://example.com",
    "total_clicks": 42,
    "last_clicked": "2026-10-11T12:00:00Z",
    "recent_clicks": [
      {
        "ip": "192.168.0.100",
        "user_agent": "Mozilla/5.0 ...",
        "referer": "https://news.ycombinator.com",
        "timestamp": "2026-10-11T12:00:00Z"
      }
    ]
  }
  ```
- **Prometheus Metrics:**
  ```http
  GET /metrics
  ```