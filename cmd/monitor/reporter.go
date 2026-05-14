package main

import (
	"context"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libweather"
)

type Reporter struct {
	Worker
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
