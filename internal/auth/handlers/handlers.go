package handlers

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	db "github.com/kaizakin/siphon/internal/auth/sqlc"
)

type Handler struct {
	queries *db.Queries
}

func NewHandler(queries *db.Queries) *Handler {
	return &Handler{
		queries: queries,
	}
}

func (h *Handler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "Email empty", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, "Password less than required length(8)", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)

	_, err := h.queries.CreateUser(
		ctx.Background(), 
		db.CreateUserParams{
			ID: ,
			Email: req.Email,
			PasswordHash: string(hash)
		}
	)
}