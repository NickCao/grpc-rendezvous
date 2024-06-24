package main

import (
	"context"
	"fmt"
	"log"
	"net"

	st "github.com/NickCao/grpc-rendezvous/go/pkg/stream"
	pb "github.com/jumpstarter-dev/jumpstarter-protocol/go/jumpstarter/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RendezvousListner struct {
	address RendezvousAddr
	router  pb.RouterServiceClient
	stream  pb.StreamServiceClient
	listen  pb.RouterService_ListenClient
	cancel  context.CancelFunc
}

type RendezvousAddr string

func NewRendezvousListner(ctx context.Context, token string) (*RendezvousListner, error) {
	client, err := grpc.NewClient(
		"127.0.0.1:8000",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	rendezvous := pb.NewRouterServiceClient(client)

	ctx, cancel := context.WithCancel(ctx)

	listen, err := rendezvous.Listen(metadata.AppendToOutgoingContext(ctx,
		"authorization", fmt.Sprintf("Bearer %s", token)), &pb.ListenRequest{})
	if err != nil {
		cancel()
		return nil, err
	}

	return &RendezvousListner{
		address: RendezvousAddr("rendezvous"),
		stream:  pb.NewStreamServiceClient(client),
		router:  rendezvous,
		listen:  listen,
		cancel:  cancel,
	}, nil
}

func (l *RendezvousListner) Accept() (net.Conn, error) {
	resp, err := l.listen.Recv()
	if err != nil {
		return nil, err
	}

	ctx := l.listen.Context()

	stream, err := l.stream.Stream(metadata.AppendToOutgoingContext(context.TODO(), // FIXME: drop authorization context
		"authorization", fmt.Sprintf("Bearer %s", resp.GetToken())))
	if err != nil {
		return nil, err
	}

	tx, rx := net.Pipe()

	go st.ForwardConn(ctx, stream, tx)

	return rx, nil
}

func (l *RendezvousListner) Close() error {
	l.cancel()
	return nil
}

func (l *RendezvousListner) Addr() net.Addr {
	return l.address
}

func (a RendezvousAddr) Network() string {
	return "rendezvous"
}

func (a RendezvousAddr) String() string {
	return string(a)
}

type ExporterServer struct {
	pb.UnimplementedExporterServiceServer
}

func (c *ExporterServer) GetReport(context.Context, *emptypb.Empty) (*pb.GetReportResponse, error) {
	return nil, status.Errorf(codes.Internal, "dummy implementation in go")
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
		Uuid: "exporter",
	})

	listen, err := NewRendezvousListner(context.Background(), resp.GetRouterToken())
	if err != nil {
		log.Fatal(err)
	}

	server := grpc.NewServer()

	pb.RegisterExporterServiceServer(server, &ExporterServer{})

	err = server.Serve(listen)
	if err != nil {
		log.Fatal(err)
	}
}
