package client

import (
	"errors"
	"context"
	"github.com/huangw1/rpc-demo/step-3/codec"
	"sync"
	"github.com/huangw1/rpc-demo/step-3/transport"
	"github.com/huangw1/rpc-demo/step-3/protocol"
	"strings"
	"log"
	"sync/atomic"
	"time"
)

var ErrorShutdown = errors.New("client is shut down")
var ErrorTimeout = errors.New("request timeout")

type RPCClient interface {
	Go(ctx context.Context, serviceName string, arg interface{}, reply interface{}, done chan *Call) *Call
	Call(ctx context.Context, serviceName string, arg interface{}, reply interface{}) error
	Close() error
}

type Call struct {
	ServiceMethod string
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *Call
}

func (c *Call) done() {
	c.Done <- c
}

type simpleClient struct {
	codec        codec.Codec
	rwc          transport.Transport
	pendingCalls sync.Map
	mutex        sync.Mutex
	shutdown     bool
	option       Option
	seq          int64
}

func NewSimpleClient(network, addr string, option Option) (RPCClient, error) {
	c := new(simpleClient)
	c.codec = codec.GetCodec(option.SerializeType)
	t := transport.NewTransport(option.TransportType)
	err := t.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	c.rwc = t
	go c.input()
	return c, nil
}

func (c *simpleClient) Go(ctx context.Context, serviceName string, arg interface{}, reply interface{}, done chan *Call) *Call {
	call := new(Call)
	call.ServiceMethod = serviceName
	call.Args = arg
	call.Reply = reply
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		panic("rpc: done channel is unbuffered")
	}
	call.Done = done
	c.send(ctx, call)
	return call
}

func (c *simpleClient) send(ctx context.Context, call *Call) {
	seq := ctx.Value(protocol.RequestSeqKey).(uint64)
	c.pendingCalls.Store(seq, call)
	serviceMethod := strings.SplitN(call.ServiceMethod, ".", 2)
	req := protocol.NewMessage(c.option.ProtocolType)
	req.ServiceName = serviceMethod[0]
	req.MethodName = serviceMethod[1]
	req.SerializeType = c.option.SerializeType
	req.CompressType = c.option.CompressType
	req.Seq = seq
	if ctx.Value(protocol.MetaDataKey) != nil {
		req.MetaData = ctx.Value(protocol.MetaDataKey).(map[string]string)
	}
	requestData, err := c.codec.Encode(call.Args)
	if err != nil {
		log.Println(err)
		c.pendingCalls.Delete(seq)
		call.Error = err
		call.done()
		return
	}
	req.Data = requestData
	data := protocol.EncodeMessage(c.option.ProtocolType, req)
	_, err = c.rwc.Write(data)
	if err != nil {
		log.Println(err)
		c.pendingCalls.Delete(seq)
		call.Error = err
		call.done()
		return
	}
}

func (c *simpleClient) Call(ctx context.Context, serviceName string, arg interface{}, reply interface{}) error {
	seq := atomic.AddInt64(&c.seq, 1)
	ctx = context.WithValue(ctx, protocol.RequestSeqKey, seq)
	cancelFunc := func() {}
	if c.option.RequestTimeout != time.Duration(0) {
		ctx, cancelFunc = context.WithTimeout(ctx, c.option.RequestTimeout)
		metaData := ctx.Value(protocol.MetaDataKey)
		var meta map[string]string
		if metaData == nil {
			meta = make(map[string]string)
		} else {
			meta = metaData.(map[string]string)
		}
		meta[protocol.MetaDataKey] = c.option.RequestTimeout.String()
		ctx = context.WithValue(ctx, protocol.MetaDataKey, meta)
	}
	done := make(chan *Call)
	call := c.Go(ctx, serviceName, arg, reply, done)
	select {
	case <-ctx.Done():
		cancelFunc()
		c.pendingCalls.Delete(seq)
		call.Error = ErrorTimeout
		call.done()
	case <-call.Done:

	}
	return call.Error
}

func (c *simpleClient) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.shutdown = true
	err := c.rwc.Close()
	c.pendingCalls.Range(func(key, value interface{}) bool {
		call, ok := key.(*Call)
		if ok {
			call.Error = ErrorShutdown
			call.done()
		}
		c.pendingCalls.Delete(key)
		return true
	})
	return err
}

func (c *simpleClient) input() {
	var err error
	var res *protocol.Message
	for err == nil {
		res, err = protocol.DecodeMessage(c.option.ProtocolType, c.rwc)
		if err != nil {
			break
		}
		seq := res.Seq
		pendingCall, _ := c.pendingCalls.Load(seq)
		call := pendingCall.(*Call)
		c.pendingCalls.Delete(seq)
		if call == nil {
			// nothing to to
		} else if res.Error != "" {
			call.Error = errors.New(res.Error)
			call.done()
		} else {
			err = c.codec.Decode(res.Data, call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
}
