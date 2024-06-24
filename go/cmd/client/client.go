package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	st "github.com/NickCao/grpc-rendezvous/go/pkg/stream"
	pb "github.com/jumpstarter-dev/jumpstarter-protocol/go/jumpstarter/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

func RendezvousDialer(ctx context.Context, address string, token string) (net.Conn, error) {
	client, err := grpc.NewClient(
		"127.0.0.1:8000",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	router := pb.NewRouterServiceClient(client)

	resp, err := router.Dial(metadata.AppendToOutgoingContext(ctx,
		"authorization", fmt.Sprintf("Bearer %s", token)), &pb.DialRequest{
		Sub: address,
	})
	if err != nil {
		return nil, err
	}

	// stream is not tied to dial context
	ctx = context.Background()

	sc := pb.NewStreamServiceClient(client)

	stream, err := sc.Stream(metadata.AppendToOutgoingContext(ctx,
		"authorization", fmt.Sprintf("Bearer %s", resp.GetToken())))
	if err != nil {
		return nil, err
	}

	tx, rx := net.Pipe()

	go st.ForwardConn(ctx, stream, tx)

	return rx, nil
}

func main() {
	client, err := grpc.NewClient(
		"127.0.0.1:8000",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}

	controller := pb.NewControllerServiceClient(client)

	resp, err := controller.Register(context.TODO(), &pb.RegisterRequest{
		Uuid: "client",
	})
	if err != nil {
		log.Fatal(err)
	}

	exporterClient, err := grpc.NewClient(
		"unix:///exporter",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			addr := strings.TrimPrefix(s, "unix:///")
			return RendezvousDialer(ctx, addr, resp.GetRouterToken())
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	exporter := pb.NewExporterServiceClient(exporterClient)

	_, err = exporter.GetReport(context.TODO(), &emptypb.Empty{})
	log.Println(err)
}
