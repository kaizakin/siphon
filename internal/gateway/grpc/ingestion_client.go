package grpcclient

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/kaizakin/siphon/gen/ingestion/v1"
)

func NewIngestionclient(addr string) (pb.EventIngestionServiceClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return pb.NewEventIngestionServiceClient(conn), nil
}
