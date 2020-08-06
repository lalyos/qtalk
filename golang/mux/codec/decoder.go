package codec

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

type Decoder struct {
	r io.Reader
	sync.Mutex
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

func (dec *Decoder) Decode() (Message, error) {
	dec.Lock()
	defer dec.Unlock()

	packet, err := readPacket(dec.r)
	if err != nil {
		return nil, err
	}

	return decode(packet)
}

func readPacket(c io.Reader) ([]byte, error) {
	msgNum := make([]byte, 1)
	_, err := c.Read(msgNum)
	if err != nil {
		return nil, err
	}

	rest := make([]byte, payloadSizes[msgNum[0]])
	_, err = c.Read(rest)
	if err != nil {
		return nil, err
	}

	packet := append(msgNum, rest...)

	if msgNum[0] == msgChannelData {
		dataSize := binary.BigEndian.Uint32(rest[4:8])
		data := make([]byte, dataSize)
		_, err := c.Read(data)
		if err != nil {
			return nil, err
		}

		packet = append(packet, data...)
	}

	return packet, nil
}

func decode(packet []byte) (Message, error) {
	var msg Message
	switch packet[0] {
	case msgChannelOpen:
		msg = new(OpenMessage)
	case msgChannelData:
		msg = new(DataMessage)
	case msgChannelOpenConfirm:
		msg = new(OpenConfirmMessage)
	case msgChannelOpenFailure:
		msg = new(OpenFailureMessage)
	case msgChannelWindowAdjust:
		msg = new(WindowAdjustMessage)
	case msgChannelEOF:
		msg = new(EOFMessage)
	case msgChannelClose:
		msg = new(CloseMessage)
	default:
		return nil, fmt.Errorf("qmux: unexpected message type %d", packet[0])
	}
	if err := Unmarshal(packet, msg); err != nil {
		return nil, err
	}
	// fmt.Println(">>", msg)
	return msg, nil
}

func Unmarshal(b []byte, v interface{}) error {
	switch msg := v.(type) {
	case *OpenMessage:
		msg.SenderID = binary.BigEndian.Uint32(b[1:5])
		msg.WindowSize = binary.BigEndian.Uint32(b[5:9])
		msg.MaxPacketSize = binary.BigEndian.Uint32(b[9:13])
		return nil

	case *OpenConfirmMessage:
		msg.ChannelID = binary.BigEndian.Uint32(b[1:5])
		msg.SenderID = binary.BigEndian.Uint32(b[5:9])
		msg.WindowSize = binary.BigEndian.Uint32(b[9:13])
		msg.MaxPacketSize = binary.BigEndian.Uint32(b[13:17])
		return nil

	case *OpenFailureMessage:
		msg.ChannelID = binary.BigEndian.Uint32(b[1:5])
		return nil

	case *WindowAdjustMessage:
		msg.ChannelID = binary.BigEndian.Uint32(b[1:5])
		msg.AdditionalBytes = binary.BigEndian.Uint32(b[5:9])
		return nil

	case *DataMessage:
		msg.ChannelID = binary.BigEndian.Uint32(b[1:5])
		msg.Length = binary.BigEndian.Uint32(b[5:9])
		msg.Data = b[9:]
		return nil

	case *EOFMessage:
		msg.ChannelID = binary.BigEndian.Uint32(b[1:5])
		return nil

	case *CloseMessage:
		msg.ChannelID = binary.BigEndian.Uint32(b[1:5])
		return nil

	default:
		return fmt.Errorf("qmux: unmarshal not supported for value %#v", v)
	}
}
