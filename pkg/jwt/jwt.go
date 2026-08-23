package jwt

import (
	"errors"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

// Claims defines the JWT claims containing the user role and registered claims.
type Claims struct {
	Role string `json:"role"`
	jwtv5.RegisteredClaims
}

// GenerateToken creates and signs a JWT for a user with the given userID, role, secret, and expiry.
func GenerateToken(userID string, role string, secret string, duration time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		Role: role,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(duration)),
		},
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken parses and validates a JWT string against secret and returns the strongly typed Claims.
func ParseToken(tokenString string, secret string) (*Claims, error) {
	token, err := jwtv5.ParseWithClaims(tokenString, &Claims{}, func(t *jwtv5.Token) (any, error) {
		if _, ok := t.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, jwtv5.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
