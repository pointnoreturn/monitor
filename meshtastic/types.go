package meshtastic

import (
	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libradios"
)

// reference of a Bonjour discovered Meshtastic service
type ResolvedNode struct {
	Service   libradios.ResolvedService // bonjour header
	NodeNum   uint32                    // node number
	ShortName string                    // short name, if any
	Label     string                    // replicate phone app label of a network node SHRT_nnnn or nnnn_nnnn
}

type PacketF func(*pb.FromRadio)
