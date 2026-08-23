package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	db "github.com/kaizakin/siphon/internal/auth/sqlc"
	"github.com/kaizakin/siphon/pkg/config"
	"github.com/kaizakin/siphon/pkg/dto"
	authjwt "github.com/kaizakin/siphon/pkg/jwt"
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
	var req dto.RegisterRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
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
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	u := uuid.New()
	id := pgtype.UUID{
		Bytes: u,
		Valid: true,
	}

	user, err := h.queries.CreateUser(
		r.Context(),
		db.CreateUserParams{
			ID:           id,
			Email:        req.Email,
			PasswordHash: string(hash),
			Role:         "user",
		},
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "email already exists", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	refreshToken, err := generateRefreshToken(h, user.ID)
	if err != nil {
		http.Error(w, "failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	accessToken, err := generateJWT(user.ID.String(), user.Role)
	if err != nil {
		http.Error(w, "failed to generate access token", http.StatusInternalServerError)
		return
	}

	response := dto.RegisterAndLoginResponse{
		Message:      "User created successfully",
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(response)
}

func generateRefreshToken(h *Handler, userID pgtype.UUID) (string, error) {
	refreshToken := uuid.NewString()
	expiresAt := pgtype.Timestamptz{
		Time:  time.Now().Add(30 * 24 * time.Hour), // 30 days
		Valid: true,
	}

	_, err := h.queries.CreateRefreshToken(
		context.Background(),
		db.CreateRefreshTokenParams{
			UserID:    userID,
			Token:     refreshToken,
			ExpiresAt: expiresAt,
		},
	)

	if err != nil {
		return "", err
	}

	return refreshToken, nil
}

func generateJWT(userID string, role string) (string, error) {
	return authjwt.GenerateToken(userID, role, config.Getenv("JWT_SECRET"), 24*time.Hour)
}

func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.queries.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "User not found!", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
		return
	}

	refreshToken, err := generateRefreshToken(h, user.ID)
	if err != nil {
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	accessToken, err := generateJWT(user.ID.String(), user.Role)
	if err != nil {
		http.Error(w, "failed to generate access token", http.StatusInternalServerError)
		return
	}

	response := dto.RegisterAndLoginResponse{
		Message:      "User successfully logged in!",
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.RefreshToken == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := h.queries.GetRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, "Invalid or nonexistent refresh token", http.StatusUnauthorized)
		return
	}

	// When expired, delete stale token and reject
	if !token.ExpiresAt.Valid || time.Now().After(token.ExpiresAt.Time) {
		_ = h.queries.DeleteRefreshToken(r.Context(), req.RefreshToken)
		http.Error(w, "Refresh token expired", http.StatusUnauthorized)
		return
	}

	user, err := h.queries.GetUserByID(r.Context(), token.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Rotation: delete the old refresh token
	_ = h.queries.DeleteRefreshToken(r.Context(), req.RefreshToken)

	// Create a new rotated refresh token
	newRefreshToken, err := generateRefreshToken(h, token.UserID)
	if err != nil {
		http.Error(w, "failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	accessToken, err := generateJWT(token.UserID.String(), user.Role)
	if err != nil {
		http.Error(w, "failed to generate access token", http.StatusInternalServerError)
		return
	}

	response := dto.RegisterAndLoginResponse{
		Message:      "token refreshed successfully",
		RefreshToken: newRefreshToken,
		AccessToken:  accessToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	var req dto.LogoutRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.RefreshToken == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err = h.queries.DeleteRefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, "Failed to revoke refresh token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
