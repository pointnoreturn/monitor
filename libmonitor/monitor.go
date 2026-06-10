package libmonitor

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libmetric"
	"github.com/pointnoreturn/monitor/libweather"
	"github.com/pointnoreturn/monitor/meshtastic"
	"google.golang.org/protobuf/proto"
)

type Monitor struct {
	logger     *slog.Logger
	weather    libweather.WeatherProvider
	nodeInfo   *pb.NodeInfo
	myNodeInfo *pb.MyNodeInfo
	writer     meshtastic.Writer
}

func NewMonitor(ctx context.Context, logger *slog.Logger) *Monitor {
	w := makeWeatherProvider(logger)

	return &Monitor{
		weather: w,
		logger:  logger,
	}
}

func (r *Monitor) Assign(nodeInfo *pb.NodeInfo, myNodeInfo *pb.MyNodeInfo, writer meshtastic.Writer) {
	r.myNodeInfo = myNodeInfo
	r.nodeInfo = nodeInfo
	r.writer = writer
}

var (
	boolStr = map[bool]string{true: "1", false: "0"}
	strBool = map[string]bool{"0": false, "false": false, "1": true, "true": true, "False": false, "True": true}

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

func (r *Monitor) Run(ctx context.Context) {
	t0 := groups[0].Ticker()
	t1 := groups[1].Ticker()
	t2 := groups[2].Ticker()
	t3 := groups[3].Ticker()

	commitGroup := func(groupId int) {
		// Todo: batch API request
		if ok := groups[groupId].Commit(); !ok {
			r.logger.Error("[Monitor] commitGroup failed")
		}
	}

	addRuntime := func(seconds float64) {
		ok := runtime.Add(
			seconds,
			"self", fmt.Sprintf("%x", r.myNodeInfo.MyNodeNum),
			"pio_env", r.myNodeInfo.PioEnv,
			"hw", strconv.Itoa(int(r.nodeInfo.User.HwModel)),
		)
		if !ok {
			r.logger.Error("[Monitor] addRuntime failed")
		}
	}

	refreshTelemetry := func(telemetry *pb.Telemetry) bool {
		if r.writer == nil {
			return false
		}

		requestId, err := meshtastic.RequestTelemetry(ctx, r.writer, r.myNodeInfo.MyNodeNum, telemetry)
		if err != nil {
			r.logger.Warn("[refreshTelemetry] RequestTelemetry() failed", "err", err, "type", fmt.Sprintf("%T", telemetry.Variant))
			return false
		}

		r.logger.Debug("[refreshTelemetry] RequestTelemetry sent", "requestId", requestId, "type", fmt.Sprintf("%T", telemetry.Variant))
		return true
	}

	updateWeather := func() {
		if r.weather == nil {
			return
		}

		w, err := r.weather.GetWeather(ctx)
		if err != nil {
			r.logger.Error("[Monitor] updateWeather failed", "err", err)
		} else {
			labels := []string{
				"self", fmt.Sprintf("%x", r.myNodeInfo.MyNodeNum),
				"location", w.Name,
			}
			weatherDifficulty.Update(
				float64(w.RadioDifficulty()),
				labels...,
			)
			weatherTempC.Update(float64(w.TempCelsiusFeelsLike), labels...)
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
			updateWeather()
			addRuntime(groups[3].Interval.Seconds())
			commitGroup(3)
		}
	}
}

func (r *Monitor) HandlePacket(p *pb.FromRadio) {
	if r == nil || r.myNodeInfo == nil || r.nodeInfo == nil {
		return
	}

	labels := []string{"self", fmt.Sprintf("%x", r.myNodeInfo.MyNodeNum)}

	switch v := p.PayloadVariant.(type) {

	case *pb.FromRadio_Packet:
		pkt := v.Packet

		if pkt.From == r.myNodeInfo.MyNodeNum {
			if d := pkt.GetDecoded(); d != nil {
				if d.ReplyId != 0 || d.RequestId != 0 {
					r.handleResponse(pkt, d, labels)
				}
			}
		}

		if pkt.From == r.myNodeInfo.MyNodeNum {
			break
		}

		// ignore for UDP-injected packets (UDP broadcast over network)
		if pkt.GetRxSnr() == 0 && pkt.GetRxRssi() == 0 {
			break
		}

		logRX(pkt, labels)
		logDirect(pkt, labels)
		logContent(pkt, labels)
		logSenders(pkt, labels)
	case *pb.FromRadio_QueueStatus:
		break // ignore
	default:
		r.logger.Warn("Unknown packet type", "type", fmt.Sprintf("%T", p.PayloadVariant))
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

func (r *Monitor) handleResponse(pkt *pb.MeshPacket, d *pb.Data, labels []string) {
	labels = cloneLabels(labels)

	switch d.Portnum {
	case pb.PortNum_TELEMETRY_APP:
		var telemetry pb.Telemetry
		err := proto.Unmarshal(d.Payload, &telemetry)
		if err != nil {
			r.logger.Error("Failed to Unmarshall telemetry packet", "err", err, "requestId", d.RequestId, "replyId", d.ReplyId, "id", pkt.Id)
			break
		}

		r.logger.Debug("Received telemetry", "type", fmt.Sprintf("%T", telemetry.Variant))
		switch t := telemetry.Variant.(type) {
		case *pb.Telemetry_DeviceMetrics:
			r.logger.Debug("Device metrics received")
			groups[1].Update(&chUtil, float64(t.DeviceMetrics.GetChannelUtilization()), labels...)
			groups[1].Update(&airUtilTx, float64(t.DeviceMetrics.GetAirUtilTx()), labels...)
		case *pb.Telemetry_LocalStats:
			r.logger.Debug("Local stats received")

			newBadPackets := t.LocalStats.GetNumPacketsRxBad()
			r.logger.Debug("Received Bad packets", "newBadPackets", newBadPackets, "oldBadPackets", oldBadPackets)

			cRxBad, err := libmetric.MakeSeries("rx_bad",
				"self", fmt.Sprintf("%x", r.myNodeInfo.MyNodeNum),
			)
			if err != nil {
				r.logger.Error("Failed to MakeSeries for rx_bad", "err", err)
				break
			}

			if oldBadPackets == 0 {
				oldBadPackets = newBadPackets
			} else if newBadPackets > oldBadPackets {
				cRxBad.Add(float64(newBadPackets) - float64(oldBadPackets))
				err := cRxBad.Commit()
				if err != nil {
					r.logger.Error("Failed to commit update on rx_bad", "err", err)
					break
				}

				oldBadPackets = newBadPackets
			}

		default:
			r.logger.Debug("Unhandled telemetry type", "replyId", d.ReplyId, "id", pkt.Id, "requestId", d.RequestId)
		}
	default:
		r.logger.Debug("Unhandled response type", "replyId", d.ReplyId, "id", pkt.Id, "requestId", d.RequestId)
	}
}
