package routes

import (
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaizakin/siphon/internal/gateway/handlers"
	"github.com/kaizakin/siphon/internal/gateway/middleware"
	"github.com/kaizakin/siphon/internal/ratelimiter"
)

func SetupRouter(auth_url string, jwtSecret string, handler *handlers.IngestionHandler, ratelimiter *ratelimiter.Limiter) *chi.Mux {
	r := chi.NewRouter()

	r.Route("/api/v1/", func(r chi.Router) {
		// auth service
		authURL, err := url.Parse(auth_url)
		if err != nil {
			panic(err)
		}
		authProxy := httputil.NewSingleHostReverseProxy(authURL)
		r.Handle("/auth/*", authProxy)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireJWT(jwtSecret))

			// 100 ingestions per-minute per user
			r.With(middleware.RateLimitByUser(ratelimiter, 100, time.Minute)).Post("/event", handler.CreateEvent)

			// admin / dead letter queue (event ingestion) - JWT + Admin role protected
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))

				r.Get("/dlq/events", handler.GetDLQEvents)
				r.Post("/dlq/events/{id}/retry", handler.RetryDLQEvent)
			})
		})
	})

	return r
}
