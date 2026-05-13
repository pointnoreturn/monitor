package meshtastic

import (
	"context"
	"errors"
	"fmt"
	"time"

	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
)

// default interval for periodic heartbeats
// sending a heartbeat is the best way to detect the connection was broken and failed.
const defaultHeartbeatInterval = 40 * time.Second

// default value for sniff timeout (receiver idle)
const defaultSniffTimeout = time.Minute * 2

var (
	// checks socket is alive and radio is active it will fail if not for some period
	ErrSniffTimeout = errors.New("Waiting for packet receive timed out")
)

type PacketF func(*pb.FromRadio)

type Dispatch struct {
	stream       *ProtoStream
	sniffers     []PacketF
	sniffTimeout time.Duration
}

func NewDispatch(stream *ProtoStream, receivers []PacketF) *Dispatch {
	return &Dispatch{
		stream:       stream,
		sniffers:     receivers,
		sniffTimeout: defaultSniffTimeout,
	}
}

func (d *Dispatch) Run(ctx context.Context) error {
	var heartbeats uint32 = 0
	keepAlive := time.NewTicker(defaultHeartbeatInterval)
	defer keepAlive.Stop()

	fmt.Println("Telemeter loop is running")

	lastPacket := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-keepAlive.C:
			heartbeats += 1
			if err := d.stream.SendHeartbeat(ctx, heartbeats); err != nil {
				fmt.Printf("Heartbeat write failed with Err %T\n", err)
				return err
			}
		default:
			packets, err := d.stream.ReadPackets(ctx, true)
			if err != nil {
				fmt.Printf("ReadPackets failed with Err %T\n", err)
				return err
			}

			if len(packets) == 0 { // no packets received
				if time.Since(lastPacket) > d.sniffTimeout { // last packet was too long ago
					return ErrSniffTimeout
				}
			}

			for _, p := range packets {
				for _, h := range d.sniffers {
					h(p)
				}
				lastPacket = time.Now()
			}
		}
	}
}
