package meshtastic

import (
	"context"
	"errors"
	"fmt"
	"time"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libradios"
)

// default interval for periodic heartbeats
// sending a heartbeat is the best way to detect the connection was broken and failed.
const defaultHeartbeatInterval = 60 * time.Second

// default value for sniff timeout (receiver idle)
const defaultSniffTimeout = time.Minute * 2

var (
	// checks socket is alive and radio is active it will fail if not for some period
	ErrSniffTimeout = errors.New("Waiting for packet receive timed         out")

	// A proto stream may be receive only
	ErrReadonly = errors.New("ProtoStream is not CanWrite()")
)

type Dispatch struct {
	libradios.Writer[*pb.ToRadio]
	stream           *ProtoStream
	HandlePacket     PacketF
	sniffTimeout     time.Duration
	sendPacketsQueue chan *pb.ToRadio
}

func NewDispatch(stream *ProtoStream, sendBuffer int, handler PacketF) *Dispatch {
	return &Dispatch{
		stream:           stream,
		HandlePacket:     handler,
		sniffTimeout:     defaultSniffTimeout,
		sendPacketsQueue: make(chan *pb.ToRadio, sendBuffer),
	}
}

// queue packets for sending
func (d *Dispatch) SendPacket(p *pb.ToRadio) error {
	if !d.stream.CanWrite() {
		return ErrReadonly
	}
	d.sendPacketsQueue <- p
	return nil
}

func (d *Dispatch) Run(ctx context.Context) error {
	var heartbeats uint32 = 0
	keepAlive := time.NewTicker(defaultHeartbeatInterval)
	defer keepAlive.Stop()

	fmt.Println("[Dispatch] Running")

	lastPacket := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-keepAlive.C:
			heartbeats += 1
			err := d.stream.SendHeartbeat(ctx, heartbeats)
			if err != nil {
				fmt.Printf("Heartbeat write failed with Err %T\n", err)
				return err
			}
		case p := <-d.sendPacketsQueue:
			err := d.stream.WritePacket(ctx, p)
			if err != nil {
				fmt.Printf("WritePacket queued in Dispatch failed with Err %T\n", err)
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
				d.HandlePacket(p)
				lastPacket = time.Now()
			}
		}
	}
}
