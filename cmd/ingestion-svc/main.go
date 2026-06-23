package main

import (
	"context"
	"log"
	"net"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"

	ingestionv1 "github.com/kaizakin/siphon/gen/ingestion/v1"
	"github.com/kaizakin/siphon/internal/ingestion/sqlc"
	grpcserver "github.com/kaizakin/siphon/internal/ingestion/grpc"
	server "github.com/kaizakin/siphon/internal/ingestion/grpc"
	"github.com/kaizakin/siphon/pkg/config"
)

type Config struct {
  Port string
  KafkaBroker string
  Db_Url string
}

func main() {
  err := godotenv.Load()
  if err != nil {
    log.Fatal("Error reading .env file")
  }

  cfg := Config{
    Port: config.Getenv("PORT"),
    KafkaBroker: config.Getenv("KAFKA_BROKER"),
    Db_Url: config.Getenv("DATABASE_URL"),
  }

  pool, err := pgxpool.New(context.Background(), cfg.Db_Url)
  if err != nil {
    log.Fatal(err)
  }
  defer pool.Close()

  queries := sqlc.New(pool)

  handler := grpcserver.NewpgxHandler(queries)

  lis, err := net.Listen("tcp", ":" + cfg.Port)
  if err != nil {
    log.Fatal(err)
  }

  grpcServer := grpc.NewServer()
  ingestionserver := server.NewIngestionServer(cfg.KafkaBroker, handler.Queries)
  
  ingestionv1.RegisterEventIngestionServiceServer(grpcServer, ingestionserver)

  log.Printf("grpc server listening on port %s", cfg.Port)
  log.Fatal(grpcServer.Serve(lis))
}
