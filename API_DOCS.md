# Siphon API — quick test reference

| Service | Port | Protocol |
|---|---|---|
| api-gateway | `5000` | HTTP — the only entrypoint you should call directly |
| auth-svc | `5002` | HTTP — normally only reached via the gateway's `/auth/*` proxy |
| ingestion-svc | `5003` | gRPC — called by the gateway, not directly by you |
| email-svc | n/a | Kafka consumer only, no exposed port |
| Kafka | `9092` | via `docker compose up -d` (see `docker-compose.yaml`) |

All routes below are served **through the gateway** on `:5000`.

---

## Auth

### Register
POST /api/v1/auth/register
Content-Type: application/json

```
{
  "email": "you@example.com",
  "password": "at-least-8-chars"
}
```
→ `201 Created`
```json
{ "message": "...", "access_token": "...", "refresh_token": "..." }
```

### Login
POST /api/v1/auth/login
Content-Type: application/json
```
{
  "email": "you@example.com",
  "password": "at-least-8-chars"
}
```
→ `202 Accepted`, same shape as register.

### Refresh
POST /api/v1/auth/refresh
Content-Type: application/json

```json
{ "refresh_token": "<refresh_token from login>" }
```
→ `200 OK` (consistent JSON response with token rotation):
```json
{
  "message": "token refreshed successfully",
  "access_token": "...",
  "refresh_token": "..."
}
```
If expired or invalid → `401 Unauthorized`.

### Logout
POST /api/v1/auth/logout
Content-Type: application/json

```json
{ "refresh_token": "<refresh_token from login>" }
```
→ `204 No Content` (revokes the refresh token)

---

## Events (ingestion → Kafka → email)

### Create event

POST /api/v1/event
Content-Type: application/json

```
{
  "event_type": "order_cancelled",
  "recipient": "you@example.com",
  "payload": {
    "Name": "Karthik",
    "OrderID": "ord_123"
  }
}
```
→ `200 OK`, proxied straight from the gRPC response:
```json
{ "event_id": "<uuid>", "status": "event accepted" }
```

`event_type` must be one of the 7 registered in
`internal/email/router.go` for an email to actually fire:
`signup_thankyou`, `order_success`, `order_failed`, `order_cancelled`,
`payment_success`, `payment_failed`, `payment_refunded`. The `payload`
fields must match the corresponding struct in
`internal/email/templatepayloads.go` (shown for `order_cancelled` above —
`Name`, `OrderID`). Field-name matching is case-insensitive courtesy of
`encoding/json`, but the *names* still have to correspond.


### List DLQ'd events
```
GET /api/v1/dlq/events
Content-Type: application/json

{ "page": 0, "limit": 20 }
```
(Note: this is a `GET` that still expects a JSON body — not query params.)
→ `200 OK`
```json
{ "events": [ /* DLQEvent[] */ ], "page": 0, "limit": 20, "total_count": 0 }
```
An event lands here only if the Kafka write itself failed (e.g. Kafka is
down), not for the payload-key mismatch above — that failure happens
downstream in email-svc, silently (see `FINDINGS.md`).

### Retry a DLQ'd event
```
POST /api/v1/dlq/events/{id}/retry
```
→ `200 OK` with the re-submitted event's new status.

---

## Suggested manual test sequence

```bash
docker compose up -d                 # starts Kafka on :9092
task db:migrate                      # applies migrations to the Neon DB
# in separate terminals:
task run:auth
task run:gateway
task run:ingestion
task run:email

# 1. register + login
curl -s -X POST localhost:5000/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"testpass123"}'

# 2. fire an event that should trigger an email
curl -s -X POST localhost:5000/api/v1/event \
  -H 'Content-Type: application/json' \
  -d '{"event_type":"order_cancelled","recipient":"you@example.com","payload":{"Name":"Karthik","OrderID":"ord_123"}}'

# 3. watch the email-svc terminal for consume/send logs,
#    and check the DLQ if the event never shows up there
curl -s -X GET localhost:5000/api/v1/dlq/events \
  -H 'Content-Type: application/json' -d '{"page":0,"limit":20}'
```
