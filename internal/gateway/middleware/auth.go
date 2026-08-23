package middleware

import (
	"context"
	"net/http"
	"strings"

	authjwt "github.com/kaizakin/siphon/pkg/jwt"
)

type contextKey string

const (
	ClaimsContextKey contextKey = "jwt_claims"
	UserIDContextKey contextKey = "user_id"
	RoleContextKey   contextKey = "user_role"
)

// RequireJWT validates the Bearer token in the Authorization header against
// jwtSecret, parses typed claims, validates exp, iat, nbf, iss, aud,
// ensures algorithm is HS256, and stores claims and identity in request context.
func RequireJWT(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			tokenString, ok := strings.CutPrefix(authHeader, "Bearer ")
			if !ok || tokenString == "" {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}

			claims, err := authjwt.ParseToken(tokenString, jwtSecret)
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			ctx = context.WithValue(ctx, UserIDContextKey, claims.Subject)
			ctx = context.WithValue(ctx, RoleContextKey, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole checks that the authenticated user has the specified required role.
// If the role claim is missing or does not match, it rejects the request with 403 Forbidden.
func RequireRole(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(RoleContextKey).(string)
			if !ok || role == "" {
				if claims, ok := r.Context().Value(ClaimsContextKey).(*authjwt.Claims); ok && claims != nil {
					role = claims.Role
				}
			}

			if role != requiredRole {
				http.Error(w, "forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetClaims retrieves Claims from request context.
func GetClaims(r *http.Request) (*authjwt.Claims, bool) {
	claims, ok := r.Context().Value(ClaimsContextKey).(*authjwt.Claims)
	return claims, ok
}

// GetUserID retrieves user ID from request context.
func GetUserID(r *http.Request) (string, bool) {
	id, ok := r.Context().Value(UserIDContextKey).(string)
	return id, ok
}

// GetUserRole retrieves user role from request context.
func GetUserRole(r *http.Request) (string, bool) {
	role, ok := r.Context().Value(RoleContextKey).(string)
	return role, ok
}
