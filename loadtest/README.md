# k6 Load Testing for Siphon

This directory contains k6 load testing scripts to benchmark the Siphon API Gateway and downstream services.

## Test Scripts

1. **`events.js`**: Load test for the primary event ingestion endpoint (`POST /api/v1/event`).
   - Simulates multi-user load ramp-up.
   - Generates dynamic payloads for different event types (`order_success`, `signup_thankyou`, `payment_success`, `order_failed`).
   - Uses unique `Idempotency-Key` headers.

2. **`auth.js`**: Load test for the authentication pipeline.
   - Tests user registration, login, and refresh token rotation under concurrency.

3. **`ratelimit.js`**: Stress test specifically testing the sliding-window Redis rate limiter (100 req/min per user).

## Running with Docker

### 1. When running Siphon via Docker Compose
When Siphon is running in Docker Compose, run k6 attached to the compose network:

```bash
docker run --rm -i \
  --network siphon_default \
  -v $(pwd)/loadtest:/loadtest \
  -e BASE_URL=http://api-gateway:5000 \
  grafana/k6 run /loadtest/events.js
```

### 2. When running Siphon locally on Host (`localhost:5000`)
On Linux, attach to the host network:

```bash
docker run --rm -i \
  --network host \
  -v $(pwd)/loadtest:/loadtest \
  -e BASE_URL=http://localhost:5000 \
  grafana/k6 run /loadtest/events.js
```

Or using `host.docker.internal`:

```bash
docker run --rm -i \
  --add-host host.docker.internal:host-gateway \
  -v $(pwd)/loadtest:/loadtest \
  -e BASE_URL=http://host.docker.internal:5000 \
  grafana/k6 run /loadtest/events.js
```
