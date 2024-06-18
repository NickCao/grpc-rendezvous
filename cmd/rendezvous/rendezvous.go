package main

import (
	"log"
	"net"

	"github.com/NickCao/grpc-rendezvous/pkg/server"
	pb "github.com/NickCao/grpc-rendezvous/proto"
	"google.golang.org/grpc"
)

func main() {
	listen, err := net.Listen("tcp", "127.0.0.1:8000")
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()

	pb.RegisterRendezvousServer(s, server.NewRendezvousServer())

	err = s.Serve(listen)
	if err != nil {
		log.Fatal(err)
	}
}
