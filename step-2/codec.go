package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"io"
	"errors"
	"reflect"
	"net/rpc"
	"net"
	"github.com/vmihailenco/msgpack"
)

const HeadSize = 4

type MsgPackReq struct {
	rpc.Request
	Arg interface{}
}

type MsgPackRes struct {
	rpc.Response
	Reply interface{}
}

/**
type ServerCodec interface {
	ReadRequestHeader(*Request) error
	ReadRequestBody(interface{}) error
	WriteResponse(*Response, interface{}) error
	Close() error
}
 */

type MessagePackServerCodec struct {
	rwc    io.ReadWriteCloser
	req    MsgPackReq
	closed bool
}

func NewServerCodec(conn net.Conn) *MessagePackServerCodec {
	return &MessagePackServerCodec{
		rwc:    conn,
		req:    MsgPackReq{},
		closed: false,
	}
}

func (s *MessagePackServerCodec) WriteResponse(r *rpc.Response, reply interface{}) error {
	if s.closed {
		return nil
	}
	res := &MsgPackRes{*r, reply}
	data, err := msgpack.Marshal(res)
	if err != nil {
		panic(err)
	}
	head := make([]byte, HeadSize)
	binary.BigEndian.PutUint32(head, uint32(len(data)))
	_, err = s.rwc.Write(head)
	if err != nil {
		panic(err)
	}
	_, err = s.rwc.Write(data)
	if err != nil {
		panic(err)
	}
	return err
}

func (s *MessagePackServerCodec) ReadRequestHeader(r *rpc.Request) error {
	if s.closed {
		return nil
	}
	data, err := readData(s.rwc)
	if err != nil {
		if err == io.EOF {
			return err
		}
		panic(err)
	}
	var req MsgPackReq
	err = msgpack.Unmarshal(data, &req)
	if err != nil {
		panic(err)
	}
	// 设置 Request 各个属性
	r.ServiceMethod = req.ServiceMethod
	r.Seq = req.Seq
	s.req = req
	return nil
}

func (s *MessagePackServerCodec) ReadRequestBody(arg interface{}) error {
	if arg != nil {
		reflect.ValueOf(arg).Elem().Set(reflect.ValueOf(s.req.Arg))
	}
	return nil
}

func (s *MessagePackServerCodec) Close() error {
	s.closed = true
	if s.rwc != nil {
		s.rwc.Close()
	}
	return nil
}

/**
type ClientCodec interface {
	WriteRequest(*Request, interface{}) error
	ReadResponseHeader(*Response) error
	ReadResponseBody(interface{}) error
	Close() error
}
 */

type MessagePackClientCodec struct {
	rwc    io.ReadWriteCloser
	res    MsgPackRes
	closed bool
}

func NewClientCodec(conn net.Conn) *MessagePackClientCodec {
	return &MessagePackClientCodec{
		rwc:    conn,
		res:    MsgPackRes{},
		closed: false,
	}
}

func (c *MessagePackClientCodec) WriteRequest(r *rpc.Request, arg interface{}) error {
	if c.closed {
		return nil
	}
	req := &MsgPackReq{*r, arg}
	data, err := msgpack.Marshal(req)
	if err != nil {
		panic(err)
	}
	head := make([]byte, HeadSize)
	binary.BigEndian.PutUint32(head, uint32(len(data)))
	_, err = c.rwc.Write(head)
	if err != nil {
		panic(err)
	}
	_, err = c.rwc.Write(data)
	if err != nil {
		panic(err)
	}
	return err
}

func (c *MessagePackClientCodec) ReadResponseHeader(r *rpc.Response) error {
	if c.closed {
		return nil
	}
	data, err := readData(c.rwc)
	if err != nil {
		panic(err)
	}
	var res MsgPackRes
	err = msgpack.Unmarshal(data, &res)
	if err != nil {
		panic(err)
	}
	// 设置 Request 各个属性
	r.ServiceMethod = res.ServiceMethod
	r.Seq = res.Seq
	c.res = res
	return nil
}

func (c *MessagePackClientCodec) ReadResponseBody(reply interface{}) error {
	if c.res.Error != "" {
		return errors.New(c.res.Error)
	}
	if reply != nil {
		reflect.ValueOf(reply).Elem().Set(reflect.ValueOf(c.res.Reply))
	}
	return nil
}


func (c *MessagePackClientCodec) Close() error {
	c.closed = true
	if c.rwc != nil {
		c.rwc.Close()
	}
	return nil
}

func readData(conn io.ReadWriteCloser) ([]byte, error) {
	headBuf := bytes.NewBuffer(make([]byte, 0, HeadSize))
	headData := make([]byte, HeadSize)
	headSize := HeadSize
	for {
		num, err := conn.Read(headData)
		if err != nil && err != io.EOF {
			log.Fatal(err)
			return nil, err
		}
		headBuf.Write(headData[0:num])
		if headBuf.Len() == HeadSize {
			break
		} else {
			headData = make([]byte, headSize-num)
			headSize -= num
		}
	}

	bodyLen := int(binary.BigEndian.Uint32(headBuf.Bytes()))
	bodySize := bodyLen
	bodyBuf := bytes.NewBuffer(make([]byte, 0, bodyLen))
	bodyData := make([]byte, bodyLen)
	for {
		num, err := conn.Read(bodyData)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		bodyBuf.Write(bodyData[:num])
		if bodyBuf.Len() == bodyLen {
			log.Printf("receive body: %v", bodyBuf.Bytes())
			break
		} else {
			bodyData = make([]byte, bodySize-num)
			bodySize -= num
		}
	}
	return bodyBuf.Bytes(), nil
}