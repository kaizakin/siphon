package routes

import (
	"net/http/httputil"
	"net/url"

	"github.com/go-chi/chi/v5"
	// h "github.com/kaizakin/siphon/internal/gateway/handlers"
)

func SetupRouter(auth_url string) *chi.Mux {
	r := chi.NewRouter()

	r.Route("/api/v1/", func(r chi.Router) {
		// auth service
		authURL, err := url.Parse(auth_url)
		if err != nil {
			panic(err)
		}
		authProxy := httputil.NewSingleHostReverseProxy(authURL)
		r.Handle("/auth/*", authProxy)
		
		// event ingestion service
		// r.Post("/event")

		// dead letter queue (event ingestion)
		// r.Get("/dlq/events")
		// r.Post("/dlq/events/{id}/retry")
	})

	return r
}