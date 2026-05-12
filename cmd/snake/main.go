package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/pointnoreturn/snake/libsnake"
	"github.com/pointnoreturn/snake/libweather"

	// This blank import triggers the automatic loading of .env
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	var w libweather.WeatherProvider = InitWeatherProvider()

	targetNode := os.Getenv("TARGET_NODE")
	if len(targetNode) == 0 {
		panic("TARGET_NODE is empty")
	}

	var c *libsnake.MeshtasticClient = InitClient(context.TODO(), targetNode)
	defer c.Close()
	fmt.Printf("Connected to: %s (!%x)\n", c.Label, c.MyNode.MyNodeNum)

	var t *libsnake.Telemeter = libsnake.NewTelemeter(c, w)
	t.RunLoop(context.TODO())
}

func InitClient(ctx context.Context, targetNode string) *libsnake.MeshtasticClient {
	ip := net.ParseIP(targetNode) // try parse as IP address

	if ip != nil { // connect by IPv4/IPv6 address
		c, err := libsnake.NewMeshtasticClient(ctx, ip.String())
		if err != nil {
			panic(fmt.Errorf("Failed to connect to TCP '%s': %w", targetNode, err))
		}
		return c
	} else if strings.Index(targetNode, "/") == 0 { // serial device is a path
		c, err := libsnake.NewMeshtasticClient(ctx, targetNode)
		if err != nil {
			panic(fmt.Errorf("Failed to connect to serial device '%s': %w", targetNode, err))
		}
		return c
	} else { // discover on LAN, using mDNS scan, match by meshtastic node label or hex num
		fmt.Println("Discover advertised meshtastic nodes on the network.")
		all := libsnake.Discover(context.Background(), 4*time.Second)
		nodes := libsnake.GetMeshtastic(all)
		node := libsnake.MatchNodeList(targetNode, nodes)
		if node == nil {
			err := fmt.Errorf("Node not found using mDNS scan and matching: '%s' (retry/longer scan may fix resolution)", targetNode)
			panic(err)
		}

		c, err := libsnake.NewMeshtasticClient(ctx, node.Service.Endpoint)
		if err != nil {
			panic(fmt.Errorf("Failed to connect using discovery for '%s': %w", targetNode, err))
		}
		return c
	}
}
