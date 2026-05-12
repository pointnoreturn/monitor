package meshtastic

import (
	"context"
	"fmt"
	"strings"
	"time"

	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/snake/libweather"
)

type Telemeter struct {
	client  *Client
	weather libweather.WeatherProvider
}

func NewTelemeter(client *Client, weather libweather.WeatherProvider) *Telemeter {
	return &Telemeter{
		client:  client,
		weather: weather,
	}
}

func (t *Telemeter) RunLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	fmt.Println("Telemeter loop is running")

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.update()
		default:
			packets, err := t.client.ReadPackets(ctx, true)
			if err != nil {
				panic(err)
			}
			for _, p := range packets {
				t.handlePacket(p)
			}
		}
	}
}

func (t *Telemeter) update() {
	fmt.Println(":update")
}

func (t *Telemeter) handlePacket(p *pb.FromRadio) {
	switch v := p.PayloadVariant.(type) {

	// case *pb.FromRadio_NodeInfo:
	// 	return "NodeInfo"

	// case *pb.FromRadio_MyInfo:
	// 	return "MyInfo"

	// case *pb.FromRadio_Config:
	// 	return "Config"

	// case *pb.FromRadio_LogRecord:
	// 	return "LogRecord"

	case *pb.FromRadio_Packet:
		pkt := v.Packet
		infos := []string{
			fmt.Sprintf("Id %d SNR %.1f RSSI %d", pkt.Id, pkt.RxSnr, pkt.RxRssi),
			fmt.Sprintf("Chan %d From !%x to !%x", pkt.Channel, pkt.From, pkt.To),
		}

		if pkt.HopStart == 0 {
			infos = append(infos, fmt.Sprintf("TTL: %d", pkt.HopLimit))
		} else {
			infos = append(infos, fmt.Sprintf("TTL: %d/%d", pkt.HopStart-pkt.HopLimit, pkt.HopStart))
		}

		if d := pkt.GetDecoded(); d != nil {
			if portName, hasPortName := GetCorePortName(d.Portnum); !hasPortName {
				infos = append(infos, fmt.Sprintf("[D] Port %d Size %d", d.Portnum, len(d.Payload)))
			} else {
				infos = append(infos, fmt.Sprintf("[D] %s Size %d", portName, len(d.Payload)))
			}
			if d.Portnum == pb.PortNum_TEXT_MESSAGE_APP {
				if len(d.Payload) > 0 {
					infos = append(infos, fmt.Sprintf("Text: \"%s\"", string(d.Payload)))
				}
			}
			if emoji := EmojiFromUint32(d.Emoji); emoji != "" { // uint32
				infos = append(infos, fmt.Sprintf("Emoji: %s", emoji))
			}
		} else if e := pkt.GetEncrypted(); e != nil {
			infos = append(infos, fmt.Sprintf("[E] Size %d", len(e)))
		}

		fmt.Println("FromRadio: " + strings.Join(infos, "\n\t"))

	default:
		fmt.Printf("FromRadio: %T\n", p.PayloadVariant)
	}
}
