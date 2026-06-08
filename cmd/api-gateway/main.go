package main

import (
	"log"
	"net/http"

	"github.com/kaizakin/siphon/internal/gateway/routes"
	"github.com/kaizakin/siphon/pkg/config"
)

func main() {
	cfg := config.Load()

	r := routes.SetupRouter()

	log.Fatal(http.ListenAndServe(":" + cfg.Port, r))
}
