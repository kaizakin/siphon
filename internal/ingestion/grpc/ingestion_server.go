package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	ingestionv1 "github.com/kaizakin/siphon/gen/ingestion/v1"
	"github.com/kaizakin/siphon/internal/ingestion/sqlc"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type Config struct {
	Port       string
	Kafka_url  string
	Topic_name string
}

type IngestionServer struct {
	ingestionv1.UnimplementedEventIngestionServiceServer
	Handler *Handler

	writer *kafka.Writer
	*sqlc.Queries

	eventQueue chan *ingestionv1.IngestEventRequest
}

func NewIngestionServer(queries *sqlc.Queries, writer *kafka.Writer, bufferSize int, workerCount int) *IngestionServer {
	s := &IngestionServer{
		Handler:    &Handler{Queries: queries},
		Queries:    queries,
		writer:     writer,
		eventQueue: make(chan *ingestionv1.IngestEventRequest, bufferSize),
	}

	for i := 0; i < workerCount; i++ {
		go s.kafkaWorker()
	}

	return s
}

func (s *IngestionServer) kafkaWorker() {
	for req := range s.eventQueue {
		if err := s.publishToKafka(context.Background(), req); err != nil {
			log.Printf("Failed to write to kafka: %v", err)
			s.writeToDLQ(req, err)
		}
	}
}

// publishToKafka marshals the event to JSON using protojson so that nested
// structpb.Struct payloads and snake_case field names match consumer expectations.
func (s *IngestionServer) publishToKafka(ctx context.Context, req *ingestionv1.IngestEventRequest) error {
	payload, err := protojson.MarshalOptions{
		UseProtoNames: true,
	}.Marshal(req)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(req.CorrelationId),
		Value: payload,
	}

	return s.writer.WriteMessages(ctx, msg)
}

func (s *IngestionServer) writeToDLQ(event *ingestionv1.IngestEventRequest, kafkaerror error) {
	var payloadMap map[string]any
	if event.GetPayload() != nil {
		payloadMap = event.GetPayload().AsMap()
	} else {
		payloadMap = make(map[string]any)
	}

	payloadbytes, err := json.Marshal(payloadMap)
	if err != nil {
		log.Printf("failed to marshal payload for DLQ: %v", err)
		return
	}

	metadata := event.GetMetadata()
	if metadata == nil {
		metadata = make(map[string]string)
	}
	metadatabytes, err := json.Marshal(metadata)
	if err != nil {
		log.Printf("failed to marshal metadata for DLQ: %v", err)
		return
	}

	var eventid pgtype.UUID
	var correlationid pgtype.UUID

	if err := eventid.Scan(event.GetEventId()); err != nil {
		log.Printf("invalid event_id %q for DLQ: %v", event.GetEventId(), err)
		return
	}
	if err := correlationid.Scan(event.GetCorrelationId()); err != nil {
		log.Printf("invalid correlation_id %q for DLQ: %v", event.GetCorrelationId(), err)
		return
	}

	parsedTime, err := time.Parse(time.RFC3339, event.Timestamp)
	if err != nil {
		parsedTime = time.Now().UTC()
	}

	ts := pgtype.Timestamptz{
		Time:  parsedTime,
		Valid: true,
	}

	recipient := pgtype.Text{
		String: event.Recipient,
		Valid:  event.Recipient != "",
	}

	_, err = s.Queries.CreateOutboxEvent(
		context.Background(),
		sqlc.CreateOutboxEventParams{
			EventID:       eventid,
			EventType:     event.EventType,
			Source:        event.Source,
			Version:       event.Version,
			Timestamp:     ts,
			CorrelationID: correlationid,
			Metadata:      metadatabytes,
			Payload:       payloadbytes,
			Recipient:     recipient,
			ErrorMessage: pgtype.Text{
				String: kafkaerror.Error(),
				Valid:  true,
			},
		},
	)
	if err != nil {
		log.Printf("failed to insert event %s into DLQ: %v", event.GetEventId(), err)
	}
}

// IngestEvent validates UUID event identifiers and places the request into the worker queue.
func (s *IngestionServer) IngestEvent(ctx context.Context, req *ingestionv1.IngestEventRequest) (*ingestionv1.IngestEventResponse, error) {
	if req.EventId == "" {
		req.EventId = uuid.NewString()
	} else if _, err := uuid.Parse(req.EventId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid event_id format (must be a valid UUID): %v", err)
	}

	if req.CorrelationId == "" {
		req.CorrelationId = uuid.NewString()
	} else if _, err := uuid.Parse(req.CorrelationId); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid correlation_id format (must be a valid UUID): %v", err)
	}

	select {
	case s.eventQueue <- req:
		return &ingestionv1.IngestEventResponse{
			EventId: req.EventId,
			Status:  "event accepted",
		}, nil

	default:
		return nil, status.Error(codes.ResourceExhausted, "Ingestion queue is full")
	}
}

func (s *IngestionServer) ListDLQEvents(ctx context.Context, req *ingestionv1.ListDLQEventsRequest) (*ingestionv1.ListDLQEventsResponse, error) {

	// method gets promoted so can be accessed like this
	events, err := s.Queries.GetPendingOutboxEvents(context.Background(),
		sqlc.GetPendingOutboxEventsParams{
			Limit:  req.GetLimit(),
			Offset: req.GetPage(),
		},
	)
	if err != nil {
		return nil, err
	}

	dlqEvents := make([]*ingestionv1.DLQEvent, 0, len(events))

	for _, e := range events {
		dlqEvents = append(dlqEvents, &ingestionv1.DLQEvent{
			EventId:       e.EventID.String(),
			CorrelationId: e.CorrelationID.String(),
			EventType:     e.EventType,
			Source:        e.Source,
			Version:       e.Version,
			FailureReason: e.ErrorMessage.String,
			FailedAt:      e.CreatedAt.Time.String(),
		})
	}

	response := &ingestionv1.ListDLQEventsResponse{
		Events:     dlqEvents,
		Page:       req.GetPage(),
		Limit:      req.GetLimit(),
		TotalCount: int64(len(events)),
	}

	return response, nil
}

func (s *IngestionServer) RetryDLQEvent(ctx context.Context, req *ingestionv1.RetryDLQEventRequest) (*ingestionv1.RetryDLQEventResponse, error) {
	var id pgtype.UUID

	err := id.Scan(req.GetEventId())
	if err != nil {
		return nil, err
	}

	res, err := s.Handler.Queries.GetOutboxEventByEventID(ctx, id)
	if err != nil {
		return nil, err
	}

	var metadata map[string]string
	var payloadmap map[string]interface{}

	err = json.Unmarshal(res.Metadata, &metadata)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(res.Payload, &payloadmap)
	if err != nil {
		return nil, err
	}

	payload, err := structpb.NewStruct(payloadmap)
	if err != nil {
		return nil, err
	}

	event := &ingestionv1.IngestEventRequest{
		EventId:       res.EventID.String(),
		EventType:     res.EventType,
		Source:        res.Source,
		Version:       res.Version,
		Timestamp:     res.Timestamp.Time.String(),
		CorrelationId: res.CorrelationID.String(),
		Recipient:     res.Recipient.String,
		Metadata:      metadata,
		Payload:       payload,
	}

	if err := s.publishToKafka(ctx, event); err != nil {
		if markErr := s.Queries.MarkOutboxEventFailed(ctx, sqlc.MarkOutboxEventFailedParams{
			EventID:      id,
			ErrorMessage: pgtype.Text{String: err.Error(), Valid: true},
		}); markErr != nil {
			log.Printf("failed to mark outbox event %s as failed: %v", req.GetEventId(), markErr)
		}

		return &ingestionv1.RetryDLQEventResponse{
			Event:   &ingestionv1.DLQEvent{EventId: res.EventID.String()},
			Status:  "failed",
			Message: fmt.Sprintf("retry failed: %v", err),
		}, nil
	}

	if err := s.Queries.MarkOutboxEventProcessed(ctx, id); err != nil {
		log.Printf("failed to mark outbox event %s as processed: %v", req.GetEventId(), err)
	}

	return &ingestionv1.RetryDLQEventResponse{
		Event: &ingestionv1.DLQEvent{
			EventId: res.EventID.String(),
		},
		Status:  "success",
		Message: "event retried and published successfully",
	}, nil
}
