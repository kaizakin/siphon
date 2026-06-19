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
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error reading .env file")
	}
	
	cfg := Config{
		Port: config.Getenv("PORT"),
		Auth_svc_url: config.Getenv("AUTH_SVC_URL"),
		Ingestion_svc_url: config.Getenv("INGESTION_SVC_URL"),
	}
	
	ingestionclient, err := grpcclient.NewIngestionclient(cfg.Ingestion_svc_url)
	if err != nil {
		log.Fatal("err")
	}

	ingestionhandler := handlers.NewIngestionHandler(ingestionclient)

	r := routes.SetupRouter(cfg.Auth_svc_url, ingestionhandler)

	log.Printf("API gateway running on port %s\n", cfg.Port)
	log.Fatal(http.ListenAndServe(":" + cfg.Port, r))
}
