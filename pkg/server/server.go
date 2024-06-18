package server

import (
	"context"
	"sync"

	st "github.com/NickCao/grpc-rendezvous/pkg/stream"
	pb "github.com/NickCao/grpc-rendezvous/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type RendezvousServer struct {
	pb.UnimplementedRendezvousServer
	listenMap *sync.Map
	streamMap *sync.Map
}

type listenCtx struct {
	cancel context.CancelFunc
	stream pb.Rendezvous_ListenServer
}

type streamCtx struct {
	cancel context.CancelFunc
	stream pb.Rendezvous_StreamServer
}

func NewRendezvousServer() *RendezvousServer {
	return &RendezvousServer{
		listenMap: &sync.Map{},
		streamMap: &sync.Map{},
	}
}

func (s *RendezvousServer) Listen(req *pb.Request, stream pb.Rendezvous_ListenServer) error {
	ctx, cancel := context.WithCancel(stream.Context())

	cond := listenCtx{
		cancel: cancel,
		stream: stream,
	}

	actual, loaded := s.listenMap.Swap(req.Address, cond)
	if loaded {
		actual.(listenCtx).cancel()
	}

	select {
	case <-ctx.Done():
		return nil
	}
}

func (s *RendezvousServer) Dial(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	value, ok := s.listenMap.Load(req.Address)
	if !ok {
		return nil, status.Errorf(codes.Unavailable, "no matching listener")
	}

	resp := &pb.Response{
		Stream: uuid.New().String(),
	}

	err := value.(listenCtx).stream.Send(resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
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
	actual, loaded := s.streamMap.LoadOrStore(id[0], cond)
	if loaded {
		defer actual.(streamCtx).cancel()
		return st.Forward(ctx, stream, actual.(streamCtx).stream)
	}

	select {
	case <-ctx.Done():
		return nil
	}
}
