package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pointnoreturn/snake/libradios"
	"github.com/pointnoreturn/snake/meshtastic"
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
	nodes := meshtastic.AsNodes(services)
	if len(nodes) == 0 {
		panic("I have discovered no Meshtastic nodes among those services.")
	}

	for _, n := range nodes {

		fmt.Printf("- Node: [%s] id=!%x\tnum=%d\tshort=%s\t%s\t%s:%d\n", n.Label, n.NodeNum, n.NodeNum, n.ShortName, n.Service.Endpoint, n.Service.Entry.HostName, n.Service.Entry.Port)
	}

	fmt.Println("Test every node connect-and-disconnect...")

	for _, n := range nodes {

		fmt.Printf("test %s...\n", n.Service.Endpoint)
		c, err := meshtastic.NewClient(ctx, n.Service.Endpoint, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed %s (%s): %v\n", n.Service.Endpoint, n.Label, err)
			continue
		}
		fmt.Printf("test OK: %s, !%x\n", c.Label, c.MyNode.MyNodeNum)

		c.Close()
	}
}
