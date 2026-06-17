package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"github.com/kaizakin/siphon/internal/gateway/routes"
	"github.com/kaizakin/siphon/pkg/config"
)

type Config struct {
	Port string
	Auth_svc_url string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error reading .env file")
	}
	
	cfg := Config{
		Port: config.Getenv("PORT"),
		Auth_svc_url: config.Getenv("AUTH_SVC_URL"),
	}

	r := routes.SetupRouter(cfg.Auth_svc_url)

	log.Fatal(http.ListenAndServe(":" + cfg.Port, r))
}
