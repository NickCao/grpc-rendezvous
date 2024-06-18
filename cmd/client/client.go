package main

import (
	"context"
	"log"
	"net"

	st "github.com/NickCao/grpc-rendezvous/pkg/stream"
	pb "github.com/jumpstarter-dev/jumpstarter-protocol/go/jumpstarter/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func RendezvousDialer(ctx context.Context, address string) (net.Conn, error) {
	client, err := grpc.NewClient(
		"127.0.0.1:8000",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	rendezvous := pb.NewRendezvousServiceClient(client)

	resp, err := rendezvous.Dial(ctx, &pb.DialRequest{
		Address: address,
	})
	if err != nil {
		return nil, err
	}

	stream, err := rendezvous.Stream(metadata.AppendToOutgoingContext(ctx, "stream", resp.Stream))
	if err != nil {
		return nil, err
	}

	tx, rx := net.Pipe()

	go st.ForwardConn(ctx, stream, tx)

	return rx, nil
}

func main() {
	client, err := grpc.NewClient(
		"dummy",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(RendezvousDialer),
	)
	if err != nil {
		log.Fatal(err)
	}

	jumpstarter := pb.NewForClientClient(client)

	_, err = jumpstarter.GetReport(context.TODO(), &emptypb.Empty{})
	if err != nil {
		log.Fatal(err)
	}
}
