package grpcserver

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	ingestionv1 "github.com/kaizakin/siphon/gen/ingestion/v1"
	"github.com/kaizakin/siphon/internal/ingestion/sqlc"
	db "github.com/kaizakin/siphon/internal/ingestion/sqlc"
)

type IngestionServer struct {
	ingestionv1.UnimplementedEventIngestionServiceServer
	writer     *kafka.Writer
	eventQueue chan *ingestionv1.IngestEventRequest
	Handler // embed handler to the IngestionServer
}

func NewIngestionServer(addr string, pgxhandler *db.Queries) *IngestionServer {
	s := &IngestionServer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(addr),
			Topic:    "events",
			Balancer: &kafka.LeastBytes{},
		},
		eventQueue: make(chan *ingestionv1.IngestEventRequest, 10000), // buffered channel that can hold 10,000 requests.
		Handler: Handler{
		  Queries: pgxhandler,
		},
	}

	for i := 0; i < 10; i++ {
		go s.kafkaWorker() // spawn 10 workers to concurrently utilize the producer resources
	}

	return s
}

// kafka worker keeps on writing messages from the channel to kafka
func (s *IngestionServer) kafkaWorker() {
	for req := range s.eventQueue {
		payload, err := proto.Marshal(req) // marshall the msg to protobuf binary format for transport
		if err != nil {
			continue
		}

		msg := kafka.Message{
			Key:   []byte(req.CorrelationId),
			Value: payload,
		}

		err = s.writer.WriteMessages(context.Background(), msg)
		if err != nil {
			s.writeToDLQ(req)
		}
	}
}

func (s *IngestionServer) writeToDLQ(event *ingestionv1.IngestEventRequest) {
  payloadbytes, err := json.Marshal(event.GetPayload().AsMap())
  if err != nil {
    log.Fatal(err)
  }

  metadatabytes, err := json.Marshal(event.GetMetadata())
  if err != nil {
    log.Fatal(err)
  }

  var eventid pgtype.UUID
  var corrlationid pgtype.UUID
  
  err = eventid.Scan(event.GetEventId())
  if err != nil {
      log.Fatal(err)
  }
  err = corrlationid.Scan(event.GetCorrelationId())
  if err != nil {
    log.Fatal(err)
  }

  paresedTime, err := time.Parse(time.RFC3339, event.Timestamp)
  if err != nil {
    log.Fatal(err)
  }

  ts := pgtype.Timestamptz{
    Time: paresedTime,
    Valid: true,
  }

  _, err = s.Queries.CreateOutboxEvent(
    context.Background(),
    sqlc.CreateOutboxEventParams{
      EventID: eventid,
      EventType: event.EventType,
      Source: event.Source,
      Version: event.Version,
      Timestamp: ts,
      CorrelationID: corrlationid,
      Metadata: metadatabytes,
      Payload: payloadbytes,
    },
  )
  if err != nil {
    log.Fatal(err)
  }
}

// ingestevent sends an optimistic acknowledgement as soon as the event reaches the buffered channel
// this works because the worker handles the event producing & dlq writes
func (s *IngestionServer) IngestEvent(ctx context.Context, req *ingestionv1.IngestEventRequest) (*ingestionv1.IngestEventResponse, error) {
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
      Limit: req.GetLimit(),
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
    Events: dlqEvents,
    Page: req.GetPage(),
    Limit: req.GetLimit(),
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
  
  res, err := s.Handler.Queries.GetOutboxEventByEventID(context.Background(), id)

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
    EventId: res.EventID.String(),
    EventType: res.EventType,
    Source: res.Source,
    Version: res.Version,
    Timestamp: res.Timestamp.Time.String(),
    CorrelationId: res.CorrelationID.String(),
    Metadata: metadata,
    Payload: payload,
  }

  ingesteventResponse, err := s.IngestEvent(context.Background(), event)  
  if err != nil {
    return nil, err
  }

  return &ingestionv1.RetryDLQEventResponse{
    Event: &ingestionv1.DLQEvent{
      EventId: ingesteventResponse.EventId,
    },
    Status: "Sucess",
    Message: "Event accepted",
  }, nil
}
