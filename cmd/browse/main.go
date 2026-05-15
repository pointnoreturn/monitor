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
	"github.com/pointnoreturn/monitor/meshtastic"
)

var log *slog.Logger

func init() {
	log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(
			groups []string,
			a slog.Attr,
		) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
}

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)
	defer stop()

	// browse timeout to wait for node announces
	timeoutContext, cancel := context.WithTimeout(ctx, time.Second*7)
	defer cancel()

	// Channels for browsing
	bs := make(chan *libradios.Broadcast)
	bn := make(chan *meshtastic.BroadcastNode)

	log.Info("Browse advertised meshtastic nodes on the network")

	// Pull observed nodes on the network to list
	allNodes := []*meshtastic.BroadcastNode{}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-timeoutContext.Done():
				return
			case n := <-bn:
				if n == nil {
					break
				}
				allNodes = append(allNodes, n)
				fmt.Printf("- Node: [%s]\tid=!%x\tnum=%d\tshort=%s\t%s\t%s:%d\n", n.Label, n.NodeNum, n.NodeNum, n.ShortName, n.Service.Endpoint, n.Service.Entry.HostName, n.Service.Entry.Port)
			}
		}
	}()

	go libradios.BrowseBroadcasts(timeoutContext, log, bs)
	meshtastic.BrowseNodes(timeoutContext, log, bs, bn)

	log.Info(fmt.Sprintf("Total %d meshtastic nodes. Try connect-and-disconnect...", len(allNodes)))
	if len(allNodes) == 0 {
		panic("No nodes found")
	}

	for _, n := range allNodes {

		log.Info(fmt.Sprintf("Test %s", n.Service.Endpoint))
		stream, myNodeInfo, nodeInfo, err := meshtastic.ConnectTCP(context.Background(), log, n.Service.Endpoint, meshtastic.ConfigId_ConfigOnly, nil)
		if stream != nil {
			stream.Close()
		}

		if err != nil {
			log.Error(fmt.Sprintf("Failed %s (%s): %v\n", n.Service.Endpoint, n.Label, err))
			continue
		}

		label := meshtastic.GetNodeLabel(nodeInfo.User.ShortName, nodeInfo.Num)

		log.Info(fmt.Sprintf("Succeded: %s, !%x\n", label, myNodeInfo.MyNodeNum))
	}
}
