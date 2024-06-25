package main

import (
	"context"
	"log"
	"net"
	"strings"
	"time"

	st "github.com/NickCao/grpc-rendezvous/go/pkg/stream"
	"github.com/NickCao/grpc-rendezvous/go/pkg/token"
	"github.com/golang-jwt/jwt/v5"
	pb "github.com/jumpstarter-dev/jumpstarter-protocol/go/jumpstarter/v1"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/local"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/protobuf/types/known/emptypb"
)

type fakeUnixConn struct {
	net.Conn
}

func (c *fakeUnixConn) RemoteAddr() net.Addr {
	return &net.UnixAddr{
		Name: "dummy",
		Net:  "unix",
	}
}

func RendezvousDialer(ctx context.Context, address string, controller pb.ControllerServiceClient) (net.Conn, error) {
	resp, err := controller.Dial(ctx, &pb.DialRequest{
		Uuid: address,
	})
	if err != nil {
		return nil, err
	}

	client, err := grpc.NewClient(
		resp.GetRouterEndpoint(),
		grpc.WithTransportCredentials(local.NewCredentials()),
		grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: resp.GetRouterToken(),
		})}),
	)
	if err != nil {
		return nil, err
	}

	router := pb.NewRouterServiceClient(client)

	// stream is not tied to dial context
	stream, err := router.Stream(context.Background())
	if err != nil {
		return nil, err
	}

	tx, rx := net.Pipe()

	go st.ForwardConn(ctx, stream, tx)

	return &fakeUnixConn{Conn: rx}, nil
}

func main() {
	client, err := grpc.NewClient(
		"unix:/tmp/jumpstarter-controller.sock",
		grpc.WithTransportCredentials(local.NewCredentials()),
		grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token.Token("client-01", jwt.NewNumericDate(time.Now().Add(time.Hour))),
		})}),
	)
	if err != nil {
		log.Fatal(err)
	}

	controller := pb.NewControllerServiceClient(client)

	exporterClient, err := grpc.NewClient(
		"unix:///exporter-01",
		grpc.WithTransportCredentials(local.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			addr := strings.TrimPrefix(s, "unix:///")
			return RendezvousDialer(ctx, addr, controller)
		}),
	)

	if err != nil {
		log.Fatal(err)
	}

	exporter := pb.NewExporterServiceClient(exporterClient)

	_, err = exporter.GetReport(context.TODO(), &emptypb.Empty{})
	log.Println(err)
}
