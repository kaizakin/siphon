package grpcserver

import (
	db "github.com/kaizakin/siphon/internal/ingestion/sqlc"
)

type Handler struct {
  Queries *db.Queries
}

func NewpgxHandler(queries *db.Queries) *Handler {
  return &Handler{
    Queries: queries,
  }
}
