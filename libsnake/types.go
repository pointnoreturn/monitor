package libsnake

import (
	"github.com/grandcat/zeroconf"
	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/snake/libradio"
)

type MeshtasticClient struct {
	Socket     libradio.Socket
	Endpoint   string
	NodeId     string
	Label      string
	MyNodeInfo *pb.MyNodeInfo
	NodeDB     []*pb.NodeInfo
}

func (c *MeshtasticClient) Close() {
	c.Socket.Close()
}

func (c *MeshtasticClient) String() string {
	if len(c.Label) == 0 {
		return c.Endpoint
	}
	return c.Label
}

type DiscoveredService struct {
	Endpoint string
	Entry    *zeroconf.ServiceEntry
	Args     map[string]string
}

type MeshtasticNode struct {
	Service   DiscoveredService
	NodeNum   uint32
	ShortName string
	Label     string
}
