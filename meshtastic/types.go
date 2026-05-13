package meshtastic

import (
	"github.com/pointnoreturn/snake/libradios"
)

// reference of a Bonjour discovered Meshtastic service
type ResolvedNode struct {
	Service   libradios.ResolvedService
	NodeNum   uint32
	ShortName string
	Label     string
}
