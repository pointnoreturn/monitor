package meshtastic

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/snake/libradios"
)

const DefaultNodeTcpPort string = "4403"

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

// from list of discovered Bonjour services, extract anouncements for Meshtastic nodes
func AsNodes(services []libradios.ResolvedService) []ResolvedNode {
	nodes := []ResolvedNode{}
	for _, svc := range services {
		if svc.Entry == nil {
			continue
		} else if svc.Entry.Service != "_meshtastic._tcp" {
			fmt.Printf("DEBUG: Unknown service '%s' at %s (%s), ignore\n", svc.Entry.Service, svc.Endpoint, svc.Entry.HostName)
			continue
		}

		if svc.Entry.Domain != "local." {
			fmt.Fprintf(os.Stderr, "INFO: Domaion is '%s', not local at %s (%s)\n", svc.Entry.Domain, svc.Endpoint, svc.Entry.HostName)
		}

		hexId, hasId := svc.Args["id"]
		shortName, hasShortName := svc.Args["shortname"]
		if !hasId || len(hexId) != 9 {
			fmt.Fprintf(os.Stderr, "ERR: Service has no 'id' key at %s (%s), drop\n", svc.Endpoint, svc.Entry.HostName)
			continue
		} else if !hasShortName {
			fmt.Fprintf(os.Stderr, "ERR: Service has no 'shortname' key at %s (%s), drop\n", svc.Endpoint, svc.Entry.HostName)
			continue
		}

		hexId = strings.TrimPrefix(hexId, "!")

		nodeNum, err := strconv.ParseUint(hexId, 16, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERR: Cannot parse 'id' key value '%s' as HEX int32 at %s (%s), drop\n", hexId, svc.Endpoint, svc.Entry.HostName)
			continue
		}

		// short name emoji fix
		if hasShortName {
			shortName = fixMeshtasticShortname(shortName)
		}

		label := shortName
		hexSuffix := hexId[len(hexId)-4:]
		if len(label) == 0 {
			label = hexSuffix + "_" + hexSuffix
		} else {
			label += "_" + hexSuffix
		}

		nodes = append(nodes, ResolvedNode{
			Service:   svc,
			ShortName: shortName,
			NodeNum:   uint32(nodeNum),
			Label:     label,
		})
	}
	return nodes
}

func GetNodeLabel(info *pb.NodeInfo) string {

	short := info.User.ShortName
	nodeID := fmt.Sprintf("!%08x", info.Num)

	if len(nodeID) >= 6 && nodeID[0] == '!' {
		suffix := nodeID[len(nodeID)-4:]
		return fmt.Sprintf("%s_%s", short, suffix)
	} else if len(short) > 0 {
		return short
	}

	return fmt.Sprintf("!%x", info.Num)
}

func FindNode(target string, nodes []ResolvedNode) *ResolvedNode {
	target = strings.Trim(target, "! ")
	target = strings.ToLower(target)
	for _, n := range nodes {
		if strings.ToLower(n.Label) == target || strings.Contains(fmt.Sprintf("%x", n.NodeNum), target) { // match by host name or IP or fragment hex num
			return &n
		}
	}

	return nil
}
