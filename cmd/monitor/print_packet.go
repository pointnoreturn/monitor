package main

import (
	"fmt"
	"strings"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/meshtastic"
)

func printPacket(p *pb.FromRadio) {
	switch v := p.PayloadVariant.(type) {

	// case *pb.FromRadio_QueueStatus: // Ignore connection status metadata (heartbeat related)
	// 	fmt.Println("...")
	// 	return

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

		relayInfo := fmt.Sprintf(" relay %x next %x", pkt.RelayNode, pkt.NextHop)
		rxInfo := fmt.Sprintf("✴️ %.1f 📶%d ", pkt.RxSnr, pkt.RxRssi)
		if pkt.RelayNode == 0 && pkt.NextHop == 0 {
			relayInfo = ""
		}

		hopsAway := meshtastic.HopsAway(pkt)

		infos := []string{
			fmt.Sprintf("%s#%d chan %d from !%x to !%x%s", rxInfo, pkt.Id, pkt.Channel, pkt.From, pkt.To, relayInfo),
		}

		if pkt.HopStart == 0 {
			infos = append(infos, fmt.Sprintf("🐇 %d", pkt.HopLimit))
		} else {
			infos = append(infos, fmt.Sprintf("🐇 %d/%d (%d hops away)", pkt.HopLimit, pkt.HopStart, hopsAway))
		}

		varType := fmt.Sprintf("bytes %T", pkt.PayloadVariant)
		switch pkt.PayloadVariant.(type) {
		case *pb.MeshPacket_Decoded:
			varType = "payload"
		case *pb.MeshPacket_Encrypted:
			varType = "encrypted"
		}

		if d := pkt.GetDecoded(); d != nil {
			if portName, hasPortName := pb.PortNum_name[int32(d.Portnum)]; !hasPortName {
				infos = append(infos, fmt.Sprintf("📗 port %d sz %d %s", d.Portnum, len(d.Payload), varType))
			} else {
				infos = append(infos, fmt.Sprintf("📗 %s sz %d %s", portName, len(d.Payload), varType))
			}
			if d.Portnum == pb.PortNum_TEXT_MESSAGE_APP {
				if len(d.Payload) > 0 {
					text := string(d.Payload)
					text = strings.ReplaceAll(text, "\n", "\\n")
					infos = append(infos, fmt.Sprintf("Text: \"%s\"", text))
				}
			}
			if emoji := meshtastic.EmojiFromUint32(d.Emoji); emoji != "" { // uint32
				infos = append(infos, fmt.Sprintf("emoji: %s", emoji))
			}
		} else if e := pkt.GetEncrypted(); e != nil {
			infos = append(infos, fmt.Sprintf("📕 sz %d %s", len(e), varType))
		}

		fmt.Println("[FromRadio] " + strings.Join(infos, "\n\t"))

	default:
		fmt.Printf("[FromRadio] %T\n", p.PayloadVariant)
	}
}
