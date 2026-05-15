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
	"github.com/pointnoreturn/monitor/meshtastic"

	// This blank import triggers the automatic loading of .env
	_ "github.com/joho/godotenv/autoload"
)

// app for connected meshtastic node
var app struct {
	log        *slog.Logger
	dispatch   *meshtastic.Dispatch
	myNodeInfo *pb.MyNodeInfo
	nodeInfo   *pb.NodeInfo
}

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)
	defer stop()

	// define handlers for FromRadio packets
	handlers := meshtastic.ChainPacketHandlers(
		PingBot,
	)

	// Simple syntax to connect to a node either using Network Broadcasts (Bonjour style) scan or raw IP
	targetNode := os.Getenv("TARGET_NODE")
	if len(targetNode) == 0 {
		panic("TARGET_NODE is empty")
	}

	app.log = slog.New(slog.NewTextHandler(os.Stdout, nil))
	app.log.Info("Pingbot")

	// Using ConfigId_ConfigOnly to omit full NodeDB sync
	stream, myNodeInfo, nodeInfo, err := meshtastic.FindAndConnect(ctx, app.log, targetNode, time.Second*5, meshtastic.ConfigId_ConfigOnly, nil)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			panic("Cannot find target node to connect: " + targetNode)
		}

		panic(err)
	}
	defer stream.Close()

	app.myNodeInfo = myNodeInfo
	app.nodeInfo = nodeInfo

	// create dispatch,
	// packet send/receive abstraction with event loop for meshtastic protocol handling,
	// on top of the stream
	app.dispatch = meshtastic.NewDispatch(stream, 100, handlers)

	// Dispatch runs till context dies
	app.log.Info(fmt.Sprintf("Pingbot connected and running on node !%x", myNodeInfo.MyNodeNum))
	err = app.dispatch.Run(ctx)
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
		if pkt.To != app.myNodeInfo.MyNodeNum {
			break
		}

		// encryption removed (must have key)
		d := pkt.GetDecoded()
		if d == nil {
			break
		}

		if d.Portnum == pb.PortNum_TEXT_MESSAGE_APP {
			text := string(d.Payload)
			app.log.Info(fmt.Sprintf("[FromRadio] text message [%d] from !%x: %s\n", pkt.Channel, pkt.From, text))

			// ignore replies
			if d.ReplyId != 0 {
				break
			}

			// test if this is a Ping request
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

			// either use {stream} or {dispatch} nto send unmanaged packets
			err := meshtastic.Send(context.TODO(), app.dispatch, &p)
			if err != nil {
				app.log.Error(fmt.Sprintf("Error sending packet: %v\n", err))
				break
			}

			app.log.Info(fmt.Sprintf("Sent Echo response to !%x", pkt.From))
		}
	}
}
