package server

import (
	"github.com/huangw1/rpc-demo/step-3/protocol"
	"github.com/huangw1/rpc-demo/step-3/codec"
	"github.com/huangw1/rpc-demo/step-3/transport"
	"time"
)

type Option struct {
	ProtocolType protocol.ProtocolType
	SerializeType codec.SerializeType
	CompressType protocol.CompressType
	TransportType transport.TransportType

	RequestTimeout time.Duration
}

var DefaultOption = Option{
	ProtocolType: protocol.Default,
	SerializeType: codec.MessagePack,
	CompressType: protocol.CompressTypeNone,
	TransportType: transport.TCPTransport,
	RequestTimeout: time.Second * 60,
}