package meshtastic

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pb "github.com/pointnoreturn/monitor/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/monitor/libradios"
)

var (
	ErrAddress = errors.New("Invalid address to connect")
)

// high level protocol client for Meshtastic
type Client struct {
	ProtoStream
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

func NewClient(
	ctx context.Context,
	target string,
	configHandler PacketF,
) (*Client, error) {

	var (
		stream libradios.Transport
		err    error
	)

	if _, isIP := libradios.ParseTCPAddress(target, DefaultNodeTcpPort); isIP {

		stream, err = libradios.NewNetStream(
			ctx,
			target,
			DefaultNodeTcpPort,
		)

		if err != nil {
			return nil, err
		}

	} else if strings.HasPrefix(target, "/") {

		stream, err = libradios.NewSerialStream(
			ctx,
			target,
		)

		if err != nil {
			return nil, err
		}

	} else {
		return nil, fmt.Errorf("%s: %v", target, ErrAddress)
	}

	c := &Client{
		ProtoStream: ProtoStream{
			Transport: stream,
		},
		Port: target,
	}

	fmt.Println("running Client.intiialize()")

	myNodeInfo, nodes, err := c.initialize(
		ctx,
		ConfigId_ConfigOnly,
		configHandler,
	)

	if err != nil {
		c.Close()
		return nil, fmt.Errorf(
			"Failed initialize for %s: %v",
			target,
			err,
		)
	}

	if myNodeInfo == nil || len(nodes) < 1 {
		return nil, errors.New("safety check failed")
	}

	if myNodeInfo.MyNodeNum != nodes[0].Num {
		return nil, fmt.Errorf(
			"MyNodeInfo Num %d (!%x) does not match first NodeInfo entry Num %d (safety check failed)",
			myNodeInfo.MyNodeNum,
			myNodeInfo.MyNodeNum,
			nodes[0].Num,
		)
	}

	c.Label = GetNodeLabel(nodes[0])
	c.MyNode = myNodeInfo
	c.NodeDB = nodes

	return c, nil
}

func (c *Client) initialize(ctx context.Context, configId uint32, configHandler PacketF) (*pb.MyNodeInfo, []*pb.NodeInfo, error) {
	nodes := []*pb.NodeInfo{}
	myNodeInfo, responses, err := c.initializeBase(ctx, configId, true)
	if err != nil {
		return myNodeInfo, nodes, err
	}

	if configHandler == nil {
		configHandler = func(*pb.FromRadio) {}
	}

	for _, p := range responses {
		configHandler(p)
		if n := p.GetNodeInfo(); n != nil {
			nodes = append(nodes, n)
		}
	}

	return myNodeInfo, nodes, err
}

func (c *Client) initializeBase(ctx context.Context, configId uint32, verifyCompleteId bool) (*pb.MyNodeInfo, []*pb.FromRadio, error) {
	fmt.Println("[initializeBase] call WantConfig")

	responses, err := c.WantConfig(ctx, configId)
	if err != nil {
		return nil, responses, err
	}

	fmt.Println("[initializeBase] WantConfig() returned no error")

	fmt.Printf("DEBUG: [initializeBase] WantConfig(%d) at %s got %d responses\n", configId, c.Port, len(responses))

	var myNodeInfo *pb.MyNodeInfo
	for i, p := range responses {
		fmt.Printf("Response %d %T\n", i, p.PayloadVariant)
		if info := p.GetMyInfo(); info != nil {
			fmt.Println("assigned myNodeInfo")
			myNodeInfo = info
			fmt.Printf("Data: %v\n", myNodeInfo)
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
