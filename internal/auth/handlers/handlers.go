package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	db "github.com/kaizakin/siphon/internal/auth/sqlc"
	"github.com/kaizakin/siphon/pkg/config"
)

var jwtsecret = []byte(config.Load().JwtSecret)

type Handler struct {
	queries *db.Queries
}

type RegisterAndLoginResponse struct {
	Message      string `json:"message"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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

	user, err := h.queries.CreateUser(
		context.Background(),
		db.CreateUserParams{
			ID:           id,
			Email:        req.Email,
			PasswordHash: string(hash),
		},
	)

	refreshToken, err := generateRefreshToken(h, user.ID)
	if err != nil {
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	accessToken, err := generateJWT(user.ID.String())
	if err != nil {
		http.Error(w, "failed to generate access token", http.StatusInternalServerError)
		return
	}

	response := RegisterAndLoginResponse{
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
		Time: time.Now().Add(30 * 24 * time.Hour), // 30 days
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

func generateJWT(userID string) (string, error) {
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub": userID,
			"exp": time.Now().Add(24 * time.Hour).Unix(), // 24 Hours of expiry time.
		},
	)

	return token.SignedString(jwtsecret)
}

func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req loginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
	}

	user, err := h.queries.GetUserByEmail(context.Background(), req.Email)
	if err != nil {
		http.Error(w, "User not found!", http.StatusUnauthorized)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		http.Error(w, "Invalid password", http.StatusUnauthorized)
	}

	refreshToken, err := generateRefreshToken(h, user.ID)
	if err != nil {
		http.Error(w, "Failed to generate refresh token", http.StatusInternalServerError)
		return
	}

	accessToken, err := generateJWT(user.ID.String())
	if err != nil {
		http.Error(w, "failed to generate access token", http.StatusInternalServerError)
		return
	}

	response := RegisterAndLoginResponse{
		Message:      "User successfully logged in!",
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	json.NewEncoder(w).Encode(response)
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Failed to decode Body", http.StatusInternalServerError)
		return
	}

	token, err := h.queries.GetRefreshToken(context.Background(), req.RefreshToken)
	if err != nil {
		http.Error(w, "Refreshtoken doesn't exist", http.StatusBadRequest)
		return
	}

	if time.Now().After(token.ExpiresAt.Time) {
		refreshToken, err := generateRefreshToken(h, token.UserID)
		if err != nil {
			http.Error(w, "failed to generate refresh token", http.StatusInternalServerError)
		}
		accessToken, err := generateJWT(token.UserID.String())
		if err != nil {
			http.Error(w, "failed to generate access token", http.StatusInternalServerError)
		}
		
		response := RegisterAndLoginResponse{
			Message:      "refreshtoken created successfully",
			RefreshToken: refreshToken,
			AccessToken:  accessToken,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		json.NewEncoder(w).Encode(response)
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Refreshtoken already valid!"))
	}
}
