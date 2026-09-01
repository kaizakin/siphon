- [x] Protect admin/DLQ endpoints with JWT auth in the API Gateway
- [x] Add role or admin authorization for DLQ routes
- [x] Fix Kafka message format mismatch between Ingestion producer and Email consumer
- [x] Decide the real email event schema, including how the recipient email reaches the Email service
- [x] Store the actual failure reason in `outbox_events.error_message`
- [x] Wire `MarkOutboxEventProcessed` and `MarkOutboxEventFailed` into the DLQ retry flow
- [ ] Demonstrate a failed event going into `outbox_events` and being retried successfully

### Event Reliability

- [x] Enforce idempotency with stable client-provided idempotency keys or event IDs
- [ ] Store accepted events before publishing to Kafka
- [ ] Convert `outbox_events` into a real transactional outbox, or rename/split it into a true DLQ table
- [x] Add automatic retry policy for transient Kafka publish failures
- [x] Add retry count and last retry timestamp to failed events
- [ ] Make Email service idempotent by storing processed event IDs
- [ ] Prevent duplicate emails on Kafka redelivery
- [ ] Add bulk email sending behavior where appropriate

### Architecture & Production Readiness

- [ ] Add structured logging across all services
- [ ] Add correlation ID propagation across all services
- [ ] Improve refresh token lifecycle and revocation
- [ ] Add health check endpoints to all services
- [x] Add rate limiting at the API Gateway
