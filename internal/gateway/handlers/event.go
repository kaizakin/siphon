package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	ingesv1 "github.com/kaizakin/siphon/gen/ingestion/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

const grpcVersion = "v1"

type IngestionHandler struct {
	Client ingesv1.EventIngestionServiceClient
}

type createEventRequest struct {
	EventType string  `json:"event_type"`
	Payload map[string]any `json:"payload"`
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
		http.Error(w, "failed to decode req body", http.StatusInternalServerError)
		return
	}

	payloadStruct, _ := structpb.NewStruct(req.Payload)

	resp, err := h.Client.IngestEvent(ctx, 
		&ingesv1.IngestEventRequest{
			EventId: uuid.NewString(),
			EventType: req.EventType,
			Source: "api-gateway",
			Version: grpcVersion,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			CorrelationId: uuid.NewString(),
			Payload: payloadStruct,
		},
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(resp)
}

type getdlqeventsRequest struct {
	Page int32 `json:"page"`
	Limit int32 `json:"limit"`
}

func (h *IngestionHandler) GetDLQEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req getdlqeventsRequest

	err:= json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Failed to parse the request", http.StatusInternalServerError)
		return
	}

	resp, err := h.Client.ListDLQEvents(ctx, 
		&ingesv1.ListDLQEventsRequest{
			Page: req.Page,
			Limit: req.Limit,
		},
	)
	if err != nil {
		http.Error(w, "grpc request failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(resp)
}

func (h *IngestionHandler) RetryDLQEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	resp, err := h.Client.RetryDLQEvent(ctx,
		&ingesv1.RetryDLQEventRequest{
			EventId: id,
		},
	)
	if err != nil {
		http.Error(w, "grpc request failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(resp)		
}
