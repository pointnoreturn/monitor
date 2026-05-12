package libsnake

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/joho/godotenv"
	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/snake/libradio"
)

func GetSerialNodes(devices ...string) [][]string {
	out := make([][]string, len(devices))
	for i, d := range devices {
		out[i] = []string{d, d}
	}
	return out
}

func Discover(ctx context.Context, timeout time.Duration) []DiscoveredService {
	resolver, _ := zeroconf.NewResolver(nil)
	entries := make(chan *zeroconf.ServiceEntry)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	services := []DiscoveredService{}

	go func() {
		_ = resolver.Browse(ctx, "_meshtastic._tcp", "local.", entries)
	}()

	timer := time.NewTimer(timeout)

	for {
		select {
		case e := <-entries:
			if e == nil {
				continue
			}

			endpoint := ""
			if len(e.AddrIPv4) > 0 {
				endpoint = fmt.Sprintf("%s:%d", e.AddrIPv4[0].String(), e.Port)
			} else if len(e.AddrIPv6) > 0 {
				endpoint = fmt.Sprintf("[%s]:%d", e.AddrIPv6[0].String(), e.Port)
			}

			// key=value pairs in Entry.Text
			args, err := godotenv.Unmarshal(strings.Join(e.Text, "\n"))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				args = make(map[string]string)
			}

			services = append(services, DiscoveredService{
				Endpoint: endpoint,
				Entry:    e,
				Args:     args,
			})

		case <-timer.C:
			return services
		}
	}
}

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

func GetMeshtastic(services []DiscoveredService) []MeshtasticNode {
	nodes := []MeshtasticNode{}
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

		nodes = append(nodes, MeshtasticNode{
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

func MatchNodeList(target string, nodes []MeshtasticNode) *MeshtasticNode {
	target = strings.Trim(target, "! ")
	target = strings.ToLower(target)
	for _, n := range nodes {
		if strings.ToLower(n.Label) == target || strings.Contains(fmt.Sprintf("%x", n.NodeNum), target) { // match by host name or IP or fragment hex num
			return &n
		}
	}

	return nil
}

func NewMeshtasticClient(ctx context.Context, target string) (*MeshtasticClient, error) {
	// TODO: implemented context for socket/operation

	socket := libradio.Socket{}
	err := socket.Init(target)
	if err != nil {
		return nil, err
	}

	c := &MeshtasticClient{
		Socket:   socket,
		Endpoint: target,
	}

	myNodeInfo, nodes, err := c.initializeNodes(libradio.ConfigId_ConfigOnly)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("Failed to GetSelfInfo for %s: %v", target, err)
	}

	if myNodeInfo == nil || len(nodes) < 1 {
		return nil, errors.New("Safety check failed")
	} else if myNodeInfo.MyNodeNum != nodes[0].Num {
		return nil, fmt.Errorf("MyNodeInfo Num %d (!%x) does not match first NodeInfo entry Num %d (safety check failed)", myNodeInfo.MyNodeNum, myNodeInfo.MyNodeNum, nodes[0].Num)
	}

	c.Label = GetNodeLabel(nodes[0])
	c.MyNode = myNodeInfo
	c.NodeDB = nodes
	return c, nil
}

func (c *MeshtasticClient) initializeNodes(configId uint32) (*pb.MyNodeInfo, []*pb.NodeInfo, error) {
	nodes := []*pb.NodeInfo{}
	myNodeInfo, responses, err := c.initializeBase(configId, true)
	if err != nil {
		return myNodeInfo, nodes, err
	}

	for _, p := range responses {
		if n := p.GetNodeInfo(); n != nil {
			nodes = append(nodes, n)
		}
	}

	return myNodeInfo, nodes, err
}

func (c *MeshtasticClient) initializeBase(configId uint32, verifyCompleteId bool) (*pb.MyNodeInfo, []*pb.FromRadio, error) {

	responses, err := c.Socket.SendWantConfigId(configId)
	if err != nil {
		return nil, responses, err
	}

	fmt.Printf("DEBUG: [initializeBase] SendWantConfigId(%d) at %s got %d responses\n", configId, c.Endpoint, len(responses))

	var myNodeInfo *pb.MyNodeInfo
	for _, p := range responses {
		if info := p.GetMyInfo(); info != nil && myNodeInfo == nil {
			myNodeInfo = info
		}
	}

	if myNodeInfo == nil {
		return nil, responses, errors.New("MyNodeInfo packet was missing the response.")
	}

	if verifyCompleteId {
		var completeId uint32 = 0
		for _, p := range responses {
			// Return FIRST node info assuming FIRST == SELF
			if complete := p.GetConfigCompleteId(); complete != 0 && completeId == 0 {
				if complete == configId {
					completeId = complete
				}
			}
		}
		if completeId != configId {
			return myNodeInfo, responses, fmt.Errorf("config_complete_id expected with value %d, but the receive was %d or not sent by the node.", configId, completeId)
		}
	}

	return myNodeInfo, responses, nil
}
