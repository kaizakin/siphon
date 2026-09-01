package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	ingesv1 "github.com/kaizakin/siphon/gen/ingestion/v1"
	"github.com/kaizakin/siphon/internal/gateway/middleware"
	"google.golang.org/protobuf/types/known/structpb"
)

const grpcVersion = "v1"

type IngestionHandler struct {
	Client ingesv1.EventIngestionServiceClient
}

type createEventRequest struct {
	EventType string         `json:"event_type"`
	Payload   map[string]any `json:"payload"`
	Recipient string         `json:"recipient"`
}

func NewIngestionHandler(client ingesv1.EventIngestionServiceClient) *IngestionHandler {
	return &IngestionHandler{
		Client: client,
	}
}

func (h *IngestionHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req createEventRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.EventType == "" {
		http.Error(w, "event_type is required", http.StatusBadRequest)
		return
	}

	if req.Recipient == "" {
		http.Error(w, "recipient is required", http.StatusBadRequest)
		return
	}

	if req.Payload == nil {
		req.Payload = make(map[string]any)
	}

	payloadStruct, err := structpb.NewStruct(req.Payload)
	if err != nil {
		http.Error(w, "failed to parse payload structure", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r)
	if !ok {
		http.Error(w, "User unauthorized", http.StatusUnauthorized)
		return
	}

	userRole, ok := middleware.GetUserRole(r)
	if !ok {
		http.Error(w, "User role not found", http.StatusUnauthorized)
		return
	}

	// Extract optional client-provided Idempotency-Key
	eventID := r.Header.Get("Idempotency-Key")
	if eventID == "" {
		eventID = uuid.NewString()
	} else if _, err := uuid.Parse(eventID); err != nil {
		http.Error(w, "invalid Idempotency-Key header: must be a valid UUID", http.StatusBadRequest)
		return
	}

	resp, err := h.Client.IngestEvent(ctx,
		&ingesv1.IngestEventRequest{
			EventId:       eventID,
			EventType:     req.EventType,
			Source:        "api-gateway",
			Version:       grpcVersion,
			Timestamp:     time.Now().UTC().Format(time.RFC3339),
			CorrelationId: uuid.NewString(),
			Payload:       payloadStruct,
			Recipient:     req.Recipient,
			Metadata: map[string]string{
				"user_id":   userID,
				"user_role": userRole,
			},
		},
	)
	if err != nil {
		http.Error(w, "failed to ingest event: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(resp)
}

type getdlqeventsRequest struct {
	Page  int32 `json:"page"`
	Limit int32 `json:"limit"`
}

func (h *IngestionHandler) GetDLQEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req getdlqeventsRequest

	if r.Body != nil && r.ContentLength > 0 {
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
	}

	if req.Limit <= 0 {
		req.Limit = 20
	}

	resp, err := h.Client.ListDLQEvents(ctx,
		&ingesv1.ListDLQEventsRequest{
			Page:  req.Page,
			Limit: req.Limit,
		},
	)
	if err != nil {
		http.Error(w, "failed to get DLQ events: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(resp)
}

type retryEventRequest struct {
	ID string `json:"id"`
}

func (h *IngestionHandler) RetryDLQEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	eventID := r.PathValue("id")
	if eventID == "" {
		http.Error(w, "event id is required", http.StatusBadRequest)
		return
	}

	resp, err := h.Client.RetryDLQEvent(
		ctx,
		&ingesv1.RetryDLQEventRequest{
			EventId: eventID,
		},
	)
	if err != nil {
		http.Error(w, "failed to retry dlq event: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(resp)
}
