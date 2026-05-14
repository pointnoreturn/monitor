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
const ConfigId_ConfigOnly = 69420

// receiving config response is what initializes connection to PhoneAPI and makes this client "subscribed"
// WantConfig may return node info, settings, node db during the initialization as series of packets to receive,
// read them all as radioResponses
// TODO: timeouts and receive full node db?? Use MyNodeInfo.NodedbCount to ensure full nodedb is received
func (c *Client) WantConfig(ctx context.Context, id uint32) (radioResponses []*pb.FromRadio, err error) {
	toRadio := pb.ToRadio{PayloadVariant: &pb.ToRadio_WantConfigId{WantConfigId: id}} // only want self node info

	fmt.Println("[WantConfig] call WritePacket")
	err = c.ProtoStream.WritePacket(ctx, &toRadio)
	if err != nil {
		return nil, err
	}

	fmt.Println("[WantConfig] call ReadPackets(timeout: true)")

	radioResponses, err = c.ProtoStream.ReadPackets(ctx, true)
	if err != nil {
		return nil, err
	}

	fmt.Printf("[WantConfig] no error, %d responses read.\n", len(radioResponses))

	if len(radioResponses) == 0 {
		return nil, errors.New("failed to get radio info")
	}
	return

}
