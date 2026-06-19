package handlers

import (
	"encoding/json"
	"net/http"
	"time"

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