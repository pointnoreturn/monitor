package main

import (
	"context"
	"strconv"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libweather"
)

type Reporter struct {
	Worker
	weather libweather.WeatherProvider
}

func (reporter *Reporter) Init(ctx context.Context) {
	reporter.weather = makeWeatherProvider(appLog)
}

func (reporter *Reporter) Run(ctx context.Context) {
	WriteMetric(
		"uptime", 0,
		"self", strconv.Itoa(int(myNodeInfo.MyNodeNum)),
	)
}

func (reporter *Reporter) HandlePacket(p *pb.FromRadio) {
}
