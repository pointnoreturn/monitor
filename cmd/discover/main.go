package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pointnoreturn/monitor/libradios"
	"github.com/pointnoreturn/monitor/meshtastic"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)
	defer stop()

	fmt.Println("Discover advertised services.")
	services := libradios.Discover(ctx, 10*time.Second)
	if len(services) == 0 {
		panic("I have discovered no broadcast services.")
	}

	for _, svc := range services {
		fmt.Printf("- Discovery: [%s], I: %s, Args: %+v\n", svc.Endpoint, svc.Entry.Instance, svc.Args)
	}

	fmt.Println("Get meshtastic nodes")
	nodes := meshtastic.ListNodes(services)
	if len(nodes) == 0 {
		panic("I have discovered no Meshtastic nodes among those services.")
	}

	for _, n := range nodes {

		fmt.Printf("- Node: [%s]\tid=!%x\tnum=%d\tshort=%s\t%s\t%s:%d\n", n.Label, n.NodeNum, n.NodeNum, n.ShortName, n.Service.Endpoint, n.Service.Entry.HostName, n.Service.Entry.Port)
	}

	fmt.Println("Test every node connect-and-disconnect...")

	for _, n := range nodes {

		fmt.Printf("test %s...\n", n.Service.Endpoint)
		stream, myNodeInfo, nodeInfo, err := meshtastic.ConnectTCP(ctx, n.Service.Endpoint, meshtastic.DefaultPort, meshtastic.ConfigId_ConfigOnly, nil)
		if stream != nil {
			stream.Close()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed %s (%s): %v\n", n.Service.Endpoint, n.Label, err)
			continue
		}

		label := meshtastic.GetNodeLabel(nodeInfo.User.ShortName, nodeInfo.Num)

		fmt.Printf("test OK: %s, !%x\n", label, myNodeInfo.MyNodeNum)
	}
}
