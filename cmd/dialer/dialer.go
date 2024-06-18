package main

import (
	"context"
	"log"

	pb "github.com/NickCao/grpc-rendezvous/proto"
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

	client := pb.NewRendezvousClient(c)

	conn, err := client.Dial(context.TODO(), &pb.Request{
		Address: "dummy",
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx := metadata.NewOutgoingContext(context.TODO(), metadata.Pairs("stream", conn.Stream))

	_, err = client.Stream(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
