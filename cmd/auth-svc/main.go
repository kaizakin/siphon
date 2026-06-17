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

type Config struct {
	Port string
	DB_URL string
	Jwt_secret string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error reading .env file")
	}

	cfg := Config{
		Port: config.Getenv("PORT"),
		DB_URL: config.Getenv("DB_URL"),
		Jwt_secret: config.Getenv("JWT_SECRET"),
	}

	pool, err := pgxpool.New(context.Background(), cfg.DB_URL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)

	handler := handlers.NewHandler(queries)

	router := routes.SetupRouter(handler)

	log.Printf("Starting Auth server on port %s\n", cfg.Port)
	err = http.ListenAndServe(":" + cfg.Port, router)
	if err != nil {
		log.Fatal(err)
	}
}