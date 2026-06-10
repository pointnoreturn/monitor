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
	mainLog, _ *slog.Logger = libsupport.LoggersFromEnv()
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

	stream, myNodeInfo, nodeInfo, err = meshtastic.FindAndConnect(ctx, mainLog, targetNode, time.Second*5, meshtastic.ConfigId_ConfigOnly, handlePacket)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			panic("Cannot find target node to connect: " + targetNode)
		}
		mainLog.Error("Connection failed", "err", err)
		panic(err)
	}
	defer stream.Close()

	mainLog.Info(fmt.Sprintf("Connected to node !%x", myNodeInfo.GetMyNodeNum()))

	dispatch = meshtastic.NewDispatch(stream, handlePacket)
	mainLog.Info("Running dispatch")
	err = dispatch.Run(ctx)
	if err != nil {
		if !errors.Is(ctx.Err(), context.Canceled) {
			mainLog.Error("Fatal error", "err", err)
			panic(err)
		}
	}
}

// ping bot is a packet handler
func PingBot(inp *pb.FromRadio) {
	switch v := inp.PayloadVariant.(type) {
	case *pb.FromRadio_Packet:
		pkt := v.Packet

		// Only process direct messages
		isBroadcast := pkt.To == 0xFFFFFFFF
		if isBroadcast || pkt.Channel != 0 {
			break
		}

		// only process messages addressed to this node directly
		if pkt.To != myNodeInfo.GetMyNodeNum() {
			break
		}

		// message must be decrypted already by node
		d := pkt.GetDecoded()
		if d == nil {
			break
		}

		// text messages only
		if d.GetPortnum() != pb.PortNum_TEXT_MESSAGE_APP {
			break
		}

		text := string(d.GetPayload())
		mainLog.Info(fmt.Sprintf("[FromRadio] text message [%d] from !%x: %s\n", pkt.GetChannel(), pkt.GetFrom(), text))

		// ignore replies
		if d.ReplyId != 0 {
			break
		}

		// test if this is a Ping request like /ping !ping or ping, etc
		if i := strings.Index(strings.ToLower(text), "ping"); i < 0 || i > 2 {
			// message is not /ping or "Ping" or "!Ping"
			break
		}

		outp := pb.ToRadio{
			PayloadVariant: &pb.ToRadio_Packet{
				Packet: &pb.MeshPacket{
					To:      pkt.From,
					Channel: pkt.GetChannel(),
					PayloadVariant: &pb.MeshPacket_Decoded{
						Decoded: &pb.Data{
							Portnum: pb.PortNum_TEXT_MESSAGE_APP,
							ReplyId: pkt.GetId(),
							Payload: []byte("Pong"),
						},
					},
				},
			},
		}

		mainLog.Info(fmt.Sprintf("Send Ping response to !%x", pkt.GetFrom()))

		err := meshtastic.Send(context.TODO(), dispatch, &outp)

		if err != nil {
			mainLog.Error(fmt.Sprintf("Error sending packet for ping response: %v\n", err))
			break
		}
	}
}
