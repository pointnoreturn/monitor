package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pointnoreturn/monitor/libradios"
	"github.com/pointnoreturn/monitor/libsupport"
	"github.com/pointnoreturn/monitor/meshtastic"
)

var mainLog, _ *slog.Logger = libsupport.LoggersFromEnv()

const (
	browseTimeout = time.Second * 7
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)
	defer stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, browseTimeout)
	defer cancel()

	foundServices := make(chan *libradios.BroadcastService)
	foundNodes := make(chan *meshtastic.BroadcastNode)

	mainLog.Info("Browse advertised meshtastic nodes on the network", "browseTimeout", browseTimeout)

	allNodes := []*meshtastic.BroadcastNode{}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-timeoutCtx.Done():
				return
			case n := <-foundNodes:
				if n == nil {
					break
				}
				allNodes = append(allNodes, n)
				fmt.Printf("- Node: [%s]\tid=!%x\tnum=%d\tshort=%s\t%s\t%s:%d\n", n.Label, n.NodeNum, n.NodeNum, n.ShortName, n.Service.Endpoint, n.Service.Entry.HostName, n.Service.Entry.Port)
			}
		}
	}()

	go libradios.BrowseBroadcasts(timeoutCtx, mainLog, foundServices)

	meshtastic.BrowseNodes(timeoutCtx, mainLog, foundServices, foundNodes)

	mainLog.Info(fmt.Sprintf("Total %d meshtastic nodes. Try connect-and-disconnect...", len(allNodes)))
	if len(allNodes) == 0 {
		mainLog.Error("No nodes found")
		panic("No nodes found")
	}

	// Extra: test each discovered node using connect
	for _, n := range allNodes {

		mainLog.Info(fmt.Sprintf("Test node with connect: %s", n.Service.Endpoint))
		stream, myNodeInfo, nodeInfo, err := meshtastic.ConnectTCP(context.Background(), mainLog, n.Service.Endpoint, meshtastic.ConfigId_ConfigOnly, nil)
		if stream != nil {
			stream.Close()
		}

		if err != nil {
			mainLog.Error(fmt.Sprintf("FAIL %s (%s): %v\n", n.Service.Endpoint, n.Label, err))
			continue
		}

		label := meshtastic.GetNodeLabel(nodeInfo.User.ShortName, nodeInfo.Num)

		mainLog.Info(fmt.Sprintf("OK: %s, !%x\n", label, myNodeInfo.MyNodeNum))
	}
}
