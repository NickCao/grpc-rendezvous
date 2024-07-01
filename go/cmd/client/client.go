package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	st "github.com/NickCao/grpc-rendezvous/go/pkg/stream"
	pb "github.com/jumpstarter-dev/jumpstarter-protocol/go/jumpstarter/v1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/local"
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

type StaticCredential map[string]string

func (c StaticCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return c, nil
}
func (c StaticCredential) RequireTransportSecurity() bool {
	return false
}

func RendezvousDialer(ctx context.Context, address string, device *string, controller pb.ControllerServiceClient) (net.Conn, error) {
	resp, err := controller.Dial(ctx, &pb.DialRequest{
		Uuid:       address,
		DeviceUuid: device,
	})
	if err != nil {
		return nil, err
	}

	client, err := grpc.NewClient(
		resp.GetRouterEndpoint(),
		grpc.WithTransportCredentials(local.NewCredentials()),
		grpc.WithPerRPCCredentials(StaticCredential{
			"Authorization": fmt.Sprintf("Bearer %s", resp.RouterToken),
		}),
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
		"127.0.0.1:8082",
		grpc.WithTransportCredentials(local.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}

	controller := pb.NewControllerServiceClient(client)

	exporterClient, err := grpc.NewClient(
		"unix:///exporter-sample",
		grpc.WithTransportCredentials(local.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			addr := strings.TrimPrefix(s, "unix:///")
			return RendezvousDialer(ctx, addr, nil, controller)
		}),
	)

	if err != nil {
		log.Fatal(err)
	}

	exporter := pb.NewExporterServiceClient(exporterClient)

	_, err = exporter.GetReport(context.TODO(), &emptypb.Empty{})
	log.Println(err)

	deviceUuid := "somedevice"
	conn, err := RendezvousDialer(context.Background(), "exporter-sample", &deviceUuid, controller)
	if err != nil {
		log.Fatal(err)
	}

	g := new(errgroup.Group)
	g.Go(func() error {
		_, err := io.Copy(os.Stdout, conn)
		return err
	})
	g.Go(func() error {
		_, err := io.Copy(conn, os.Stdin)
		return err
	})
	g.Wait()
}
