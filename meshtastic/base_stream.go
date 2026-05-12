package meshtastic

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"time"

	"github.com/jacobsa/go-serial/serial"
	"github.com/pointnoreturn/snake/libsnake"
)

type baseStream struct {
	libsnake.Transport
	serialPort io.ReadWriteCloser
	netPort    net.Conn
	isTCP      bool
}

func (s *baseStream) CanRead() bool { return true }

func (s *baseStream) CanWrite() bool { return true }

func (s *baseStream) Close() {
	if s.isTCP {
		if s.netPort != nil {
			s.netPort.Close()
		}
	} else {
		if s.serialPort != nil {
			s.serialPort.Close()
		}
	}
}

func (s *baseStream) Init(
	ctx context.Context,
	addr string,
	defaultPort string,
) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if tcpAddr, ok := ParseTCPAddress(addr, defaultPort); ok {

		d := net.Dialer{}

		client, err := d.DialContext(
			ctx,
			"tcp",
			tcpAddr,
		)
		if err != nil {
			return err
		}

		s.netPort = client
		s.isTCP = true

		return nil
	}

	// serial path

	options := serial.OpenOptions{
		PortName:              addr,
		BaudRate:              115200,
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       0,
		InterCharacterTimeout: 100,
		ParityMode:            serial.PARITY_NONE,
	}

	type result struct {
		port io.ReadWriteCloser
		err  error
	}

	ch := make(chan result, 1)

	go func() {
		p, err := serial.Open(options)
		ch <- result{
			port: p,
			err:  err,
		}
	}()

	select {

	case <-ctx.Done():
		return ctx.Err()

	case res := <-ch:

		if res.err != nil {
			return res.err
		}

		s.serialPort = res.port
		s.isTCP = false

		return nil
	}
}

func (s *baseStream) Write(
	ctx context.Context,
	p []byte,
) error {

	written := 0

	for written < len(p) {

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if s.isTCP {

			err := s.netPort.SetWriteDeadline(
				time.Now().Add(200 * time.Millisecond),
			)
			if err != nil {
				return err
			}

			n, err := s.netPort.Write(p[written:])

			if err != nil {

				if errors.Is(err, os.ErrDeadlineExceeded) {
					continue
				}

				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}

				return err
			}

			written += n

		} else {

			// serial libraries vary a lot
			if dl, ok := s.serialPort.(interface {
				SetWriteDeadline(time.Time) error
			}); ok {
				_ = dl.SetWriteDeadline(
					time.Now().Add(200 * time.Millisecond),
				)
			}

			n, err := s.serialPort.Write(p[written:])

			if err != nil {

				if errors.Is(err, os.ErrDeadlineExceeded) {
					continue
				}

				return err
			}

			written += n

			// optional throttling for serial radios
			time.Sleep(5 * time.Millisecond)
		}
	}

	return nil
}

func (s *baseStream) Read(ctx context.Context, p []byte) error {

	for {

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if s.isTCP {

			// short polling deadline
			s.netPort.SetReadDeadline(time.Now().Add(200 * time.Millisecond))

			_, err := s.netPort.Read(p)

			if err != nil {

				// timeout is expected, continue polling
				if errors.Is(err, os.ErrDeadlineExceeded) {
					continue
				}

				netErr, ok := err.(net.Error)
				if ok && netErr.Timeout() {
					continue
				}

				return err
			}

			return nil

		} else {

			// serial ports usually support deadlines too
			if dl, ok := s.serialPort.(interface {
				SetReadDeadline(time.Time) error
			}); ok {
				dl.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			}

			_, err := s.serialPort.Read(p)

			if err != nil {

				if errors.Is(err, os.ErrDeadlineExceeded) {
					continue
				}

				return err
			}

			return nil
		}
	}
}
