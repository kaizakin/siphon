package routes

import (
	"github.com/go-chi/chi/v5"

	"github.com/kaizakin/siphon/internal/auth/handlers"
)

func SetupRouter(h *handlers.Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", h.RegisterHandler)
	})

	return r
}
