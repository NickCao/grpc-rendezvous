package stream

import (
	"context"
	"errors"
	"io"
	"net"

	pb "github.com/jumpstarter-dev/jumpstarter-protocol/go/jumpstarter/v1"
	"golang.org/x/sync/errgroup"
)

func ForwardConn(ctx context.Context, stream pb.RouterService_StreamClient, conn net.Conn) error {
	g, _ := errgroup.WithContext(ctx)

	g.Go(func() error {
		buffer := make([]byte, 4096)
		for seq := uint64(0); ; seq++ {
			n, err := conn.Read(buffer)
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			err = stream.Send(&pb.StreamRequest{
				Payload: buffer[:n],
			})
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
		}
	})

	g.Go(func() error {
		for {
			frame, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
			_, err = conn.Write(frame.Payload)
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return err
			}
		}
	})

	return g.Wait()
}
