package grpcserver

import (
	"context"

	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	
	ingestionv1 "github.com/kaizakin/siphon/gen/ingestion/v1"
	"github.com/kaizakin/siphon/internal/ingestion/sqlc"
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
		Queries: pgxhandler,
	}

	for i := 0; i < 10; i++ {
		go s.kafkaWorker() //
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
			s.writeToDLQ(msg)
		}
	}
}

func (s *IngestionServer) writeToDLQ(msg kafka.Message) {
  _, err := 
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
// TODO: configure postgres to store dlq events and return the pending dlq events as response   
}

func (s *IngestionServer) RetryDLQEvent(ctx context.Context, req *ingestionv1.RetryDLQEventRequest) (*ingestionv1.RetryDLQEventResponse, error) {
// TODO: get the event by eventID push it back in the ingestion buffer
}
