package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	ingesv1 "github.com/kaizakin/siphon/gen/ingestion/v1"
	"google.golang.org/grpc"
)

type mockIngestionClient struct {
	ingesv1.EventIngestionServiceClient
	ingestFunc func(ctx context.Context, in *ingesv1.IngestEventRequest, opts ...grpc.CallOption) (*ingesv1.IngestEventResponse, error)
}

func (m *mockIngestionClient) IngestEvent(ctx context.Context, in *ingesv1.IngestEventRequest, opts ...grpc.CallOption) (*ingesv1.IngestEventResponse, error) {
	if m.ingestFunc != nil {
		return m.ingestFunc(ctx, in, opts...)
	}
	return &ingesv1.IngestEventResponse{EventId: in.EventId, Status: "event accepted"}, nil
}

func TestCreateEvent_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "invalid json body",
			body:           `{invalid-json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing event_type",
			body:           `{"recipient": "user@example.com", "payload": {}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing recipient",
			body:           `{"event_type": "order_success", "payload": {}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "valid event request",
			body:           `{"event_type": "order_success", "recipient": "user@example.com", "payload": {"Name": "Test"}}`,
			expectedStatus: http.StatusOK,
		},
	}

	handler := NewIngestionHandler(&mockIngestionClient{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/event", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			handler.CreateEvent(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRetryDLQEvent_MissingID(t *testing.T) {
	handler := NewIngestionHandler(&mockIngestionClient{})

	r := chi.NewRouter()
	r.Post("/api/v1/dlq/events/{id}/retry", handler.RetryDLQEvent)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dlq/events//retry", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Errorf("expected non-OK status for empty ID, got %d", rec.Code)
	}
}
