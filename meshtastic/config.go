package meshtastic

import (
	"context"
	"errors"
	"fmt"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
)

// firmware consts for want_config_id:
// #define SPECIAL_NONCE_ONLY_CONFIG 69420
// #define SPECIAL_NONCE_ONLY_NODES 69421 // ( ͡° ͜ʖ ͡°)
const ConfigId_OnlyNodes = 69421
const ConfigId_ConfigOnly = 69420 // Using ConfigId_ConfigOnly to omit full NodeDB sync

// send a heartbeat, node will send a response to keep connection alive if needed
// in TCP, losing connection may remain undetected without trying send anything.
func Heartbeat(ctx context.Context, stream Writer, nonce uint32) (err error) {
	toRadio := pb.ToRadio{PayloadVariant: &pb.ToRadio_Heartbeat{
		Heartbeat: &pb.Heartbeat{
			Nonce: nonce,
		},
	}}

	err = stream.WritePacket(ctx, &toRadio)

	return err
}

// shortcut to send packets to node
func Send(ctx context.Context, stream Writer, toRadio *pb.ToRadio) (err error) {
	return stream.WritePacket(ctx, toRadio)
}

// receiving config response is what initializes connection to PhoneAPI and makes this client "subscribed"
// WantConfig may return node info, settings, node db during the initialization as series of packets to receive,
// read them all as radioResponses
// TODO: timeouts and receive full node db?? Use MyNodeInfo.NodedbCount to ensure full nodedb is received
func WantConfig(ctx context.Context, stream *ProtoStream, id uint32) (radioResponses []*pb.FromRadio, err error) {
	toRadio := pb.ToRadio{PayloadVariant: &pb.ToRadio_WantConfigId{WantConfigId: id}} // only want self node info

	stream.Log.Debug("[WantConfig] call WritePacket")
	err = stream.WritePacket(ctx, &toRadio)
	if err != nil {
		return nil, err
	}

	stream.Log.Debug("[WantConfig] call ReadPackets")

	radioResponses, err = stream.ReadPackets(ctx, true)
	if err != nil {
		return nil, err
	}

	stream.Log.Debug(fmt.Sprintf("[WantConfig] no error, %d responses read.\n", len(radioResponses)))

	if len(radioResponses) == 0 {
		return nil, errors.New("failed to get radio info")
	}
	return

}

func WantConfigSequence(ctx context.Context, stream *ProtoStream, configId uint32, verifyCompleteId bool) (*pb.MyNodeInfo, []*pb.FromRadio, error) {
	stream.Log.Debug("[WantConfigSequence] call WantConfig")

	responses, err := WantConfig(ctx, stream, configId)
	if err != nil {
		return nil, responses, err
	}

	stream.Log.Debug(fmt.Sprintf("[WantConfigSequence] WantConfig(%d) got %d responses", configId, len(responses)))

	var myNodeInfo *pb.MyNodeInfo
	for i, p := range responses {
		stream.Log.Debug(fmt.Sprintf("[WantConfigSequence] Response %d %T", i, p.PayloadVariant))
		if info := p.GetMyInfo(); info != nil {
			myNodeInfo = info
			stream.Log.Debug(fmt.Sprintf("[WantConfigSequence] myNodeInfo Data: %+v", myNodeInfo))
		}
	}

	if myNodeInfo == nil {
		return nil, responses, errors.New("MyNodeInfo packet was missing the response.")
	}

	if verifyCompleteId {
		var completeId uint32 = 0
		for _, p := range responses {
			if complete := p.GetConfigCompleteId(); complete != 0 && completeId == 0 {
				if complete == configId {
					completeId = complete
					break
				}
			}
		}

		if completeId != configId {
			return myNodeInfo, responses, fmt.Errorf("config_complete_id expected with value %d, have %d.", configId, completeId)
		}
	}

	stream.Log.Debug("[WantConfigSequence] end")

	return myNodeInfo, responses, nil
}
