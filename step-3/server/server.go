package server

import (
	"github.com/huangw1/rpc-demo/step-3/codec"
	"sync"
	"github.com/huangw1/rpc-demo/step-3/transport"
	"reflect"
	"context"
	"unicode/utf8"
	"unicode"
	"errors"
	"fmt"
	"github.com/huangw1/rpc-demo/step-3/protocol"
	"io"
	"log"
)

type RPCServer interface {
	Register(receive interface{}, metaData map[string]string) error
	Serve(network string, addr string) error
	Close() error
}

type simpleServer struct {
	codec      codec.Codec
	tr         transport.ServerTransport
	serviceMap sync.Map
	mutex      sync.Mutex
	shutdown   bool
	option     Option
}

func NewSimpleServer(option Option) *simpleServer {
	s := new(simpleServer)
	s.option = option
	s.codec = codec.GetCodec(option.SerializeType)
	return s
}

type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
}

type service struct {
	name    string
	typ     reflect.Type
	rcvr    reflect.Value
	methods map[string]*methodType
}

func (s *simpleServer) Register(rcvr interface{}, metaData map[string]string) error {
	typ := reflect.TypeOf(rcvr)
	name := typ.Name()
	service := new(service)
	service.name = name
	service.typ = typ
	service.rcvr = reflect.ValueOf(rcvr)
	methods := suitableMethods(typ)
	service.methods = methods
	if len(service.methods) == 0 {
		var errString string
		methods := suitableMethods(reflect.PtrTo(typ))
		if len(methods) != 0 {
			errString = fmt.Sprintf("rpc-server: service %s has no exported methods(hint: not use pointer)", name)
		} else {
			errString = fmt.Sprintf("rpc-server: service %s has no exported methods", name)
		}
		return errors.New(errString)
	}
	if _, duplicate := s.serviceMap.LoadOrStore(name, service); duplicate {
		return errors.New(fmt.Sprintf("rpc-server: service %s already defined", name))
	}
	return nil
}

/**
	reflect.TypeOf((error)(nil)) -> nil
	reflect.TypeOf((*error)(nil)).Elem() -> error
	PkgPath() == "" -> package empty => buildIn
	Kind() -> type Kind uint
	Type -> build diff from Kind()
 */
var typeOfContext = reflect.TypeOf((*context.Context)(nil)).Elem()
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

func suitableMethods(typ reflect.Type) map[string]*methodType {
	methods := make(map[string]*methodType)
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		mname := method.Name
		mtype := method.Type
		if mtype.PkgPath() != "" {
			continue
		}
		if mtype.NumIn() != 4 {
			continue
		}
		ctxType := mtype.In(1)
		if !ctxType.Implements(typeOfContext) {
			continue
		}
		argType := mtype.In(2)
		if !isExportedOrBuiltinType(argType) {
			continue
		}
		replyType := mtype.In(3)
		if replyType.Kind() != reflect.Ptr {
			continue
		}
		if !isExportedOrBuiltinType(replyType) {
			continue
		}
		if mtype.NumOut() != 1 {
			continue
		}
		returnType := mtype.Out(0)
		if returnType != typeOfError {
			continue
		}
		methods[mname] = &methodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
	}
	return methods
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return isExported(t.Name()) || t.PkgPath() == ""
}

func isExported(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}

func (s *simpleServer) Serve(network string, addr string) error {
	s.tr = transport.NewServerTransport(s.option.TransportType)
	err := s.tr.Listen(network, addr)
	if err != nil {
		return err
	}
	for {
		tr, err := s.tr.Accept()
		if err != nil {
			return err
		}
		go s.serveTransport(tr)
	}
	return err
}

func (s *simpleServer) serveTransport(tr transport.Transport) {
	for {
		req, err := protocol.DecodeMessage(s.option.ProtocolType, tr)
		if err != nil {
			fmt.Println(err)
			if err == io.EOF {
				log.Println("rpc-server: client has closed connection")
			} else {
				log.Println("rpc-server: fail to read request")
			}
			return
		} else {
			res := req.Clone()
			res.MessageType = protocol.MessageTypeRes
			serviceName := res.ServiceName
			methodName := res.MethodName
			serviceVal, ok := s.serviceMap.Load(serviceName)
			if !ok {
				log.Printf("rpc-server: can not find service %s", serviceName)
				return
			}
			service, ok := serviceVal.(*service)
			if !ok {
				log.Println("rpc-server: not *service type")
				return
			}
			method := service.methods[methodName]
			ctx := context.Background()
			arg := newVal(method.ArgType)
			reply := newVal(method.ReplyType)
			err = codec.GetCodec(s.option.SerializeType).Decode(res.Data, arg)
			var returns []reflect.Value
			if method.ArgType.Kind() != reflect.Ptr {
				returns = method.method.Func.Call([]reflect.Value{
					service.rcvr,
					reflect.ValueOf(ctx),
					reflect.ValueOf(arg).Elem(),
					reflect.ValueOf(reply),
				})
			} else {
				returns = method.method.Func.Call([]reflect.Value{
					service.rcvr,
					reflect.ValueOf(ctx),
					reflect.ValueOf(arg),
					reflect.ValueOf(reply),
				})
			}
			if len(returns) > 0 && returns[0].Interface() != nil {
				err := returns[0].Interface().(error)
				s.writeErrorResponse(res, tr, err.Error())
				return
			}
			data, err := codec.GetCodec(s.option.SerializeType).Encode(reply)
			if err != nil {
				s.writeErrorResponse(res, tr, err.Error())
				return
			}
			res.StatusCode = protocol.StatusOk
			res.Data =data
			tr.Write(protocol.EncodeMessage(s.option.ProtocolType, res))
		}
	}
}

func newVal(t reflect.Type) interface{} {
	if t.Kind() == reflect.Ptr {
		return reflect.New(t.Elem()).Interface()
	} else {
		return reflect.New(t).Interface()
	}
}

func (s *simpleServer) writeErrorResponse(res *protocol.Message, w io.Writer, err string) {
	res.Error = err
	res.Data = res.Data[:0]
	res.StatusCode = protocol.StatusError
	w.Write(protocol.EncodeMessage(s.option.ProtocolType, res))
}

func (s *simpleServer) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.shutdown = true
	err := s.tr.Close()
	s.serviceMap.Range(func(key, value interface{}) bool {
		s.serviceMap.Delete(key)
		return true
	})
	return err
}
