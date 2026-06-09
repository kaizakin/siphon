package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
	"github.com/golang-jwt/jwt/v5"

	db "github.com/kaizakin/siphon/internal/auth/sqlc"
	"github.com/kaizakin/siphon/pkg/config"
)

var jwtsecret = []byte(config.Load().JwtSecret)

type Handler struct {
	queries *db.Queries
}

type createUserResponse struct {
	Message string 			`json:"message"`
	AccessToken string 		`json:"access_token"`
	RefreshToken string 	`json:"refresh_token"`
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
			ID: id,
			Email: req.Email,
			PasswordHash: string(hash),
		},
	)

	refreshToken := uuid.NewString()
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days

	_, err = h.queries.CreateRefreshToken(
		context.Background(),
		db.CreateRefreshTokenParams{
			UserID: user.ID,
			Token: refreshToken,
			ExpiresAt: expiresAt,
		},
	)

	accessToken, err := generateJWT(user.ID.String())
	if err != nil {
		http.Error(w, "failed to generate access token", http.StatusInternalServerError)
		return
	}

	response := createUserResponse{
		Message: "User created successfully",
		RefreshToken: refreshToken,
		AccessToken: accessToken,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(response)
	return
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