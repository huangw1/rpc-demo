package protocol

import (
	"github.com/huangw1/rpc-demo/step-3/codec"
	"io"
	"errors"
	"encoding/binary"
	"github.com/vmihailenco/msgpack"
)

/**
-------------------------------------------------------------------------------------------------
|2byte|1byte  |4byte       |4byte        | header length |(total length - header length - 4byte)|
-------------------------------------------------------------------------------------------------
|magic|version|total length|header length|     header    |                    body              |
-------------------------------------------------------------------------------------------------
 */

const (
	RequestSeqKey     = "rpc_request_seq"
	RequestTimeoutKey = "rpc_request_timeout"
	MetaDataKey       = "rpc_meta_data"
)

type MessageType byte

const (
	MessageTypeReq MessageType = iota
	MessageTypeRes
)

type CompressType byte

const (
	CompressTypeNone CompressType = iota
)

type StatusCode byte

const (
	StatusOk    StatusCode = iota
	StatusError
)

type ProtocolType byte

const (
	Default ProtocolType = iota
)

var protocols = map[ProtocolType]Protocol{
	Default: &RPCProtocol{},
}

func NewMessage(t ProtocolType) *Message {
	return protocols[t].NewMessage()
}

func DecodeMessage(t ProtocolType, r io.Reader) (*Message, error) {
	return protocols[t].DecodeMessage(r)
}

func EncodeMessage(t ProtocolType, m *Message) []byte {
	return protocols[t].EncodeMessage(m)
}

type Header struct {
	Seq           uint64
	MessageType   MessageType
	CompressType  CompressType
	SerializeType codec.SerializeType
	StatusCode    StatusCode
	ServiceName   string
	MethodName    string
	Error         string
	MetaData      map[string]string
}

type Message struct {
	*Header
	Data []byte
}

func (m *Message) Clone() *Message {
	header := *m.Header
	return &Message{
		Header: &header,
		Data:   m.Data,
	}
}

type Protocol interface {
	NewMessage() *Message
	DecodeMessage(r io.Reader) (*Message, error)
	EncodeMessage(m *Message) []byte
}

type RPCProtocol struct {
}

func (RPCProtocol) NewMessage() *Message {
	return &Message{Header: &Header{}}
}

func (RPCProtocol) DecodeMessage(r io.Reader) (msg *Message, err error) {
	firstBytes := make([]byte, 3)
	_, err = io.ReadFull(r, firstBytes[:2])
	if err != nil {
		return
	}
	if !checkMagic(firstBytes) {
		err = errors.New("wrong protocol")
		return
	}
	totalBytes := make([]byte, 4)
	io.ReadFull(r, totalBytes)
	if err != nil {
		return
	}
	totalLen := int(binary.BigEndian.Uint32(totalBytes))
	if totalLen < 4 {
		err = errors.New("invalid total length")
		return
	}
	data := make([]byte, totalLen)
	_, err = io.ReadFull(r, data)
	if err != nil {
		return
	}
	headLen := int(binary.BigEndian.Uint32(data[:4]))
	headBytes := data[4 : headLen+4]
	header := &Header{}
	err = msgpack.Unmarshal(headBytes, header)
	if err != nil {
		return
	}
	msg = &Message{}
	msg.Header = header
	msg.Data = data[headLen+4:]
	return
}

func (RPCProtocol) EncodeMessage(m *Message) []byte {
	firstBytes := []byte{0xab, 0xba, 0x00}
	headBytes, _ := msgpack.Marshal(m.Header)

	totalLen := 4 + len(headBytes) + len(m.Data)
	totalLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(totalLenBytes, uint32(totalLen))

	start := 0
	data := make([]byte, totalLen+7)
	copyBytesOffset(data, firstBytes, &start)
	copyBytesOffset(data, totalLenBytes, &start)

	headerLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(headerLenBytes, uint32(len(headBytes)))
	copyBytesOffset(data, headerLenBytes, &start)
	copyBytesOffset(data, headBytes, &start)
	copyBytesOffset(data, m.Data, &start)
	return nil
}

func checkMagic(bytes []byte) bool {
	return bytes[0] == 0xab && bytes[1] == 0xba
}

func copyBytesOffset(dst []byte, src []byte, start *int) {
	copy(dst[*start:len(src)], src)
	*start += len(src)
}
