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
	"google.golang.org/grpc/credentials/local"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RendezvousListener struct {
	address RendezvousAddr
	router  pb.RouterServiceClient
	listen  pb.ControllerService_ListenClient
	cancel  context.CancelFunc
}

type RendezvousAddr string

func NewRendezvousListener(ctx context.Context, controller pb.ControllerServiceClient) (*RendezvousListener, error) {
	ctx, cancel := context.WithCancel(ctx)

	listen, err := controller.Listen(ctx, &pb.ListenRequest{})
	if err != nil {
		cancel()
		return nil, err
	}

	return &RendezvousListener{
		address: RendezvousAddr("dummy"),
		listen:  listen,
		cancel:  cancel,
	}, nil
}

func (l *RendezvousListener) Accept() (net.Conn, error) {
	resp, err := l.listen.Recv()
	if err != nil {
		return nil, err
	}

	client, err := grpc.NewClient(resp.GetRouterEndpoint(),
		grpc.WithTransportCredentials(local.NewCredentials()),
		grpc.WithPerRPCCredentials(StaticCredential{
			"Authorization": fmt.Sprintf("Bearer %s", resp.RouterToken),
		}),
	)
	if err != nil {
		return nil, err
	}

	router := pb.NewRouterServiceClient(client)

	stream, err := router.Stream(context.TODO())
	if err != nil {
		return nil, err
	}

	tx, rx := net.Pipe()

	go st.ForwardConn(l.listen.Context(), stream, tx)

	return rx, nil
}

func (l *RendezvousListener) Close() error {
	l.cancel()
	return nil
}

func (l *RendezvousListener) Addr() net.Addr {
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

type StaticCredential map[string]string

func (c StaticCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return c, nil
}
func (c StaticCredential) RequireTransportSecurity() bool {
	return false
}

func main() {
	client, err := grpc.NewClient(
		"127.0.0.1:8082",
		grpc.WithTransportCredentials(local.NewCredentials()),
		grpc.WithPerRPCCredentials(StaticCredential{
			"namespace": "default",
			"name":      "exporter-sample",
			"token":     "54d8cd395728888be9fcb93c4575d99e",
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	controller := pb.NewControllerServiceClient(client)

	log.Println(controller.Register(
		context.Background(),
		&pb.RegisterRequest{},
	))

	log.Println(controller.Bye(
		context.Background(),
		&pb.ByeRequest{Reason: "maintenance"},
	))

	listen, err := NewRendezvousListener(context.TODO(), controller)
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
