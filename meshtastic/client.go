package meshtastic

import (
	"context"
	"errors"
	"fmt"

	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
)

// high level protocol client for Meshtastic
type Client struct {
	streamer
	Port   string         // IP:port, /serial/path, etc
	Label  string         // SHRT_12af node label
	MyNode *pb.MyNodeInfo // populated during connect or manually updated later
	NodeDB []*pb.NodeInfo // populated during connect or manually updated later
}

func (c *Client) String() string {
	if len(c.Label) == 0 {
		return c.Port
	}
	return c.Label
}

func NewClient(ctx context.Context, connectPort string) (*Client, error) {
	// TODO: implemented context for socket/operation

	socket := streamer{}
	err := socket.Init(ctx, connectPort, DefaultNodeTcpPort)
	if err != nil {
		return nil, err
	}

	c := &Client{
		streamer: socket,
		Port:     connectPort,
	}

	myNodeInfo, nodes, err := c.initializeNodes(ctx, ConfigId_ConfigOnly)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("Failed NewClient for %s: %v", connectPort, err)
	}

	if myNodeInfo == nil || len(nodes) < 1 {
		return nil, errors.New("Safety check failed")
	} else if myNodeInfo.MyNodeNum != nodes[0].Num {
		return nil, fmt.Errorf("MyNodeInfo Num %d (!%x) does not match first NodeInfo entry Num %d (safety check failed)", myNodeInfo.MyNodeNum, myNodeInfo.MyNodeNum, nodes[0].Num)
	}

	c.Label = GetNodeLabel(nodes[0])
	c.MyNode = myNodeInfo
	c.NodeDB = nodes
	return c, nil
}

func (c *Client) initializeNodes(ctx context.Context, configId uint32) (*pb.MyNodeInfo, []*pb.NodeInfo, error) {
	nodes := []*pb.NodeInfo{}
	myNodeInfo, responses, err := c.initializeBase(ctx, configId, true)
	if err != nil {
		return myNodeInfo, nodes, err
	}

	for _, p := range responses {
		if n := p.GetNodeInfo(); n != nil {
			nodes = append(nodes, n)
		}
	}

	return myNodeInfo, nodes, err
}

func (c *Client) initializeBase(ctx context.Context, configId uint32, verifyCompleteId bool) (*pb.MyNodeInfo, []*pb.FromRadio, error) {

	responses, err := c.streamer.QueryWantConfig(ctx, configId)
	if err != nil {
		return nil, responses, err
	}

	fmt.Printf("DEBUG: [initializeBase] QueryWantConfig(%d) at %s got %d responses\n", configId, c.Port, len(responses))

	var myNodeInfo *pb.MyNodeInfo
	for _, p := range responses {
		if info := p.GetMyInfo(); info != nil && myNodeInfo == nil {
			myNodeInfo = info
		}
	}

	if myNodeInfo == nil {
		return nil, responses, errors.New("MyNodeInfo packet was missing the response.")
	}

	if verifyCompleteId {
		var completeId uint32 = 0
		for _, p := range responses {
			// Return FIRST node info assuming FIRST == SELF
			if complete := p.GetConfigCompleteId(); complete != 0 && completeId == 0 {
				if complete == configId {
					completeId = complete
				}
			}
		}
		if completeId != configId {
			return myNodeInfo, responses, fmt.Errorf("config_complete_id expected with value %d, but the receive was %d or not sent by the node.", configId, completeId)
		}
	}

	return myNodeInfo, responses, nil
}
