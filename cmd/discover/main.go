package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pointnoreturn/snake/libsnake"
)

func main() {
	fmt.Println("Discover advertised services.")
	services := libsnake.Discover(context.Background(), 10*time.Second)
	if len(services) == 0 {
		panic("I have discovered no broadcast services.")
	}

	for _, svc := range services {
		fmt.Printf("- Discovery: [%s], I: %s, Args: %+v\n", svc.Endpoint, svc.Entry.Instance, svc.Args)
	}

	fmt.Println("Get meshtastic nodes")
	nodes := libsnake.GetMeshtastic(services)
	if len(nodes) == 0 {
		panic("I have discovered no Meshtastic nodes among those services.")
	}

	for _, n := range nodes {

		fmt.Printf("- Node: [%s] id=!%x\tnum=%d\tshort=%s\t%s\t%s:%d\n", n.Label, n.NodeNum, n.NodeNum, n.ShortName, n.Service.Endpoint, n.Service.Entry.HostName, n.Service.Entry.Port)
	}

	for _, n := range nodes {

		fmt.Printf("test %s...\n", n.Service.Endpoint)
		c, err := libsnake.NewMeshtasticClient(context.Background(), n.Service.Endpoint)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to connect %s (%s): %v\n", n.Service.Endpoint, n.Label, err)
			continue
		}
		fmt.Println("test OK: " + c.Label + ", " + c.NodeId)

		c.Close()
	}
}
