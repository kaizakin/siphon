package handlers

import (
	"encoding/json"
	"net/http"
	"context"

	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/kaizakin/siphon/internal/auth/sqlc"
)

type Handler struct {
	queries *db.Queries
}

type createUserResponse struct {
	Message string `json:"message"`
	UserID string `json:"user_id"`
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

	u := uuid.New()
	// a uuid column in postgresql can contain NULL, but in go normal UUID type can only hold values
	// so wrap that in a new type which has a valid flag which denotes whether this uuid has a real value or not.
	id := pgtype.UUID{ 
		Bytes: u,
		Valid: true,
	}

	_, err = h.queries.CreateUser(
		context.Background(), 
		db.CreateUserParams{
			ID: id,
			Email: req.Email,
			PasswordHash: string(hash),
		},
	)

	response := createUserResponse{
		Message: "User created successfully",
		UserID: id.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(response)
}