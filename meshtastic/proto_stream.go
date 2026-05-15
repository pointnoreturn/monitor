package meshtastic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libradios"
	"google.golang.org/protobuf/proto"
)

// TODO: meshtastic stuff here on protoStream primitive?
const start1 = byte(0x94)
const start2 = byte(0xc3)
const headerLen = 4
const maxToFromRadioSzie = 512

// read and write Meshtastic Protobuf packets on the underrelying Stream using magic byte codings
type ProtoStream struct {
	libradios.Transport
	libradios.Writer[*pb.ToRadio]
	libradios.Reader[*pb.FromRadio]
	Log *slog.Logger
}

func (stream *ProtoStream) WritePacket(
	ctx context.Context,
	p *pb.ToRadio,
) error {

	protobufPacket, err := proto.Marshal(p)
	if err != nil {
		return err
	}

	packageLength := len(protobufPacket)
	stream.Log.Debug(fmt.Sprintf("[WritePacket] %d bytes %T\n", packageLength, stream.Transport))

	header := []byte{
		start1,
		start2,
		byte(packageLength>>8) & 0xff,
		byte(packageLength) & 0xff,
	}

	data := append(header, protobufPacket...)

	return stream.Write(ctx, data)
}

func (stream *ProtoStream) ReadPackets(ctx context.Context, timeout bool) ([]*pb.FromRadio, error) {
	stream.Log.Debug(fmt.Sprintf("[ReadPackets] timeout %v on %T\n", timeout, stream.Transport))

	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	b := make([]byte, 1)

	var (
		processed []byte
		packets   []*pb.FromRadio
	)

	for {

		n, err := stream.Read(readCtx, b)

		// -------------------------
		// ONLY REAL FATAL ERRORS
		// -------------------------

		if err != nil {

			if errors.Is(err, context.Canceled) {
				return packets, err
			}

			if errors.Is(err, context.DeadlineExceeded) {
				return packets, nil
			}

			if err == io.EOF {
				return packets, nil
			}

			// IMPORTANT: ignore transport timeouts
			continue
		}

		if n <= 0 {
			continue
		}

		c := b[0]

		processed = append(processed, c)

		// -------------------------
		// resync header
		// -------------------------

		if len(processed) == 1 && c != start1 {
			processed = processed[:0]
			continue
		}

		if len(processed) == 2 && c != start2 {
			processed = processed[:0]
			continue
		}

		// -------------------------
		// need header first
		// -------------------------

		if len(processed) < headerLen {
			continue
		}

		length := int(processed[2])<<8 | int(processed[3])

		if length > maxToFromRadioSzie {
			processed = processed[:0]
			continue
		}

		// -------------------------
		// full packet ready
		// -------------------------

		if len(processed) == headerLen+length {

			var fr pb.FromRadio

			if err := proto.Unmarshal(
				processed[headerLen:],
				&fr,
			); err != nil {
				return nil, err
			}

			packets = append(packets, &fr)
			processed = processed[:0]
		}
	}

	return packets, nil
}
