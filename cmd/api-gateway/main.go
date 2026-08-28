package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"github.com/kaizakin/siphon/internal/gateway/routes"
	"github.com/kaizakin/siphon/pkg/config"
	grpcclient "github.com/kaizakin/siphon/internal/gateway/grpc"
	"github.com/kaizakin/siphon/internal/gateway/handlers"
)

type Config struct {
	Port string
	Auth_svc_url string
	Ingestion_svc_url string
	Jwt_secret string
}

func main() {
	_ = godotenv.Load()
	
	cfg := Config{
		Port:              config.Getenv("PORT"),
		Auth_svc_url:      config.Getenv("AUTH_SVC_URL"),
		Ingestion_svc_url: config.Getenv("INGESTION_SVC_URL"),
		Jwt_secret:        config.Getenv("JWT_SECRET"),
	}
	
	ingestionclient, err := grpcclient.NewIngestionclient(cfg.Ingestion_svc_url)
	if err != nil {
		log.Fatalf("failed to connect to ingestion service at %s: %v", cfg.Ingestion_svc_url, err)
	}

	ingestionhandler := handlers.NewIngestionHandler(ingestionclient)

	r := routes.SetupRouter(cfg.Auth_svc_url, cfg.Jwt_secret, ingestionhandler)

	log.Printf("API gateway running on port %s\n", cfg.Port)
	log.Fatal(http.ListenAndServe(":" + cfg.Port, r))
}
