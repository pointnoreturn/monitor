package meshtastic

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libradios"
)

var (
	ErrBrowseNotFound = errors.New("Node not found in browse mode.")
)

func ConnectTCP(
	ctx context.Context,
	log *slog.Logger,
	tcpAddr string,
	wantConfigId uint32,
	configHandler PacketF,
) (*ProtoStream, *pb.MyNodeInfo, *pb.NodeInfo, error) {

	stream, err := libradios.NewNetStream(ctx, tcpAddr)

	if err != nil {
		return nil, nil, nil, err
	}

	return createCompletedClient(ctx, log, tcpAddr, stream, wantConfigId, configHandler)
}

func ConnectSerial(
	ctx context.Context,
	log *slog.Logger,
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

	return createCompletedClient(ctx, log, device, stream, wantConfigId, configHandler)
}

func createCompletedClient(
	ctx context.Context,
	log *slog.Logger,
	target string,
	pipe libradios.Transport,
	wantConfigId uint32,
	configHandler PacketF,
) (*ProtoStream, *pb.MyNodeInfo, *pb.NodeInfo, error) {
	stream := &ProtoStream{
		Transport: pipe,
		Log:       log,
	}

	stream.Log.Debug("call Client.intiialize()")

	myNodeInfo, responses, err := WantConfigSequence(ctx, stream, wantConfigId, true)
	if err != nil {
		stream.Close()
		return nil, myNodeInfo, nil, err
	}

	if configHandler == nil {
		configHandler = func(*pb.FromRadio) {}
	}

	var nodeInfo *pb.NodeInfo

	stream.Log.Debug(fmt.Sprintf("[createCompletedClient] %d responses\n", len(responses)))

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

// creates completed configured connection to a node using TARGET_NODE specification:
// either "/dev/ttyUSB0" like system path
// or raw IP, IP:port
// or node label like {short_name}_{last_bytes_hex} when a resolved node sends Bonjour broadcasts (announce) on the local network.
// wantConfigId: See PhoneAPI
// handleConfig: Packet handler before configuration is completed.
func FindAndConnect(ctx context.Context, log *slog.Logger, targetNode string, timeout time.Duration, wantConfigId uint32, handleConfig PacketF) (*ProtoStream, *pb.MyNodeInfo, *pb.NodeInfo, error) {
	// serial device is a path
	if strings.Index(targetNode, "/") == 0 {
		stream, myNodeInfo, nodeInfo, err := ConnectSerial(ctx, log, targetNode, wantConfigId, handleConfig)
		if err != nil {
			err := fmt.Errorf("Failed to connect serial '%s': %w", targetNode, err)
			return nil, nil, nil, err
		}
		return stream, myNodeInfo, nodeInfo, nil
	}

	// non-path target
	// check if it is IP, IP:port, [IPv6]:port or IPv6
	if ipEndpoint, isIpEndpoint := libradios.ParseTCPAddress(targetNode, fmt.Sprintf("%d", DefaultPort)); isIpEndpoint {
		log.Info(fmt.Sprintf("Parsed %s as IP endpoint %s", targetNode, ipEndpoint))
		stream, myNodeInfo, nodeInfo, err := ConnectTCP(ctx, log, ipEndpoint, wantConfigId, handleConfig)
		if err != nil {
			err := fmt.Errorf("Failed ConnectTCP '%s': %w", targetNode, err)
			return nil, nil, nil, err
		}

		return stream, myNodeInfo, nodeInfo, nil
	}

	// Non-IP format string, ASSUME: node broadcast on the local network (NOT hostname)
	// assume NODE number, node Label
	// resolve on LAN using mDNS broadcasts, match by meshtastic node label or hex num
	log.Info(fmt.Sprintf("[FindAndConnect] Browse advertised meshtastic nodes on the network timeout=%s", timeout))

	timeoutContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	services := make(chan *libradios.BroadcastService)
	nodes := make(chan *BroadcastNode)

	go libradios.BrowseBroadcasts(timeoutContext, log, services)
	go BrowseNodes(timeoutContext, log, services, nodes)

	for {
		select {
		case <-ctx.Done():
			return nil, nil, nil, ctx.Err()
		case <-timeoutContext.Done():
			return nil, nil, nil, timeoutContext.Err()
		case n := <-nodes:
			if n == nil || n.Service == nil || n.Service.Entry == nil {
				return nil, nil, nil, ErrBrowseNotFound
			}

			log.Debug(fmt.Sprintf("Node: %+v\n", n))

			if !MatchNode(targetNode, n) {
				continue
			}

			log.Debug(fmt.Sprintf("[FindAndConnect] Connect to node %s\n", n.Service.Endpoint))
			stream, myNodeInfo, nodeInfo, err := ConnectTCP(ctx, log, n.Service.Endpoint, wantConfigId, handleConfig)
			if err != nil {
				err := fmt.Errorf("Failed ConnectTCP '%s': %w", targetNode, err)
				return nil, nil, nil, err
			}

			return stream, myNodeInfo, nodeInfo, nil

		}
	}
}
