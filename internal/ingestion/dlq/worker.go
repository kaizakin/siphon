package dlq

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	ingestionv1 "github.com/kaizakin/siphon/gen/ingestion/v1"
	"github.com/kaizakin/siphon/internal/ingestion/sqlc"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type RetryWorker struct {
	queries      *sqlc.Queries
	writer       *kafka.Writer
	pollInterval time.Duration
	baseDelay    time.Duration
	maxDelay     time.Duration
	batchSize    int32
}

func NewRetryWorker(queries *sqlc.Queries, writer *kafka.Writer) *RetryWorker {
	return &RetryWorker{
		queries:      queries,
		writer:       writer,
		pollInterval: 5 * time.Second,
		baseDelay:    2 * time.Second,
		maxDelay:     5 * time.Minute,
		batchSize:    20,
	}
}

// Start begins the background polling loop that periodically retries failed DLQ events.
func (w *RetryWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	log.Println("[DLQ Worker] Automatic retry worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("[DLQ Worker] Stopping automatic retry worker")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *RetryWorker) processBatch(ctx context.Context) {
	events, err := w.queries.GetEventsReadyForRetry(ctx, w.batchSize)
	if err != nil {
		log.Printf("[DLQ Worker] failed to query eligible DLQ events: %v", err)
		return
	}

	for _, e := range events {
		w.retryEvent(ctx, e)
	}
}

func (w *RetryWorker) retryEvent(ctx context.Context, e sqlc.OutboxEvent) {
	var metadata map[string]string
	var payloadMap map[string]interface{}

	if err := json.Unmarshal(e.Metadata, &metadata); err != nil {
		log.Printf("[DLQ Worker] failed to unmarshal metadata for event %s: %v", e.EventID.String(), err)
		return
	}

	if err := json.Unmarshal(e.Payload, &payloadMap); err != nil {
		log.Printf("[DLQ Worker] failed to unmarshal payload for event %s: %v", e.EventID.String(), err)
		return
	}

	payloadStruct, err := structpb.NewStruct(payloadMap)
	if err != nil {
		log.Printf("[DLQ Worker] failed to create structpb for event %s: %v", e.EventID.String(), err)
		return
	}

	eventReq := &ingestionv1.IngestEventRequest{
		EventId:       e.EventID.String(),
		EventType:     e.EventType,
		Source:        e.Source,
		Version:       e.Version,
		Timestamp:     e.Timestamp.Time.Format(time.RFC3339),
		CorrelationId: e.CorrelationID.String(),
		Recipient:     e.Recipient.String,
		Metadata:      metadata,
		Payload:       payloadStruct,
	}

	kafkaPayload, err := protojson.MarshalOptions{
		UseProtoNames: true,
	}.Marshal(eventReq)
	if err != nil {
		log.Printf("[DLQ Worker] failed to marshal protojson for event %s: %v", e.EventID.String(), err)
		return
	}

	msg := kafka.Message{
		Key:   []byte(e.CorrelationID.String()),
		Value: kafkaPayload,
	}

	err = w.writer.WriteMessages(ctx, msg)
	if err == nil {
		if markErr := w.queries.MarkOutboxEventProcessed(ctx, e.EventID); markErr != nil {
			log.Printf("[DLQ Worker] failed to mark event %s as processed: %v", e.EventID.String(), markErr)
		} else {
			log.Printf("[DLQ Worker] Successfully retried and published event %s to Kafka", e.EventID.String())
		}
		return
	}

	nextDelay := w.CalculateBackoff(int(e.RetryCount))
	nextRetryTime := time.Now().UTC().Add(nextDelay)

	if failErr := w.queries.RecordRetryFailure(ctx, sqlc.RecordRetryFailureParams{
		EventID:      e.EventID,
		NextRetryAt:  pgtype.Timestamptz{Time: nextRetryTime, Valid: true},
		ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
	}); failErr != nil {
		log.Printf("[DLQ Worker] failed to record retry failure for event %s: %v", e.EventID.String(), failErr)
	}

	log.Printf("[DLQ Worker] Retry failed for %s (attempt %d). Next retry in %v: %v",
		e.EventID.String(), e.RetryCount+1, nextDelay, err)
}

// CalculateBackoff computes exponential backoff duration with random jitter.
func (w *RetryWorker) CalculateBackoff(retryCount int) time.Duration {
	multiplier := math.Pow(2, float64(retryCount))
	backoff := time.Duration(float64(w.baseDelay) * multiplier)
	if backoff > w.maxDelay {
		backoff = w.maxDelay
	}

	// Add 10-20% random jitter
	jitter := time.Duration(rand.Int63n(int64(backoff / 5)))
	return backoff + jitter
}
