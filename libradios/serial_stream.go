package libradios

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"time"

	"github.com/jacobsa/go-serial/serial"
)

type SerialStream struct {
	serialPort io.ReadWriteCloser
}

func (s *SerialStream) CanRead() bool  { return true }
func (s *SerialStream) CanWrite() bool { return true }

func (s *SerialStream) Close() {
	if s.serialPort != nil {
		_ = s.serialPort.Close()
	}
}

func NewSerialStream(ctx context.Context, addr string) (*SerialStream, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	options := serial.OpenOptions{
		PortName:              addr,
		BaudRate:              115200,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       0,
		InterCharacterTimeout: 800,
		ParityMode:            serial.PARITY_NONE,
	}

	type result struct {
		port io.ReadWriteCloser
		err  error
	}

	ch := make(chan result, 1)

	go func() {
		p, err := serial.Open(options)
		ch <- result{port: p, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()

	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		return &SerialStream{serialPort: res.port}, nil
	}
}

func (s *SerialStream) Write(ctx context.Context, p []byte) error {
	written := 0

	for written < len(p) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if dl, ok := s.serialPort.(interface {
			SetWriteDeadline(time.Time) error
		}); ok {
			_ = dl.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
		}

		n, err := s.serialPort.Write(p[written:])
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return err
		}

		written += n
		time.Sleep(5 * time.Millisecond)
	}

	return nil
}

func (s *SerialStream) Read(ctx context.Context, p []byte) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	if dl, ok := s.serialPort.(interface {
		SetReadDeadline(time.Time) error
	}); ok {
		_ = dl.SetReadDeadline(
			time.Now().Add(8200 * time.Millisecond),
		)
	}

	return s.serialPort.Read(p)
}
