package main

import (
	"context"
	"log"
	"net"

	st "github.com/NickCao/grpc-rendezvous/pkg/stream"
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
	client  pb.RendezvousServiceClient
	listen  pb.RendezvousService_ListenClient
	cancel  context.CancelFunc
}

type RendezvousAddr string

func NewRendezvousListner(ctx context.Context, address string) (*RendezvousListner, error) {
	client, err := grpc.NewClient(
		"127.0.0.1:8000",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	rendezvous := pb.NewRendezvousServiceClient(client)

	ctx, cancel := context.WithCancel(ctx)

	listen, err := rendezvous.Listen(ctx, &pb.ListenRequest{
		Address: address,
	})
	if err != nil {
		cancel()
		return nil, err
	}

	return &RendezvousListner{
		address: RendezvousAddr(address),
		client:  rendezvous,
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

	stream, err := l.client.Stream(metadata.AppendToOutgoingContext(ctx, "stream", resp.Stream))
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

type ForClientServer struct {
	pb.UnimplementedForClientServer
}

func (c *ForClientServer) GetReport(context.Context, *emptypb.Empty) (*pb.ExporterReport, error) {
	return nil, status.Errorf(codes.Internal, "dummy implementation")
}

func main() {
	listen, err := NewRendezvousListner(context.Background(), "unix:///dummy")
	if err != nil {
		log.Fatal(err)
	}

	server := grpc.NewServer()

	pb.RegisterForClientServer(server, &ForClientServer{})

	err = server.Serve(listen)
	if err != nil {
		log.Fatal(err)
	}
}
