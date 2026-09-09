# API Docs (Postman walkthrough)

Base URL for everything: `http://localhost:5000` (the api-gateway). Run through these in order.

## 1. Register

`POST /api/v1/auth/register`

```json
{
  "email": "test@example.com",
  "password": "password123"
}
```

Expect `201 Created`:

```json
{
  "message": "User created successfully",
  "access_token": "<jwt>",
  "refresh_token": "<uuid>"
}
```

**Next step:** save `access_token` and `refresh_token` from the response, you'll need them below.

## 2. Login

`POST /api/v1/auth/login`

```json
{
  "email": "test@example.com",
  "password": "password123"
}
```

Expect `202 Accepted`:

```json
{
  "message": "User successfully logged in!",
  "access_token": "<jwt>",
  "refresh_token": "<uuid>"
}
```

**Next step:** use this `access_token` as a Bearer token for the requests below.

## 3. Refresh

`POST /api/v1/auth/refresh`

```json
{
  "refresh_token": "<refresh_token from step 1 or 2>"
}
```

Expect `200 OK`:

```json
{
  "message": "token refreshed successfully",
  "access_token": "<jwt>",
  "refresh_token": "<new uuid>"
}
```

**Next step:** the old refresh token is now invalid, replace it with the new one. This is refresh token rotation.

## 4. Create an event

`POST /api/v1/event`

Headers:
```
Authorization: Bearer <access_token>
Content-Type: application/json
Idempotency-Key: 3fa85f64-5717-4562-b3fc-2c963f66afa6   (optional)
```

```json
{
  "event_type": "order.confirmed",
  "recipient": "customer@example.com",
  "payload": {
    "order_id": "ORD-1001",
    "amount": 49.99
  }
}
```

Expect `200 OK`:

```json
{
  "event_id": "<uuid>",
  "status": "event accepted"
}
```

**Next step:** this call is rate-limited to 100/min per user. Send it a few times to see the flow work, then check the DLQ below (only useful if a publish actually failed, but the endpoint is there to inspect either way).

## 5. Promote yourself to admin (needed before step 6 and 7)

The DLQ endpoints are admin-only, and `/register` always creates a `user`-role account. Promote the account from the shell:

```bash
task admin:create -- -email test@example.com
```

Then log in again (step 2) to get a fresh access token, JWTs bake the role in at login time, so your old token is still `user`.

## 6. List DLQ events (admin only)

`GET /api/v1/dlq/events`

Headers:
```
Authorization: Bearer <admin access_token>
```

Body (optional, defaults apply if omitted):

```json
{
  "page": 0,
  "limit": 20
}
```

Expect `200 OK`:

```json
{
  "events": [
    {
      "event_id": "<uuid>",
      "correlation_id": "<uuid>",
      "event_type": "order.confirmed",
      "source": "...",
      "version": "...",
      "payload": { "order_id": "ORD-1001", "amount": 49.99 },
      "failure_reason": "...",
      "retry_count": 0,
      "failed_at": "..."
    }
  ],
  "page": 0,
  "limit": 20,
  "total_count": 1
}
```

**Next step:** grab an `event_id` from the list to retry it.

## 7. Retry a DLQ event (admin only)

`POST /api/v1/dlq/events/{event_id}/retry`

Headers:
```
Authorization: Bearer <admin access_token>
```

No body needed.

Expect `200 OK`:

```json
{
  "event": { "event_id": "<uuid>" },
  "status": "success",
  "message": "event retried and published successfully"
}
```

**Next step:** none, you've exercised the full flow: register, auth, ingest, inspect DLQ, retry.

## 8. Logout

`POST /api/v1/auth/logout`

```json
{
  "refresh_token": "<current refresh_token>"
}
```

Expect `204 No Content`.
