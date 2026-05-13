package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/snake/meshtastic"

	// This blank import triggers the automatic loading of .env
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)
	defer stop()

	targetNode := os.Getenv("TARGET_NODE")
	if len(targetNode) == 0 {
		panic("TARGET_NODE is empty")
	}

	// Run NodeDB
	nodedb := NewNodeDB()
	go nodedb.Run(ctx)

	// create and connect client
	var client *meshtastic.Client = connect(ctx, targetNode, []meshtastic.PacketF{
		// connection initialization handlers chain
		nodedb.HandlePacket,
	})
	defer client.Close()
	fmt.Printf("Connected to: %s (!%x) at %s\n", client.Label, client.MyNode.MyNodeNum, client.Port)

	if client.MyNode == nil || client.MyNode.MyNodeNum == 0 {
		panic("Client MyNodeInfo initialization has failed for some weird reason.")
	}

	// Run reporter
	reporter := NewReporter(client.MyNode.MyNodeNum, nodedb)
	go reporter.Run(ctx)

	// create dispatch with packet handlers configured
	var dispatch *meshtastic.Dispatch = meshtastic.NewDispatch(&client.ProtoStream, 10, []meshtastic.PacketF{
		// packet sniffing handlers chain
		func(p *pb.FromRadio) {
			logPacket(p, client.MyNode.MyNodeNum)
		},
		nodedb.HandlePacket,
		reporter.HandlePacket,
	})

	// run packet handlers as Dispatch
	err := dispatch.Run(ctx)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			fmt.Println("Non-critical error: " + err.Error())
			return
		}

		fmt.Println("Critical error in Dispatch.Run()")
		panic(err)
	}
}
