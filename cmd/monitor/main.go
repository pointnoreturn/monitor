package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/meshtastic"

	// This blank import triggers the automatic loading of .env
	_ "github.com/joho/godotenv/autoload"
)

var (
	stream     *meshtastic.ProtoStream
	dispatch   *meshtastic.Dispatch
	myNodeInfo *pb.MyNodeInfo
	nodeInfo   *pb.NodeInfo
	db         DB
	reporter   Reporter
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)
	defer stop()

	handlers := meshtastic.ChainPacketHandlers(
		printPacket,
		db.HandlePacket,
		reporter.HandlePacket,
	)

	targetNode := os.Getenv("TARGET_NODE")
	if len(targetNode) == 0 {
		slog.Error("TARGET_NODE is empty")
		os.Exit(3)
	}

	var err error

	stream, myNodeInfo, nodeInfo, err = meshtastic.FindAndConnect(ctx, log, targetNode, time.Second*10, meshtastic.ConfigId_ConfigOnly, db.HandlePacket)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Error("Cannot find target node to connect: " + targetNode)
			os.Exit(2)
		}
		panic(err)
	}
	defer stream.Close()

	label := meshtastic.GetNodeLabel(nodeInfo.User.ShortName, nodeInfo.Num)
	log.Info("Connected node "+label,
		"label", label,
		"self", myNodeInfo.MyNodeNum,
		"pio_env", myNodeInfo.PioEnv,
	)

	dispatch = meshtastic.NewDispatch(stream, 100, handlers)

	go db.Run(ctx)
	go reporter.Run(ctx)

	log.Info("Monitor dispatch running")
	err = dispatch.Run(ctx)
	if err != nil {
		if !errors.Is(ctx.Err(), context.Canceled) {
			log.Error("Critical error in Dispatch.Run()", "err", err)
			os.Exit(1)
		}
	}
}
