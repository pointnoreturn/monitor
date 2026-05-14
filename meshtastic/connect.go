package meshtastic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libradios"
)

func ConnectTCP(
	ctx context.Context,
	address string,
	defaultPort int,
	wantConfigId uint32,
	configHandler PacketF,
) (*ProtoStream, *pb.MyNodeInfo, *pb.NodeInfo, error) {

	stream, err := libradios.NewNetStream(
		ctx,
		address,
		fmt.Sprintf("%d", defaultPort),
	)

	if err != nil {
		return nil, nil, nil, err
	}

	return createCompletedClient(ctx, address, stream, wantConfigId, configHandler)
}

func ConnectSerial(
	ctx context.Context,
	device string,
	wantConfigId uint32,
	configHandler PacketF,
) (*ProtoStream, *pb.MyNodeInfo, *pb.NodeInfo, error) {

	stream, err := libradios.NewSerialStream(
		ctx,
		device,
	)

	if err != nil {
		return nil, nil, nil, err
	}

	return createCompletedClient(ctx, device, stream, wantConfigId, configHandler)
}

func createCompletedClient(
	ctx context.Context,
	target string,
	pipe libradios.Transport,
	wantConfigId uint32,
	configHandler PacketF,
) (*ProtoStream, *pb.MyNodeInfo, *pb.NodeInfo, error) {
	stream := &ProtoStream{
		Transport: pipe,
	}

	fmt.Println("call Client.intiialize()")

	myNodeInfo, responses, err := WantConfigSequence(ctx, stream, wantConfigId, true)
	if err != nil {
		stream.Close()
		return nil, myNodeInfo, nil, err
	}

	if configHandler == nil {
		configHandler = func(*pb.FromRadio) {}
	}

	var nodeInfo *pb.NodeInfo

	fmt.Printf("[loadConfigResponse] %d responses\n", len(responses))

	for _, p := range responses {

		if n := p.GetNodeInfo(); n != nil {
			if n.Num == myNodeInfo.MyNodeNum {
				nodeInfo = n
			}
		}

		configHandler(p)
	}

	if err != nil {
		stream.Close()
		return nil, myNodeInfo, nil, fmt.Errorf(
			"Failed initialize for %s: %v",
			target,
			err,
		)
	}

	if myNodeInfo == nil {
		return nil, myNodeInfo, nil, errors.New("safety check failed (no MyNodeInfo)")
	} else if nodeInfo == nil {
		return nil, myNodeInfo, nil, errors.New("safety check failed (no NodeInfo)")
	} else if nodeInfo.User == nil || nodeInfo.Num != myNodeInfo.MyNodeNum {
		return nil, myNodeInfo, nil, errors.New("safety check failed (invalid NodeInfo)")
	}

	return stream, myNodeInfo, nodeInfo, nil
}

func AutoConnect(ctx context.Context, targetNode string, timeout time.Duration, wantConfigId uint32, handleConfig PacketF) (*ProtoStream, *pb.MyNodeInfo, *pb.NodeInfo, error) {
	// serial device is a path
	if strings.Index(targetNode, "/") == 0 {
		stream, myNodeInfo, nodeInfo, err := ConnectSerial(ctx, targetNode, wantConfigId, handleConfig)
		if err != nil {
			err := fmt.Errorf("Failed to connect serial '%s': %w", targetNode, err)
			return nil, nil, nil, err
		}
		return stream, myNodeInfo, nodeInfo, nil
	}

	// non-path target
	// assume NODE number, node Label (TODO)
	// discover on LAN using mDNS scan, match by meshtastic node label or hex num

	fmt.Println("Discover advertised meshtastic nodes on the network")
	all := libradios.Discover(context.Background(), 5*time.Second)

	fmt.Printf("Find target node '%s' among %d services\n", targetNode, len(all))
	nodes := ListNodes(all)
	node := FindNode(targetNode, nodes)
	if node == nil {
		err := fmt.Errorf("Node not found using mDNS scan and matching: '%s' (retry/longer scan may fix resolution)", targetNode)
		return nil, nil, nil, err
	}

	fmt.Printf("Connect to node %s\n", node.Service.Endpoint)
	stream, myNodeInfo, nodeInfo, err := ConnectTCP(ctx, node.Service.Endpoint, DefaultPort, wantConfigId, handleConfig)
	if err != nil {
		err := fmt.Errorf("Failed to connect tcp using discovery '%s': %w", targetNode, err)
		return nil, nil, nil, err
	}

	return stream, myNodeInfo, nodeInfo, nil
}
