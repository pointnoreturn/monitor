package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/snake/libsnake"
	"github.com/pointnoreturn/snake/libweather"
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

	var w libweather.WeatherProvider = InitWeatherProvider()
	w = w // TODO weather use

	targetNode := os.Getenv("TARGET_NODE")
	if len(targetNode) == 0 {
		panic("TARGET_NODE is empty")
	}

	var c *meshtastic.Client = InitClient(ctx, targetNode)
	defer c.Close()
	fmt.Printf("Connected to: %s (!%x) at %s\n", c.Label, c.MyNode.MyNodeNum, c.Port)

	var t *meshtastic.Dispatch = meshtastic.NewDispatch(&c.ProtoStream, []meshtastic.PacketF{
		func(p *pb.FromRadio) {
			logPacket(p, c.MyNode.MyNodeNum)
		},
	})

	err := t.Run(ctx)

	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			fmt.Println("Non-critical error: " + err.Error())
			return
		}

		fmt.Println("Critical error in Dispatch.Run()")
		panic(err)
	}
}

func InitClient(ctx context.Context, targetNode string) *meshtastic.Client {
	ip, isIP := libsnake.ParseTCPAddress(targetNode, meshtastic.DefaultNodeTcpPort) // try parse as IP address

	if isIP { // connect by IPv4/IPv6 address
		c, err := meshtastic.NewClient(ctx, ip)
		if err != nil {
			panic(fmt.Errorf("Failed to connect to TCP '%s': %w", targetNode, err))
		}
		return c
	} else if strings.Index(targetNode, "/") == 0 { // serial device is a path
		c, err := meshtastic.NewClient(ctx, targetNode)
		if err != nil {
			panic(fmt.Errorf("Failed to connect to serial device '%s': %w", targetNode, err))
		}
		return c
	} else { // discover on LAN, using mDNS scan, match by meshtastic node label or hex num
		fmt.Println("Discover advertised meshtastic nodes on the network")
		all := libsnake.Discover(context.Background(), 4*time.Second)

		fmt.Printf("Find target node '%s' among %d services\n", targetNode, len(all))
		nodes := meshtastic.GetNodes(all)
		node := meshtastic.FindMatch(targetNode, nodes)
		if node == nil {
			err := fmt.Errorf("Node not found using mDNS scan and matching: '%s' (retry/longer scan may fix resolution)", targetNode)
			panic(err)
		}

		fmt.Printf("Connect to node %s\n", node.Service.Endpoint)
		c, err := meshtastic.NewClient(ctx, node.Service.Endpoint)
		if err != nil {
			panic(fmt.Errorf("Failed to connect using discovery for '%s': %w", targetNode, err))
		}
		return c
	}
}
