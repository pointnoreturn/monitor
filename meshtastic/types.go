package meshtastic

import (
	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libradios"
)

// reference of a Bonjour discovered Meshtastic service
type ResolvedNode struct {
	Service   libradios.ResolvedService
	NodeNum   uint32
	ShortName string
	Label     string
}

type PacketF func(*pb.FromRadio)
