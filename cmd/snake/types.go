package main

import (
	"context"

	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
)

type SnifferBase interface {
	HandlePacket(*pb.FromRadio)
	Run(context.Context)
}
