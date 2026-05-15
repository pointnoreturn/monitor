package main

import (
	"context"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libweather"
)

type Reporter struct {
	Worker
	weather libweather.WeatherProvider
}

func (reporter *Reporter) Init(ctx context.Context) {
	reporter.weather = makeWeatherProvider(log)
}

func (reporter *Reporter) Run(ctx context.Context) {
}

func (reporter *Reporter) HandlePacket(p *pb.FromRadio) {
}
