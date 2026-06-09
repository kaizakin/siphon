package main

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/kaizakin/siphon/internal/auth/handlers"
	"github.com/kaizakin/siphon/internal/auth/routes"
	"github.com/kaizakin/siphon/internal/auth/sqlc"
	"github.com/kaizakin/siphon/pkg/config"
)
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error reading .env file")
	}

	cfg := config.Load()

	pool, err := pgxpool.New(context.Background(), cfg.DB_URL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)

	handler := handlers.NewHandler(queries)

	router := routes.SetupRouter(handler)

	err = http.ListenAndServe(":" + cfg.Port, router)
	if err != nil {
		log.Fatal(err)
	}
}