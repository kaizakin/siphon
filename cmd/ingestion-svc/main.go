package main

import (
	"context"
	"log"
	"net"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"

	ingestionv1 "github.com/kaizakin/siphon/gen/ingestion/v1"
	"github.com/kaizakin/siphon/internal/ingestion/dlq"
	grpcserver "github.com/kaizakin/siphon/internal/ingestion/grpc"
	"github.com/kaizakin/siphon/internal/ingestion/sqlc"
	"github.com/kaizakin/siphon/pkg/config"
)

type Config struct {
	Port        string
	KafkaBroker string
	Db_Url      string
}

func main() {
	_ = godotenv.Load()

	cfg := Config{
		Port:        config.Getenv("PORT"),
		KafkaBroker: config.Getenv("KAFKA_BROKER"),
		Db_Url:      config.Getenv("DATABASE_URL"),
	}

	pool, err := pgxpool.New(context.Background(), cfg.Db_Url)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	queries := sqlc.New(pool)

	writer := &kafka.Writer{
		Addr:     kafka.TCP(cfg.KafkaBroker),
		Topic:    "events",
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	// Launch background DLQ retry worker with graceful cancellation context
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()

	retryWorker := dlq.NewRetryWorker(queries, writer)
	go retryWorker.Start(workerCtx)

	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()
	ingestionserver := grpcserver.NewIngestionServer(queries, writer, 100, 4)

	ingestionv1.RegisterEventIngestionServiceServer(grpcServer, ingestionserver)

	log.Printf("grpc server listening on port %s", cfg.Port)
	log.Fatal(grpcServer.Serve(lis))
}
