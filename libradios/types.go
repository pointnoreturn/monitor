package libradios

import (
	"context"

	"github.com/grandcat/zeroconf"
)

// Descriptor for a Bonjour service resolved on the network
type ResolvedService struct {
	Endpoint string
	Entry    *zeroconf.ServiceEntry
	Args     map[string]string
}

// send/receive IO primitive
type Transport interface {
	Close()
	CanRead() bool
	CanWrite() bool
	Write(context.Context, []byte) error
	Read(context.Context, []byte) error
}

// protocol-agnostic packet IO streaming WritePacket
type Writer[T any] interface {
	WritePacket(ctx context.Context, packet T) error
}

// protocol-agnostic packet IO streaming ReadPackets
type Reader[T any] interface {
	ReadPackets(ctx context.Context, timeout bool) (packets []T, err error)
}
