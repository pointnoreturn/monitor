package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pointnoreturn/monitor/libmetric"
	"github.com/pointnoreturn/monitor/libmonitor"
	"github.com/pointnoreturn/monitor/libsupport"
	"github.com/pointnoreturn/monitor/meshtastic"

	// This blank import triggers the automatic loading of .env

	_ "github.com/joho/godotenv/autoload"
)

var (
	victoriaMetricsUrl = os.Getenv("VICTORIA_METRICS")
	targetNode         = os.Getenv("TARGET_NODE")
	appLog, libLog     = libsupport.LoggersFromEnv()
)

func init() {
	if victoriaMetricsUrl == "" {
		slog.Error("VICTORIA_METRICS is empty")
		os.Exit(3)
	}
	libmetric.Init(victoriaMetricsUrl, libLog)

	if len(targetNode) == 0 {
		slog.Error("TARGET_NODE is empty")
		os.Exit(3)
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
		monitor                         = libmonitor.NewMonitor(ctx, appLog)
		handlePacket meshtastic.PacketF = monitor.HandlePacket
	)

	stream, myNodeInfo, nodeInfo, err := meshtastic.FindAndConnect(ctx, libLog, targetNode, time.Second*10, meshtastic.ConfigId_ConfigOnly, handlePacket)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			appLog.Error("Cannot find target node to connect: " + targetNode)
			os.Exit(2)
		}
		appLog.Error("Connection failed", "err", err)
		panic(err)
	}
	defer stream.Close()

	label := meshtastic.GetNodeLabel(nodeInfo.User.ShortName, nodeInfo.Num)
	appLog.Info("Connected node "+label,
		"label", label,
		"self", myNodeInfo.MyNodeNum,
		"pio_env", myNodeInfo.PioEnv,
	)

	dispatch := meshtastic.NewDispatch(stream, handlePacket)

	appLog.Info("Running Monitor")
	monitor.Assign(nodeInfo, myNodeInfo, dispatch)
	go monitor.Run(ctx)

	appLog.Info("Running Dispatch")
	err = dispatch.Run(ctx)
	if err != nil {
		if !errors.Is(ctx.Err(), context.Canceled) {
			appLog.Error("Critical error in Dispatch.Run()", "err", err)
			os.Exit(1)
		}
	}
}
