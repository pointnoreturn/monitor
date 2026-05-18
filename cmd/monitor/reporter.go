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
	"google.golang.org/protobuf/proto"
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
	chUtil       = libmetric.AutoCommit{"ch_util"}
	airUtilTx    = libmetric.AutoCommit{"air_util_tx"}

	oldBadPackets uint32 = 0 // todo clean

	rxRssi            = libmetric.AutoCommit{"rssi"}
	rxSnr             = libmetric.AutoCommit{"snr"}
	weatherDifficulty = libmetric.AutoCommit{"weather_difficulty"}
	weatherTempC      = libmetric.AutoCommit{"temp_c"}

	groups = []libmetric.Group{
		{Interval: time.Second * 10},
		{Interval: time.Second * 30},
		{Interval: time.Minute * 3},
		{Interval: time.Minute * 5},
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
			appLog.Error("[Reporter] commitGroup failed")
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
			appLog.Error("[Reporter] addRuntime failed")
		}
	}

	refreshTelemetry := func(telemetry *pb.Telemetry) bool {
		requestId, err := meshtastic.RequestTelemetry(ctx, dispatch, myNodeInfo.MyNodeNum, telemetry)
		if err != nil {
			appLog.Warn("[refreshTelemetry] RequestTelemetry() failed", "err", err, "type", fmt.Sprintf("%T", telemetry.Variant))
			return false
		}

		appLog.Debug("[refreshTelemetry] RequestTelemetry sent", "requestId", requestId, "type", fmt.Sprintf("%T", telemetry.Variant))
		return true
	}

	updateWeather := func(groupId int) {
		if weather != nil {
			w, err := weather.GetWeather(ctx)
			if err != nil {
				appLog.Error("[Reporter] updateWeather failed", "err", err)
			} else {
				labels := []string{
					"self", fmt.Sprintf("%x", myNodeInfo.MyNodeNum),
					"location", w.Name,
				}
				weatherDifficulty.Update(
					float64(w.RadioDifficulty()),
					labels...,
				)
				weatherTempC.Update(float64(w.TempCelsiusFeelsLike), labels...)
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
			refreshTelemetry(&pb.Telemetry{Variant: &pb.Telemetry_LocalStats{}})
			commitGroup(1)

		case <-t2.C:
			refreshTelemetry(&pb.Telemetry{Variant: &pb.Telemetry_DeviceMetrics{}})
			commitGroup(2)

		case <-t3.C:
			updateWeather(3)
			addRuntime(groups[3].Interval.Seconds())
			commitGroup(3)
		}
	}
}

func (r *Reporter) HandlePacket(p *pb.FromRadio) {
	labels := []string{"self", fmt.Sprintf("%x", myNodeInfo.MyNodeNum)}

	switch v := p.PayloadVariant.(type) {

	case *pb.FromRadio_Packet:
		pkt := v.Packet

		if pkt.From == myNodeInfo.MyNodeNum {
			if d := pkt.GetDecoded(); d != nil {
				if d.ReplyId != 0 || d.RequestId != 0 {
					onResponse(pkt, d, labels)
				}
			}
		}

		if pkt.From == myNodeInfo.MyNodeNum {
			break
		}

		logRX(pkt, labels)
		logDirect(pkt, labels)
		logContent(pkt, labels)
		logSenders(pkt, labels)
	case *pb.FromRadio_QueueStatus:
		break // ignore
	default:
		appLog.Warn("Unknown packet type", "type", fmt.Sprintf("%T", p.PayloadVariant))
	}
}

func cloneLabels(l []string) []string {
	out := make([]string, len(l))
	copy(out, l)
	return out
}

func logRX(pkt *pb.MeshPacket, labels []string) {
	labels = cloneLabels(labels)

	rxRssi.Update(float64(pkt.RxRssi), labels...)
	rxSnr.Update(float64(pkt.RxSnr), labels...)

	if d := pkt.GetDecoded(); d != nil {
		labels = append(labels, "port", d.Portnum.String())
	} else {
		labels = append(labels, "port", "UNKNOWN_APP")
	}

	groups[0].AddOne(&totalRX, labels...)
}

func logDirect(pkt *pb.MeshPacket, labels []string) {
	labels = cloneLabels(labels)

	hopsAway := int(meshtastic.HopsAway(pkt))
	if hopsAway > 0 {
		return
	}

	isStrong := pkt.RxRssi > -105 && pkt.RxSnr > -5
	labels = append(labels, "strong", boolStr[isStrong])
	groups[0].AddOne(&rxDirect, labels...)
}

func logSenders(pkt *pb.MeshPacket, labels []string) {
	labels = cloneLabels(labels)

	hopsAway := int(meshtastic.HopsAway(pkt))
	labels = append(labels, "from", fmt.Sprintf("%x", pkt.From))

	if hopsAway <= 3 {
		groups[0].AddOne(&rxProximity, labels...)
	}

	labels = append(labels, "hops", strconv.Itoa(hopsAway))
	groups[1].AddOne(&senders, labels...)
}

func logContent(pkt *pb.MeshPacket, labels []string) {
	labels = cloneLabels(labels)

	d := pkt.GetDecoded()
	if d == nil {
		groups[0].AddOne(&totalUnknown, labels...)
		return
	}

	groups[0].AddOne(&totalDecoded, labels...)
}

func onResponse(pkt *pb.MeshPacket, d *pb.Data, labels []string) {
	labels = cloneLabels(labels)

	switch d.Portnum {
	case pb.PortNum_TELEMETRY_APP:
		var telemetry pb.Telemetry
		err := proto.Unmarshal(d.Payload, &telemetry)
		if err != nil {
			appLog.Error("Failed to Unmarshall telemetry packet", "err", err, "requestId", d.RequestId, "replyId", d.ReplyId, "id", pkt.Id)
			break
		}

		appLog.Debug("Received telemetry", "type", fmt.Sprintf("%T", telemetry.Variant))
		switch t := telemetry.Variant.(type) {
		case *pb.Telemetry_DeviceMetrics:
			appLog.Debug("Device metrics received")
			groups[1].Update(&chUtil, float64(t.DeviceMetrics.GetChannelUtilization()), labels...)
			groups[1].Update(&airUtilTx, float64(t.DeviceMetrics.GetAirUtilTx()), labels...)
		case *pb.Telemetry_LocalStats:
			appLog.Debug("Local stats received")

			newBadPackets := t.LocalStats.GetNumPacketsRxBad()
			appLog.Debug("Received Bad packets", "newBadPackets", newBadPackets, "oldBadPackets", oldBadPackets)

			cRxBad, err := libmetric.MakeSeries("rx_bad",
				"self", fmt.Sprintf("%x", myNodeInfo.MyNodeNum),
			)
			if err != nil {
				appLog.Error("Failed to MakeSeries for rx_bad", "err", err)
				break
			}

			if oldBadPackets == 0 {
				oldBadPackets = newBadPackets
			} else if newBadPackets > oldBadPackets {
				cRxBad.Add(float64(newBadPackets) - float64(oldBadPackets))
				err := cRxBad.Commit()
				if err != nil {
					appLog.Error("Failed to commit update on rx_bad", "err", err)
					break
				}

				oldBadPackets = newBadPackets
			}

		default:
			appLog.Debug("Unhandled telemetry type", "replyId", d.ReplyId, "id", pkt.Id, "requestId", d.RequestId)
		}
	default:
		appLog.Debug("Unhandled response type", "replyId", d.ReplyId, "id", pkt.Id, "requestId", d.RequestId)
	}
}
