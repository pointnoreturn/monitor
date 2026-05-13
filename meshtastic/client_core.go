package meshtastic

import (
	"context"
	"errors"

	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
)

// firmware consts for want_config_id:
// #define SPECIAL_NONCE_ONLY_CONFIG 69420
// #define SPECIAL_NONCE_ONLY_NODES 69421 // ( ͡° ͜ʖ ͡°)
const ConfigId_OnlyNodes = 69421
const ConfigId_ConfigOnly = 69420

// receiving config response is what initializes connection to PhoneAPI and makes this client "subscribed"
// WantConfig may return node info, settings, node db during the initialization as series of packets to receive,
// read them all as radioResponses
// TODO: timeouts and receive full node db?? Use MyNodeInfo.NodedbCount to ensure full nodedb is received
func (r *ProtoStream) WantConfig(ctx context.Context, id uint32) (radioResponses []*pb.FromRadio, err error) {
	nodeInfo := pb.ToRadio{PayloadVariant: &pb.ToRadio_WantConfigId{WantConfigId: id}} // only want self node info

	err = r.WritePacket(ctx, &nodeInfo)
	if err != nil {
		return nil, err
	}

	radioResponses, err = r.ReadPackets(ctx, true)
	if err != nil {
		return nil, err
	}

	if len(radioResponses) == 0 {
		return nil, errors.New("failed to get radio info")
	}
	return

}

func (r *ProtoStream) SendHeartbeat(ctx context.Context, nonce uint32) (err error) {
	nodeInfo := pb.ToRadio{PayloadVariant: &pb.ToRadio_Heartbeat{
		Heartbeat: &pb.Heartbeat{
			Nonce: nonce,
		},
	}}

	return r.WritePacket(ctx, &nodeInfo)
}
