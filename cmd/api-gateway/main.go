package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	grpcclient "github.com/kaizakin/siphon/internal/gateway/grpc"
	"github.com/kaizakin/siphon/internal/gateway/handlers"
	"github.com/kaizakin/siphon/internal/gateway/routes"
	"github.com/kaizakin/siphon/internal/ratelimiter"
	"github.com/kaizakin/siphon/pkg/config"
)

type Config struct {
	Port              string
	Auth_svc_url      string
	Ingestion_svc_url string
	Jwt_secret        string
	Redis_url         string
}

func main() {
	_ = godotenv.Load()

	cfg := Config{
		Port:              config.Getenv("PORT"),
		Auth_svc_url:      config.Getenv("AUTH_SVC_URL"),
		Ingestion_svc_url: config.Getenv("INGESTION_SVC_URL"),
		Jwt_secret:        config.Getenv("JWT_SECRET"),
		Redis_url:         config.Getenv("REDIS_ADDR"),
	}


	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis_url,
	})

	// quick  ping to verify the connection
	ctx, cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect with redis at %s: %v", cfg.Redis_url, err)
	}	

	ratelimiter := ratelimiter.NewLimiter(rdb)

	ingestionclient, err := grpcclient.NewIngestionclient(cfg.Ingestion_svc_url)
	if err != nil {
		log.Fatalf("failed to connect to ingestion service at %s: %v", cfg.Ingestion_svc_url, err)
	}

	ingestionhandler := handlers.NewIngestionHandler(ingestionclient)

	r := routes.SetupRouter(cfg.Auth_svc_url, cfg.Jwt_secret, ingestionhandler, ratelimiter)

	log.Printf("API gateway running on port %s\n", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}
