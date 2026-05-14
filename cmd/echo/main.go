package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/meshtastic"

	// This blank import triggers the automatic loading of .env
	_ "github.com/joho/godotenv/autoload"
)

// state for connected meshtastic node
var state struct {
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

	handlers := meshtastic.ChainPacketHandlers(
		Echo,
	)

	targetNode := os.Getenv("TARGET_NODE")
	if len(targetNode) == 0 {
		panic("TARGET_NODE is empty")
	}

	stream, myNodeInfo, nodeInfo, err := meshtastic.AutoConnect(ctx, targetNode, time.Second*5, meshtastic.ConfigId_ConfigOnly, nil)
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	state.myNodeInfo = myNodeInfo
	state.nodeInfo = nodeInfo
	state.dispatch = meshtastic.NewDispatch(stream, 100, handlers)

	err = state.dispatch.Run(ctx)
	if err != nil {
		if !errors.Is(ctx.Err(), context.Canceled) {
			panic(err)
		}
	}
}

// echo bot is a packet handling function (PacketF)
func Echo(p *pb.FromRadio) {
	switch v := p.PayloadVariant.(type) {
	case *pb.FromRadio_Packet:
		pkt := v.Packet

		// Only process direct messages to THIS node
		isBroadcast := pkt.To == 0xFFFFFFFF
		if isBroadcast || pkt.Channel != 0 || pkt.To != state.myNodeInfo.MyNodeNum {
			break
		}

		// encryption removed (must have key)
		d := pkt.GetDecoded()
		if d == nil {
			break
		}

		if d.Portnum == pb.PortNum_TEXT_MESSAGE_APP {
			text := string(d.Payload)
			fmt.Printf("[FromRadio] text message [%d] from !%x: %s\n", pkt.Channel, pkt.From, text)

			// ignore replies
			if d.ReplyId != 0 {
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
								Payload: []byte("Echo!"),
							},
						},
					},
				},
			}

			err := meshtastic.Send(context.TODO(), state.dispatch, &p)
			if err != nil {
				fmt.Printf("Error sending packet: %v\n", err)
				break
			}

			fmt.Println("Sent Echo response.")
		}
	}
}
