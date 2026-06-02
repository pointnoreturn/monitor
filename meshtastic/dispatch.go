package meshtastic

import (
	"context"
	"errors"
	"fmt"
	"time"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
)

// default interval for periodic heartbeats
// sending a heartbeat is the best way to detect the connection was broken and failed.
const heartbeatInterval = time.Second * 15

// default timeout for "should have received anything"
const receiveTimeout = time.Minute * 5

var (
	// checks socket is alive and radio is active it will fail if not for some period
	ErrReceiveTimeout = errors.New("Waiting for packet receive timed         out")

	// A proto stream may be receive only
	ErrReadonly = errors.New("ProtoStream is not CanWrite()")
)

// high level send and receive packets with multiple handlers and keepalive (heartbeats)
type Dispatch struct {
	Writer       // WritePacket, queued send packets
	Reader       // ReadPacket waits for worker to encounter a packet, as a proxy
	stream       *ProtoStream
	HandlePacket PacketF
	recvTimeout  time.Duration
	sendQueue    chan *pb.ToRadio   // WritePackets synchronous  implemented as a buffered channel consumed by Run()
	recvQueue    chan *pb.FromRadio // ReadPackets synchronous implemeted as a waiter for channel populated in Run() if someone is waiting
}

// make a new dispatch on the compatible connection (one per connection)
func NewDispatch(stream *ProtoStream, sendBuffer int, handler PacketF) *Dispatch {
	return &Dispatch{
		stream:       stream,
		HandlePacket: handler,
		recvTimeout:  receiveTimeout,
		sendQueue:    make(chan *pb.ToRadio, sendBuffer),
		recvQueue:    make(chan *pb.FromRadio),
	}
}

// send and receive packets worker
func (dispatch *Dispatch) Run(ctx context.Context) error {
	var heartbeats uint32 = 0
	keepAlive := time.NewTicker(heartbeatInterval)
	defer keepAlive.Stop()

	log := dispatch.stream.Log

	log.Debug("[Dispatch] Running")

	lastPacket := time.Now()

	for {
		select {

		// context cancellation
		case <-ctx.Done():
			return ctx.Err()

		// send periodic hearbeats
		case <-keepAlive.C:
			heartbeats += 1
			err := Heartbeat(ctx, dispatch.stream, heartbeats)
			if err != nil {
				log.Error(fmt.Sprintf("[Dispatch] Heartbeat write failed with Err %T %v", err, err))
				return err
			}

		// packets queued to send
		case p := <-dispatch.sendQueue:
			err := dispatch.stream.WritePacket(ctx, p)
			if err != nil {
				log.Error(fmt.Sprintf("Dispatch] WritePacket queued in Dispatch failed with Err %T %v", err, err))
				return err
			}

		// receive packets
		default:
			packets, err := dispatch.stream.ReadPackets(ctx, true)
			if err != nil {
				log.Error(fmt.Sprintf("ReadPackets failed with Err %T %v", err, err))
				return err
			}

			if len(packets) == 0 { // no packets received
				if time.Since(lastPacket) > dispatch.recvTimeout { // last packet was too long ago
					log.Error("[Dispatch] receive timed out")
					return ErrReceiveTimeout
				}
			} else {
				for _, p := range packets {

					// send packet for synchronious ReadPacket. If there is no receiver, ignore
					select {
					case dispatch.recvQueue <- p:
					default:
					}

					dispatch.HandlePacket(p)
					lastPacket = time.Now()
				}
			}
		}
	}
}

// queue packet for sending
func (dispatch *Dispatch) WritePacket(ctx context.Context, p *pb.ToRadio) error {
	if !dispatch.stream.CanWrite() {
		return ErrReadonly
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case dispatch.sendQueue <- p:
		return nil
	}
}

// func (dispatch *Dispatch) ReadPackets(ctx context.Context, timeout bool) (packets []*pb.FromRadio, err error) {
// 	if !timeout {
// 		select {
// 		case <-ctx.Done():
// 			return nil, ctx.Err()
// 		case packet := <-dispatch.recvQueue:
// 			return []*pb.FromRadio{packet}, nil
// 		}
// 	} else {
// 		t := time.NewTicker(time.Second * 5)
// 		packets := []*pb.FromRadio{}
// 		select {
// 		case <-ctx.Done():
// 			return nil, ctx.Err()
// 		case p := <-dispatch.recvQueue:
// 			packets = append(packets, p)
// 		case <-t.C:
// 			return packets, nil
// 		}
// 		return packets, nil
// 	}
// }
