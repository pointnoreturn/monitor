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
	log        *slog.Logger
	weather    libweather.WeatherProvider
	nodeInfo   *pb.NodeInfo
	myNodeInfo *pb.MyNodeInfo
	writer     meshtastic.Writer
}

func NewMonitor(ctx context.Context, log *slog.Logger) *Monitor {
	w := makeWeatherProvider(log)

	return &Monitor{
		weather: w,
		log:     log,
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
			r.log.Error("[Monitor] commitGroup failed")
		}
	}

	addRuntime := func(seconds float64) {
		ok := runtime.Add(
			seconds,
			"self", fmt.Sprintf("%x", r.myNodeInfo.GetMyNodeNum()),
			"pio_env", r.myNodeInfo.GetPioEnv(),
			"hw", strconv.Itoa(int(r.nodeInfo.GetUser().GetHwModel())),
		)
		if !ok {
			r.log.Error("[Monitor] addRuntime failed")
		}
	}

	refreshTelemetry := func(telemetry *pb.Telemetry) bool {
		if r.writer == nil {
			return false
		}

		requestId, err := meshtastic.RequestTelemetry(ctx, r.writer, r.myNodeInfo.GetMyNodeNum(), telemetry)
		if err != nil {
			r.log.Warn("[refreshTelemetry] RequestTelemetry() failed", "err", err, "type", fmt.Sprintf("%T", telemetry.Variant))
			return false
		}

		r.log.Debug("[refreshTelemetry] RequestTelemetry sent", "requestId", requestId, "type", fmt.Sprintf("%T", telemetry.Variant))
		return true
	}

	updateWeather := func() {
		if r.weather == nil {
			return
		}

		w, err := r.weather.GetWeather(ctx)
		if err != nil {
			r.log.Error("[Monitor] updateWeather failed", "err", err)
		} else {
			labels := []string{
				"self", fmt.Sprintf("%x", r.myNodeInfo.GetMyNodeNum()),
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

func (r *Monitor) HandlePacket(inp *pb.FromRadio) {
	if r == nil || r.myNodeInfo == nil || r.nodeInfo == nil {
		return
	}

	labels := []string{"self", fmt.Sprintf("%x", r.myNodeInfo.GetMyNodeNum())}

	switch v := inp.PayloadVariant.(type) {

	case *pb.FromRadio_Packet:
		inpkt := v.Packet

		if inpkt.GetFrom() == 0 {
			// fishy..
			break
		}

		if inpkt.GetFrom() == r.myNodeInfo.GetMyNodeNum() {
			if d := inpkt.GetDecoded(); d != nil {
				if d.GetReplyId() != 0 || d.GetRequestId() != 0 {
					r.handleResponse(inpkt, d, labels)
				}
			}
		}

		if inpkt.GetFrom() == r.myNodeInfo.GetMyNodeNum() {
			break
		}

		// ignore for UDP-injected packets (UDP broadcast over network)
		if !isRealistic(inpkt) {
			break
		}

		logRX(inpkt, labels)
		logDirect(inpkt, labels)
		logContent(inpkt, labels)
		logSenders(inpkt, labels)
	case *pb.FromRadio_QueueStatus:
		break // ignore
	default:
		r.log.Warn("Unknown packet type", "type", fmt.Sprintf("%T", inp.PayloadVariant))
	}
}

func cloneLabels(l []string) []string {
	out := make([]string, len(l))
	if len(l) > 0 {
		copy(out, l)
	}
	return out
}

func logRX(inpkt *pb.MeshPacket, labels []string) {
	if len(labels) == 0 {
		return
	}
	labels = cloneLabels(labels)

	rxRssi.Update(float64(inpkt.GetRxRssi()), labels...)
	rxSnr.Update(float64(inpkt.GetRxSnr()), labels...)

	if d := inpkt.GetDecoded(); d != nil {
		labels = append(labels, "port", d.GetPortnum().String())
	} else {
		labels = append(labels, "port", "UNKNOWN_APP")
	}

	groups[0].AddOne(&totalRX, labels...)
}

func isRealistic(inpkt *pb.MeshPacket) bool {
	return inpkt.GetRxRssi() != 0 && inpkt.GetRxRssi() < -20 &&
		inpkt.GetRxSnr() < 20 && inpkt.GetRxSnr() != 0 && inpkt.GetRxSnr() != -32
}

func logDirect(inpkt *pb.MeshPacket, labels []string) {
	if len(labels) == 0 {
		return
	}
	labels = cloneLabels(labels)

	hopsAway := int(meshtastic.HopsAway(inpkt))
	if hopsAway > 0 {
		return
	}

	isStrong := inpkt.GetRxRssi() > -105 && inpkt.GetRxSnr() > -5
	labels = append(labels, "strong", boolStr[isRealistic(inpkt) && isStrong])
	groups[0].AddOne(&rxDirect, labels...)
}

func logSenders(inpkt *pb.MeshPacket, labels []string) {
	if len(labels) == 0 {
		return
	}
	labels = cloneLabels(labels)

	hopsAway := int(meshtastic.HopsAway(inpkt))
	labels = append(labels, "from", fmt.Sprintf("%x", inpkt.GetFrom()))

	if hopsAway <= 3 {
		groups[0].AddOne(&rxProximity, labels...)
	}

	labels = append(labels, "hops", strconv.Itoa(hopsAway))
	groups[1].AddOne(&senders, labels...)
}

func logContent(inpkt *pb.MeshPacket, labels []string) {
	if len(labels) == 0 {
		return
	}
	labels = cloneLabels(labels)

	d := inpkt.GetDecoded()
	if d == nil {
		groups[0].AddOne(&totalUnknown, labels...)
		return
	}

	groups[0].AddOne(&totalDecoded, labels...)
}

func (r *Monitor) handleResponse(inpkt *pb.MeshPacket, d *pb.Data, labels []string) {
	if r == nil || len(labels) == 0 {
		return
	}

	labels = cloneLabels(labels)

	switch d.GetPortnum() {
	case pb.PortNum_TELEMETRY_APP:
		var telemetry pb.Telemetry
		err := proto.Unmarshal(d.GetPayload(), &telemetry)
		if err != nil {
			r.log.Error("Failed to Unmarshall telemetry packet", "err", err, "requestId", d.GetRequestId(), "replyId", d.GetReplyId(), "id", inpkt.Id)
			break
		}

		r.log.Debug("Received telemetry", "type", fmt.Sprintf("%T", telemetry.GetVariant()))
		switch t := telemetry.GetVariant().(type) {

		case *pb.Telemetry_DeviceMetrics:
			r.log.Debug("Device metrics received")
			groups[1].Update(&chUtil, float64(t.DeviceMetrics.GetChannelUtilization()), labels...)
			groups[1].Update(&airUtilTx, float64(t.DeviceMetrics.GetAirUtilTx()), labels...)

		case *pb.Telemetry_LocalStats:
			r.log.Debug("Local stats received")

			newBadPackets := t.LocalStats.GetNumPacketsRxBad()
			r.log.Debug("Received Bad packets", "newBadPackets", newBadPackets, "oldBadPackets", oldBadPackets)

			cRxBad, err := libmetric.MakeSeries("rx_bad",
				"self", fmt.Sprintf("%x", r.myNodeInfo.GetMyNodeNum()),
			)
			if err != nil {
				r.log.Error("Failed to MakeSeries for rx_bad", "err", err)
				break
			}

			if oldBadPackets == 0 {
				oldBadPackets = newBadPackets
			} else if newBadPackets > oldBadPackets {
				cRxBad.Add(float64(newBadPackets) - float64(oldBadPackets))
				err := cRxBad.Commit()
				if err != nil {
					r.log.Error("Failed to commit update on rx_bad", "err", err)
					break
				}

				oldBadPackets = newBadPackets
			}

		default:
			r.log.Debug("Unhandled telemetry type", "replyId", d.GetReplyId(), "id", inpkt.GetId(), "requestId", d.GetRequestId())
		}
	default:
		r.log.Debug("Unhandled response type", "replyId", d.GetReplyId(), "id", inpkt.GetId(), "requestId", d.GetRequestId())
	}
}
