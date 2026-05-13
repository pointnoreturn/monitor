package main

import (
	"context"

	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/snake/libweather"
)

type Reporter struct {
	SnifferBase
	selfNodeNum uint32
	nodedb      *NodeDB
	weather     libweather.WeatherProvider
}

func NewReporter(selfNodeNum uint32, nodedb *NodeDB) *Reporter {
	return &Reporter{
		selfNodeNum: selfNodeNum,
		nodedb:      nodedb,
		weather:     makeWeatherProvider(),
	}
}

func (reporter *Reporter) HandlePacket(p *pb.FromRadio) {
}

func (reporter *Reporter) Run(ctx context.Context) {

}
