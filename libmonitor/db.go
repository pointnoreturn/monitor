package libmonitor

import (
	"context"
	"fmt"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/meshtastic"
	"google.golang.org/protobuf/proto"
)

type DB struct {
	// relayCandidate map[uint32]  TODO: relayByte := uint8(pkt.RelayNode)
}

func (db *DB) Init(ctx context.Context) {

}

func (db *DB) Run(ctx context.Context) {

}

func (db *DB) HandlePacket(p *pb.FromRadio) {
	switch v := p.PayloadVariant.(type) {
	case *pb.FromRadio_NodeInfo:
		db.update(v.NodeInfo)
	case *pb.FromRadio_Packet:
		pkt := v.Packet

		if d := pkt.GetDecoded(); d != nil {
			switch d.Portnum {

			case pb.PortNum_NODEINFO_APP:
				var user pb.User // NODEINFO_APP carries User payloads
				err := proto.Unmarshal(d.Payload, &user)
				if err != nil {
					fmt.Printf("[NodeDB] Unmarshal error from !%x: %v\n", pkt.From, err)
				} else {
					hopsAway := meshtastic.HopsAway(pkt)
					db.updateUser(&user, &hopsAway)
				}
			}
		}
	}
}

func (db *DB) update(nodeInfo *pb.NodeInfo) {
	//fmt.Printf("[NodeDB] NodeInfo !%x, via_mqtt: %v\n", nodeInfo.Num, nodeInfo.ViaMqtt)

	if nodeInfo.User != nil {
		db.updateUser(nodeInfo.User, nodeInfo.HopsAway)
	}
}

func (db *DB) updateUser(user *pb.User, hopsAway *uint32) {
	infos := []string{
		//fmt.Sprintf("User Id '%s' role %s name '%s' '%s'\t", user.Id, user.Role, user.ShortName, user.LongName),
	}
	if hopsAway != nil {
		infos = append(infos, fmt.Sprintf("%d hops away", *hopsAway))
	}
	if user.HwModel != pb.HardwareModel_UNSET {
		if hwName, ok := pb.HardwareModel_name[int32(user.HwModel)]; ok {
			infos = append(infos, fmt.Sprintf("📻 %s (%d)", hwName, user.HwModel))
		} else {
			infos = append(infos, fmt.Sprintf("📻 (%d)", user.HwModel))
		}
		if len(user.PublicKey) > 0 {
			infos = append(infos, "🔑 PKI")
		}
	}
	//fmt.Println("[NodeDB] " + strings.Join(infos, ", "))
}
