package codec

import "github.com/vmihailenco/msgpack"

type SerializeType byte

const (
	MessagePack SerializeType = iota
)

type Codec interface {
	Encode(val interface{}) ([]byte, error)
	Decode(data []byte, val interface{}) error
}

var codecs = map[SerializeType]Codec{
	MessagePack: &MessagePackCodec{},
}

type MessagePackCodec struct {
}

func (m *MessagePackCodec) Encode(val interface{}) ([]byte, error) {
	return msgpack.Marshal(val)
}

func (m *MessagePackCodec) Decode(data []byte, val interface{}) error {
	return msgpack.Unmarshal(data, val)
}

func GetCodec(t SerializeType) Codec {
	return codecs[t]
}
