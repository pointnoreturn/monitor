package meshtastic

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
)

const DefaultPort int = 4403

// Fix escaped emoji in Bonjour service descriptor
func fixMeshtasticShortname(input string) string {
	// Match backslash followed by 3 digits
	re := regexp.MustCompile(`\\(\d{3})`)

	// Replace matches with the actual byte value
	result := re.ReplaceAllFunc([]byte(input), func(match []byte) []byte {
		// match[1:] skips the backslash
		val, err := strconv.Atoi(string(match[1:]))
		if err != nil || val > 255 {
			return match
		}
		return []byte{byte(val)}
	})

	return string(result)
}

func EmojiFromUint32(e uint32) string {
	if e == 0 {
		return ""
	}

	r := rune(e)

	if !unicode.IsGraphic(r) {
		return strconv.Itoa(int(e))
	}

	return string(r)
}

// with NodeInfo returns "canonical" common label like SHRT_nnnn
// same as phone apps show this node without connection
func GetNodeLabel(shortName string, nodeNum uint32) string {
	strId := fmt.Sprintf("!%08x", nodeNum)

	if len(shortName) > 0 {
		if len(strId) >= 6 && strId[0] == '!' {
			strId = strId[len(strId)-4:]
			return fmt.Sprintf("%s_%s", shortName, strId)
		}
		return fmt.Sprintf("%s_%s", strId, strId)
	}

	return strId
}

// find specific meshtastic node in the list
func MatchNode(target string, n *BroadcastNode) bool {
	target = strings.Trim(target, "! ")
	target = strings.ToLower(target)

	if strings.ToLower(n.Label) == target || strings.Contains(fmt.Sprintf("%x", n.NodeNum), target) { // match by host name or IP or fragment hex num
		return true
	}

	return false
}

// number of hops spent on the received mesh packet.
// Hop limit spending is logical, not actual physical rebroadcast count
func HopsAway(pkt *pb.MeshPacket) uint32 {
	if pkt.GetHopStart() == 0 {
		return 0
	} else if pkt.GetHopLimit() > pkt.GetHopStart() {
		return pkt.GetHopLimit()
	}
	return pkt.GetHopStart() - pkt.GetHopLimit()
}

// helper to chain packet handlers in a row
func ChainPacketHandlers(handlers ...PacketF) PacketF {
	return func(inp *pb.FromRadio) {
		for _, h := range handlers {
			h(inp)
		}
	}
}
