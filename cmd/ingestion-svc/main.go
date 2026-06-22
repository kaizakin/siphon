package main

import (
	"log"
	"net"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"

	ingestionv1 "github.com/kaizakin/siphon/gen/ingestion/v1"
	server "github.com/kaizakin/siphon/internal/ingestion/grpc"
	"github.com/kaizakin/siphon/pkg/config"
)

type Config struct {
  Port string
  KafkaBroker string
}

func main() {
  err := godotenv.Load()
  if err != nil {
    log.Fatal("Error reading .env file")
  }

  cfg := Config{
    Port: config.Getenv("PORT"),
    KafkaBroker: config.Getenv("KAFKA_BROKER"),
  }

  lis, err := net.Listen("tcp", ":" + cfg.Port)
  if err != nil {
    log.Fatal(err)
  }

  grpcServer := grpc.NewServer()
  ingestionserver := server.NewIngestionServer(cfg.KafkaBroker)
  
  ingestionv1.RegisterEventIngestionServiceServer(grpcServer, ingestionserver)

  log.Printf("grpc server listening on port %s", cfg.Port)
  log.Fatal(grpcServer.Serve(lis))
}
