package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libsupport"
	"github.com/pointnoreturn/monitor/meshtastic"

	// This blank import triggers the automatic loading of .env
	_ "github.com/joho/godotenv/autoload"
)

var (
	targetNode              = os.Getenv("TARGET_NODE")
	log, _     *slog.Logger = libsupport.LoggersFromEnv()
	dispatch   *meshtastic.Dispatch
	myNodeInfo *pb.MyNodeInfo
	nodeInfo   *pb.NodeInfo
)

func init() {
	if len(targetNode) == 0 {
		panic("TARGET_NODE is empty")
	}
}

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)
	defer stop()

	var (
		err          error
		stream       *meshtastic.ProtoStream
		handlePacket meshtastic.PacketF = meshtastic.ChainPacketHandlers(
			PingBot,
		)
	)

	// Using ConfigId_ConfigOnly to omit full NodeDB sync
	stream, myNodeInfo, nodeInfo, err = meshtastic.FindAndConnect(ctx, log, targetNode, time.Second*5, meshtastic.ConfigId_ConfigOnly, handlePacket)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			panic("Cannot find target node to connect: " + targetNode)
		}
		log.Error("Connection failed", "err", err)
		panic(err)
	}
	defer stream.Close()

	// create dispatch,
	// packet send/receive abstraction with event loop for meshtastic protocol handling,
	// on top of the stream
	dispatch = meshtastic.NewDispatch(stream, 100, handlePacket)

	// Dispatch runs till context dies
	log.Info(fmt.Sprintf("Pingbot connected and running on node !%x", myNodeInfo.MyNodeNum))
	err = dispatch.Run(ctx)
	if err != nil {
		if !errors.Is(ctx.Err(), context.Canceled) {
			panic(err)
		}
	}
}

// ping bot is a packet handler
func PingBot(p *pb.FromRadio) {
	switch v := p.PayloadVariant.(type) {
	case *pb.FromRadio_Packet:
		pkt := v.Packet

		// Only process direct messages
		isBroadcast := pkt.To == 0xFFFFFFFF
		if isBroadcast || pkt.Channel != 0 {
			break
		}

		// only process messages addressed to this node directly
		if pkt.To != myNodeInfo.MyNodeNum {
			break
		}

		// message must be decrypted already by node
		d := pkt.GetDecoded()
		if d == nil {
			break
		}

		// text messages
		if d.Portnum == pb.PortNum_TEXT_MESSAGE_APP {
			text := string(d.Payload)
			log.Info(fmt.Sprintf("[FromRadio] text message [%d] from !%x: %s\n", pkt.Channel, pkt.From, text))

			// ignore replies
			if d.ReplyId != 0 {
				break
			}

			// test if this is a Ping request like /ping !ping or ping, etc
			if i := strings.Index(strings.ToLower(text), "ping"); i < 0 || i > 2 {
				// message is not /ping or "Ping" or "!Ping"
				break
			}

			p := pb.ToRadio{
				PayloadVariant: &pb.ToRadio_Packet{
					Packet: &pb.MeshPacket{
						To: pkt.From,
						PayloadVariant: &pb.MeshPacket_Decoded{
							Decoded: &pb.Data{
								Portnum: pb.PortNum_TEXT_MESSAGE_APP,
								ReplyId: pkt.Id,
								Payload: []byte("Pong"),
							},
						},
					},
				},
			}

			// send over dispatch/stream
			err := meshtastic.Send(context.TODO(), dispatch, &p)
			if err != nil {
				log.Error(fmt.Sprintf("Error sending packet: %v\n", err))
				break
			}

			log.Info(fmt.Sprintf("Sent Ping response to !%x", pkt.From))
		}
	}
}
