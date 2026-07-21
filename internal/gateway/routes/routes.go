package routes

import (
	"net/http/httputil"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/kaizakin/siphon/internal/gateway/handlers"
	"github.com/kaizakin/siphon/internal/gateway/middleware"
)

func SetupRouter(auth_url string, jwtSecret string, handler *handlers.IngestionHandler) *chi.Mux {
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

			// event ingestion service
			r.Post("/event", handler.CreateEvent)

			// admin / dead letter queue (event ingestion) - JWT protected
			r.Get("/dlq/events", handler.GetDLQEvents)
			r.Post("/dlq/events/{id}/retry", handler.RetryDLQEvent)
		})
	})

	return r
}
