package server

import (
	"context"
	"sync"

	st "github.com/NickCao/grpc-rendezvous/pkg/stream"
	pb "github.com/NickCao/grpc-rendezvous/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type RendezvousServer struct {
	pb.UnimplementedRendezvousServer
	streamMap *sync.Map
}

type streamCtx struct {
	cancel context.CancelFunc
	stream pb.Rendezvous_StreamServer
}

func (s *RendezvousServer) Stream(stream pb.Rendezvous_StreamServer) error {
	ctx := stream.Context()

	// extract connection id from context
	md, loaded := metadata.FromIncomingContext(ctx)
	if !loaded {
		return status.Errorf(codes.InvalidArgument, "missing context")
	}

	id := md.Get("stream")
	if len(id) != 1 {
		return status.Errorf(codes.InvalidArgument, "missing stream id in context")
	}

	// create new context for stream
	ctx, cancel := context.WithCancel(ctx)

	cond := streamCtx{
		cancel: cancel,
		stream: stream,
	}

	// find stream with matching id
	actual, loaded := s.streamMap.LoadOrStore(id, cond)
	if loaded {
		defer actual.(streamCtx).cancel()
		return st.Forward(ctx, stream, actual.(streamCtx).stream)
	}

	select {
	case <-ctx.Done():
		return nil
	}
}
