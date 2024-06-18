package stream

import (
	"context"
	"errors"
	"io"
	"net"

	pb "github.com/jumpstarter-dev/jumpstarter-protocol/go/jumpstarter/v1"
	"golang.org/x/sync/errgroup"
)

type Stream[T any] interface {
	Send(T) error
	Recv() (T, error)
}

func pipe[T any, A Stream[T], B Stream[T]](a A, b B) error {
	for {
		msg, err := a.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		err = b.Send(msg)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func Forward[T any, A Stream[T], B Stream[T]](ctx context.Context, a A, b B) error {
	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error { return pipe(a, b) })
	g.Go(func() error { return pipe(b, a) })
	return g.Wait()
}

func ForwardConn[T Stream[*pb.Frame]](ctx context.Context, stream T, conn net.Conn) error {
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
			err = stream.Send(&pb.Frame{
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
