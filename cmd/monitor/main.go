package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pointnoreturn/monitor/meshtastic"

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

	// Run NodeDB
	nodedb := NewNodeDB()
	go nodedb.Run(ctx)

	targetNode := os.Getenv("TARGET_NODE")
	if len(targetNode) == 0 {
		panic("TARGET_NODE is empty")
	}

	// create and connect client
	stream, myNodeInfo, nodeInfo, err := meshtastic.AutoConnect(ctx, targetNode, time.Second*5, meshtastic.ConfigId_ConfigOnly, nodedb.HandlePacket)
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	label := meshtastic.GetNodeLabel(nodeInfo.User.ShortName, nodeInfo.Num)
	fmt.Printf("Connected to node: %s (!%x), pio %s\n", label, myNodeInfo.MyNodeNum, myNodeInfo.PioEnv)

	// Run reporter
	reporter := NewReporter(myNodeInfo.MyNodeNum, nodedb)
	go reporter.Run(ctx)

	handlers := meshtastic.ChainPacketHandlers(
		printPacket,
		nodedb.HandlePacket,
		reporter.HandlePacket,
	)

	// create dispatch with packet handlers configured
	var dispatch *meshtastic.Dispatch = meshtastic.NewDispatch(stream, 100, handlers)

	// run packet handlers as Dispatch
	err = dispatch.Run(ctx)
	if err != nil {
		if !errors.Is(ctx.Err(), context.Canceled) {
			fmt.Println("Critical error in Dispatch.Run()")
			panic(err)
		}
	}
}
