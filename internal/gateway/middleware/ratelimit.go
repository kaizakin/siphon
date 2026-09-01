package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/kaizakin/siphon/internal/ratelimiter"
) 

func RateLimitByUser(l *ratelimiter.Limiter, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
			userID, ok := GetUserID(r)
			if !ok || userID == "" {
				userID = r.RemoteAddr // fallback to Ip if the user is unauthenticated
			}

			key := fmt.Sprintf("ratelimit:user:%s", userID)
			allowed, remaining, err := l.Allow(r.Context(), key, limit, window)
			if err != nil {
				// fail-open
				fmt.Printf("Something went wrong with redis %s", err)
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

			if !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
				http.Error(w, "too many requests: rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}