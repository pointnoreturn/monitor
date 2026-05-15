package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libmetric"
	"github.com/pointnoreturn/monitor/libweather"
	"github.com/pointnoreturn/monitor/meshtastic"
)

var (
	weather libweather.WeatherProvider
	boolStr = map[bool]string{true: "1", false: "0"}
)

type Reporter struct{}

func (r *Reporter) Init(ctx context.Context) {
	if weather != nil {
		panic("Cannot intiialize Reporter twice")
	}

	weather = makeWeatherProvider(appLog)
}

var (
	runtime = libmetric.AutoCommit{Name: "runtime"}

	totalRX      = libmetric.AutoCommit{"total_rx"}
	rxProximity  = libmetric.AutoCommit{"rx_proximity"}
	rxDirect     = libmetric.AutoCommit{"rx_direct"}
	totalUnknown = libmetric.AutoCommit{"total_unknown"}
	totalDecoded = libmetric.AutoCommit{"total_decoded"}
	senders      = libmetric.AutoCommit{"senders"}

	rxRssi = libmetric.Sampler{
		Name:     "rssi_p50",
		Function: libmetric.MedianP50,
		MinCount: 100,
		MaxCount: 1000,
	}
	rxSnr = libmetric.Sampler{
		Name:     "snr_p50",
		Function: libmetric.MedianP50,
		MinCount: 100,
		MaxCount: 1000,
	}

	weatherDifficulty = libmetric.AutoCommit{"weather_difficulty"}
	weatherTempA      = libmetric.AutoCommit{"temp_a"}

	groups = []libmetric.Group{
		{Interval: time.Second * 10},
		{Interval: time.Minute * 3},
		{Interval: time.Minute * 7},
		{Interval: time.Minute * 15},
	}
)

func (r *Reporter) Run(ctx context.Context) {
	t0 := groups[0].Ticker()
	t1 := groups[1].Ticker()
	t2 := groups[2].Ticker()
	t3 := groups[3].Ticker()

	commitGroup := func(groupId int) {
		// Todo: batch API request
		if ok := groups[groupId].Commit(); !ok {
			// todo log
		}
	}

	addRuntime := func(seconds float64) {
		ok := runtime.Add(
			seconds,
			"self", fmt.Sprintf("%x", myNodeInfo.MyNodeNum),
			"pio_env", myNodeInfo.PioEnv,
			"hw", strconv.Itoa(int(nodeInfo.User.HwModel)),
		)
		if !ok {
			// todo log
		}
	}

	updateWeather := func(groupId int) {
		if weather != nil {
			w, err := weather.GetWeather(ctx)
			if err != nil {
				// todo log
			} else {
				labels := []string{
					"self", fmt.Sprintf("%x", myNodeInfo.MyNodeNum),
					"location", w.Name,
				}
				groups[groupId].Set(
					&weatherDifficulty, float64(w.RadioDifficulty()),
					labels...,
				)
				groups[groupId].Set(&weatherTempA, float64(w.TempCelsiusFeelsLike), labels...)
			}
		}
	}

	addRuntime(1)

	for {
		select {
		case <-ctx.Done():
			return

		case <-t0.C:
			commitGroup(0)

		case <-t1.C:
			commitGroup(1)

		case <-t2.C:
			updateWeather(2)
			addRuntime(groups[2].Interval.Seconds())
			commitGroup(2)

		case <-t3.C:
			commitGroup(3)
		}
	}
}

func (r *Reporter) HandlePacket(p *pb.FromRadio) {
	switch v := p.PayloadVariant.(type) {

	case *pb.FromRadio_Packet:
		pkt := v.Packet

		if pkt.From == myNodeInfo.MyNodeNum {
			break
		}

		labels := []string{"self", fmt.Sprintf("%x", myNodeInfo.MyNodeNum)}

		logRX(pkt, labels)
		logDirect(pkt, labels)
		logContent(pkt, labels)
		logSenders(pkt, labels)
	}
}

func logRX(pkt *pb.MeshPacket, labels []string) {
	groups[1].Sample(&rxRssi, float64(pkt.RxRssi), labels...)
	groups[1].Sample(&rxSnr, float64(pkt.RxSnr), labels...)

	if d := pkt.GetDecoded(); d != nil {
		labels = append(labels, "port", d.Portnum.String())
	} else {
		labels = append(labels, "port", "UNKNOWN_APP")
	}

	groups[0].AddOne(&totalRX, labels...)
}

func logDirect(pkt *pb.MeshPacket, labels []string) {
	hopsAway := int(meshtastic.HopsAway(pkt))
	if hopsAway > 0 {
		return
	}

	isStrong := pkt.RxRssi > -105 && pkt.RxSnr > -5
	labels = append(labels, "strong", boolStr[isStrong])
	groups[0].AddOne(&rxDirect, labels...)
}

func logSenders(pkt *pb.MeshPacket, labels []string) {
	hopsAway := int(meshtastic.HopsAway(pkt))
	labels = append(labels, "from", fmt.Sprintf("%x", pkt.From))

	if hopsAway <= 3 {
		groups[0].AddOne(&rxProximity, labels...)
	}

	labels = append(labels, "hops", strconv.Itoa(hopsAway))
	groups[1].AddOne(&senders, labels...)
}

func logContent(pkt *pb.MeshPacket, labels []string) {
	d := pkt.GetDecoded()
	if d == nil {
		groups[0].AddOne(&totalUnknown, labels...)
		return
	}

	groups[0].AddOne(&totalDecoded, labels...)
}
