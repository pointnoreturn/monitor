package meshtastic

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	pb "github.com/pointnoreturn/snake/github.com/meshtastic/go/generated"
	"github.com/pointnoreturn/snake/libsnake"
	"google.golang.org/protobuf/proto"
)

// TODO: meshtastic stuff here on protoStream primitive?
const start1 = byte(0x94)
const start2 = byte(0xc3)
const headerLen = 4
const maxToFromRadioSzie = 512

// read and write Meshtastic Protobuf packets on the underrelying Stream using magic byte codings
type protoStream struct {
	libsnake.BaseStream
	libsnake.Writer[*pb.ToRadio]
	libsnake.Reader[*pb.FromRadio]
}

func (r *protoStream) WritePacket(
	ctx context.Context,
	p *pb.ToRadio,
) error {
	protobufPacket, err := proto.Marshal(p)
	if err != nil {
		return err
	}

	packageLength := len(protobufPacket)

	header := []byte{
		start1,
		start2,
		byte(packageLength>>8) & 0xff,
		byte(packageLength) & 0xff,
	}

	radioPacket := append(header, protobufPacket...)

	return r.Write(ctx, radioPacket)
}

// ReadResponse reads any responses in the serial port, convert them to a FromRadio protobuf and return
func (r *protoStream) ReadPackets(ctx context.Context, timeout bool) (FromRadioPackets []*pb.FromRadio, err error) {
	readCtx, cancel := context.WithTimeout(
		ctx,
		5*time.Second,
	)
	defer cancel()

	b := make([]byte, 1)

	emptyByte := make([]byte, 0)
	processedBytes := make([]byte, 0)
	repeatByteCounter := 0
	previousByte := make([]byte, 1)
	/************************************************************************************************
	* Process the returned data byte by byte until we have a valid command
	* Each command will come back with [START1, START2, PROTOBUF_PACKET]
	* where the protobuf packet is sent in binary. After reading START1 and START2
	* we use the next bytes to find the length of the packet.
	* After finding the length the looop continues to gather bytes until the length of the gathered
	* bytes is equal to the packet length plus the header
	 */
	for {

		err := r.Read(readCtx, b)
		// fmt.Printf("Byte: %q\n", b)
		if bytes.Equal(b, previousByte) {
			repeatByteCounter++
		} else {
			repeatByteCounter = 0
		}
		// Only break on repeated bytes if we're not in the middle of reading a valid packet
		shouldBreakOnRepeat := repeatByteCounter > 20 && (len(processedBytes) < headerLen)

		if errors.Is(err, context.DeadlineExceeded) {
			err = nil
			if len(processedBytes) > 0 { // in the middle of reading packet
				// Hmm we would be able to recover in this case and continue using stream.
			}
			return FromRadioPackets, nil
		} else if err == io.EOF || shouldBreakOnRepeat || errors.Is(err, context.Canceled) {
			break
		} else if err != nil {
			fmt.Println("return err 1")
			return nil, err
		}
		copy(previousByte, b)

		if len(b) > 0 {

			pointer := len(processedBytes)

			processedBytes = append(processedBytes, b...)

			if pointer == 0 {
				if b[0] != start1 {
					processedBytes = emptyByte
				}
			} else if pointer == 1 {
				if b[0] != start2 {
					processedBytes = emptyByte
				}
			} else if pointer >= headerLen {
				packetLength := int(processedBytes[2])<<8 | int(processedBytes[3])

				if pointer == headerLen {
					if packetLength > maxToFromRadioSzie {
						processedBytes = emptyByte
					}
				}

				if len(processedBytes) != 0 && pointer+1 == packetLength+headerLen {
					fromRadio := pb.FromRadio{}
					if err := proto.Unmarshal(processedBytes[headerLen:], &fromRadio); err != nil {
						fmt.Println("return err 2")
						return nil, err
					}
					FromRadioPackets = append(FromRadioPackets, &fromRadio)
					processedBytes = emptyByte
				}
			}

		} else {
			break
		}

	}

	return FromRadioPackets, nil

}

// GetRadioInfo retrieves information from the radio including config and adjacent Node information
// Right after TCP dial is finished
func (r *protoStream) WantConfig(ctx context.Context, id uint32) (radioResponses []*pb.FromRadio, err error) {
	nodeInfo := pb.ToRadio{PayloadVariant: &pb.ToRadio_WantConfigId{WantConfigId: id}} // only want self node info

	err = r.WritePacket(ctx, &nodeInfo)
	if err != nil {
		return nil, err
	}

	fmt.Println("SendPacketContext success")

	radioResponses, err = r.ReadPackets(ctx, true)
	fmt.Printf("ReadResponseContext returned with err=%v\n", err)
	if err != nil {
		return nil, err
	}

	if len(radioResponses) == 0 {
		return nil, errors.New("failed to get radio info")
	}
	return

}
func (r *protoStream) SendHeartbeat(ctx context.Context, nonce uint32) (err error) {
	// Send first request for Radio and Node information
	nodeInfo := pb.ToRadio{PayloadVariant: &pb.ToRadio_Heartbeat{
		Heartbeat: &pb.Heartbeat{
			Nonce: nonce,
		},
	}} // only want self node info

	return r.WritePacket(ctx, &nodeInfo)
}
