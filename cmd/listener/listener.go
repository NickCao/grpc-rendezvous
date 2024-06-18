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
)

func main() {
	c, err := grpc.NewClient("127.0.0.1:8000",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	client := pb.NewRendezvousServiceClient(c)

	listen, err := client.Listen(context.TODO(), &pb.ListenRequest{
		Address: "dummy",
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("listening on address dummy")

	for {
		resp, err := listen.Recv()
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("new incoming connection: %s\n", resp.Stream)

		ctx := metadata.NewOutgoingContext(context.TODO(), metadata.Pairs("stream", resp.Stream))

		stream, err := client.Stream(ctx)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("established connection: %s\n", resp.Stream)

		conn, err := net.Dial("tcp", "127.0.0.1:5201") // iperf3
		if err != nil {
			log.Fatal(err)
		}

		go st.ForwardConn(ctx, stream, conn)
	}
}
