package main

import (
	"context"
	"log"
	"net"

	st "github.com/NickCao/grpc-rendezvous/pkg/stream"
	pb "github.com/NickCao/grpc-rendezvous/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func main() {
	listen, err := net.Listen("tcp", "127.0.0.1:8001")
	if err != nil {
		log.Fatal(err)
	}

	c, err := grpc.NewClient("127.0.0.1:8000",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	client := pb.NewRendezvousClient(c)

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatal(err)
		}

		resp, err := client.Dial(context.TODO(), &pb.Request{
			Address: "dummy",
		})
		if err != nil {
			log.Fatal(err)
		}

		ctx := metadata.NewOutgoingContext(context.TODO(), metadata.Pairs("stream", resp.Stream))

		stream, err := client.Stream(ctx)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("established connection: %s\n", resp.Stream)

		go st.ForwardConn(ctx, stream, conn)
	}
}
