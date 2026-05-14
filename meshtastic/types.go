package meshtastic

import (
	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libradios"
)

type Writer libradios.Writer[*pb.ToRadio]
type Reader libradios.Reader[*pb.FromRadio]

// reference of a Bonjour discovered Meshtastic service
type ResolvedNode struct {
	Service   libradios.ResolvedService // bonjour header
	NodeNum   uint32                    // node number
	ShortName string                    // short name, if any
	Label     string                    // replicate phone app label of a network node SHRT_nnnn or nnnn_nnnn
}

type PacketF func(*pb.FromRadio)

// high level protocol client for Meshtastic
type Client struct {
	ProtoStream
	Port       string         // IP:port, /serial/path, etc
	Label      string         // SHRT_12af node label
	MyNodeInfo *pb.MyNodeInfo // populated during connect or manually updated later
}

func (c *Client) String() string {
	return c.Label
}
