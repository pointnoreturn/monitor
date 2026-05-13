package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/snake/libradios"
	"github.com/pointnoreturn/snake/meshtastic"
)

func connect(ctx context.Context, targetNode string, configHandlers []meshtastic.PacketF) *meshtastic.Client {
	ip, isIP := libradios.ParseTCPAddress(targetNode, meshtastic.DefaultNodeTcpPort) // try parse as IP address

	configHandler := func(p *pb.FromRadio) {
		if configHandlers != nil {
			for _, h := range configHandlers {
				h(p)
			}
		}
	}

	if isIP { // connect by IPv4/IPv6 address
		c, err := meshtastic.NewClient(ctx, ip, configHandler)
		if err != nil {
			panic(fmt.Errorf("Failed to connect to TCP '%s': %w", targetNode, err))
		}
		return c
	} else if strings.Index(targetNode, "/") == 0 { // serial device is a path
		c, err := meshtastic.NewClient(ctx, targetNode, configHandler)
		if err != nil {
			panic(fmt.Errorf("Failed to connect to serial device '%s': %w", targetNode, err))
		}
		return c
	} else { // discover on LAN, using mDNS scan, match by meshtastic node label or hex num
		fmt.Println("Discover advertised meshtastic nodes on the network")
		all := libradios.Discover(context.Background(), 4*time.Second)

		fmt.Printf("Find target node '%s' among %d services\n", targetNode, len(all))
		nodes := meshtastic.AsNodes(all)
		node := meshtastic.FindNode(targetNode, nodes)
		if node == nil {
			err := fmt.Errorf("Node not found using mDNS scan and matching: '%s' (retry/longer scan may fix resolution)", targetNode)
			panic(err)
		}

		fmt.Printf("Connect to node %s\n", node.Service.Endpoint)
		c, err := meshtastic.NewClient(ctx, node.Service.Endpoint, configHandler)
		if err != nil {
			panic(fmt.Errorf("Failed to connect using discovery for '%s': %w", targetNode, err))
		}
		return c
	}
}
