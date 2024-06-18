package main

import (
	"log"
	"net"

	"github.com/NickCao/grpc-rendezvous/go/pkg/server"
	pb "github.com/jumpstarter-dev/jumpstarter-protocol/go/jumpstarter/v1"
	"google.golang.org/grpc"
)

func main() {
	listen, err := net.Listen("tcp", "127.0.0.1:8000")
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()

	pb.RegisterRendezvousServiceServer(s, server.NewRendezvousServer())

	err = s.Serve(listen)
	if err != nil {
		log.Fatal(err)
	}
}
