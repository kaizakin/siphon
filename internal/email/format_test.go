package email

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	ingestionv1 "github.com/kaizakin/siphon/gen/ingestion/v1"
)

func TestKafkaProducerConsumerMessageFormatConsistency(t *testing.T) {
	// 1. Prepare an IngestEventRequest as created by gateway and ingestion-svc
	payloadMap := map[string]any{
		"name":     "Karthik",
		"order_id": "ord_98765",
	}
	payloadStruct, err := structpb.NewStruct(payloadMap)
	if err != nil {
		t.Fatalf("failed to create structpb: %v", err)
	}

	req := &ingestionv1.IngestEventRequest{
		EventId:       "evt_12345",
		EventType:     "order_cancelled",
		Source:        "api-gateway",
		Version:       "v1",
		Timestamp:     "2026-08-24T00:00:00Z",
		CorrelationId: "corr_54321",
		Recipient:     "user@example.com",
		Payload:       payloadStruct,
	}

	// 2. Marshal using protojson with UseProtoNames (as in IngestionServer.publishToKafka)
	kafkaMessageBytes, err := protojson.MarshalOptions{
		UseProtoNames: true,
	}.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal protojson: %v", err)
	}

	// 3. Unmarshal in email consumer (as in kafkaconsumer.go)
	var event Event
	if err := json.Unmarshal(kafkaMessageBytes, &event); err != nil {
		t.Fatalf("failed to unmarshal kafka json message: %v", err)
	}

	// 4. Assert all top-level fields are parsed accurately
	if event.EventID != "evt_12345" {
		t.Errorf("expected EventID 'evt_12345', got %q", event.EventID)
	}
	if event.EventType != "order_cancelled" {
		t.Errorf("expected EventType 'order_cancelled', got %q", event.EventType)
	}
	if event.Recipient != "user@example.com" {
		t.Errorf("expected Recipient 'user@example.com', got %q", event.Recipient)
	}
	if event.CorrelationID != "corr_54321" {
		t.Errorf("expected CorrelationID 'corr_54321', got %q", event.CorrelationID)
	}

	// 5. Assert payload decodes to typed handler struct without nesting or loss
	var orderData OrderCancelledData
	if err := decodeEventData(event.Payload, &orderData); err != nil {
		t.Fatalf("decodeEventData failed: %v", err)
	}

	if orderData.Name != "Karthik" {
		t.Errorf("expected Name 'Karthik', got %q", orderData.Name)
	}
	if orderData.OrderID != "ord_98765" {
		t.Errorf("expected OrderID 'ord_98765', got %q", orderData.OrderID)
	}
}
