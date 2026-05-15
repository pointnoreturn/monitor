package main

import (
	"context"
	"fmt"
	"time"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libmetric"
	"github.com/pointnoreturn/monitor/libweather"
)

var (
	runtimeSeconds = libmetric.AutoWrite{
		Name:          "runtime",
		WriteInterval: time.Minute * 7,
	}
)

type Reporter struct {
	Worker
	weather libweather.WeatherProvider
}

func (r *Reporter) Init(ctx context.Context) {
	r.weather = makeWeatherProvider(appLog)
}

func (r *Reporter) Run(ctx context.Context) {
	interval := time.Minute
	t := time.NewTicker(interval)
	for {
		select {
		case <-t.C:
			runtimeSeconds.Add(interval.Seconds(), "self", fmt.Sprintf("%x", myNodeInfo.MyNodeNum))
		case <-ctx.Done():
			return
		}
	}
}

func (r *Reporter) HandlePacket(p *pb.FromRadio) {
}
