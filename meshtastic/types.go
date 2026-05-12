package meshtastic

import (
	"github.com/pointnoreturn/snake/libsnake"
)

// reference of a Bonjour discovered Meshtastic service
type ResolvedNode struct {
	Service   libsnake.ResolvedService
	NodeNum   uint32
	ShortName string
	Label     string
}
