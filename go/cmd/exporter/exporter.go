package main

import (
	"context"
	"log"
	"net"
	"time"

	st "github.com/NickCao/grpc-rendezvous/go/pkg/stream"
	"github.com/NickCao/grpc-rendezvous/go/pkg/token"
	"github.com/golang-jwt/jwt/v5"
	pb "github.com/jumpstarter-dev/jumpstarter-protocol/go/jumpstarter/v1"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/local"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"
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
		grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: resp.GetRouterToken(),
		})}),
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

func main() {
	client, err := grpc.NewClient(
		"unix:/tmp/jumpstarter-controller.sock",
		grpc.WithTransportCredentials(local.NewCredentials()),
		grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token.Token("exporter-01", jwt.NewNumericDate(time.Now().Add(time.Hour))),
		})}),
	)
	if err != nil {
		log.Fatal(err)
	}

	controller := pb.NewControllerServiceClient(client)

	log.Println(controller.Register(
		metadata.AppendToOutgoingContext(
			context.Background(),
			"namespace", "default",
			"name", "exporter-01",
			"token", "supersecret",
		),
		&pb.RegisterRequest{},
	))

}
