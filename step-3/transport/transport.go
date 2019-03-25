package transport

import (
	"io"
	"net"
)

type TransportType byte

const (
	TCPTransport = iota
)

var transports = map[TransportType]Transport{
	TCPTransport: &Socket{},
}

type Transport interface {
	Dial(network, addr string) error
	io.ReadWriteCloser
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

type Socket struct {
	conn net.Conn
}

func NewTransport(t TransportType) Transport {
	return transports[t]
}

func (s *Socket) Dial(network, addr string) error {
	conn, err := net.Dial(network, addr)
	s.conn = conn
	return err
}

func (s *Socket) Read(bytes []byte) (int, error) {
	return s.conn.Read(bytes)
}

func (s *Socket) Write(bytes []byte) (int, error) {
	return s.conn.Write(bytes)
}

func (s *Socket) Close() error {
	return s.conn.Close()
}

func (s *Socket) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

func (s *Socket) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

var serverTransports = map[TransportType]ServerTransport{
	TCPTransport: &ServerSocket{},
}

type ServerTransport interface {
	Listen(network, addr string) error
	Accept() (Transport, error)
	io.Closer
}

type ServerSocket struct {
	ln net.Listener
}

func NewServerTransport(t TransportType) ServerTransport {
	return serverTransports[t]
}

func (s *ServerSocket) Listen(network, addr string) error {
	ln, err := net.Listen(network, addr)
	s.ln = ln
	return err
}

func (s *ServerSocket) Accept() (Transport, error) {
	conn, err := s.ln.Accept()
	return &Socket{conn: conn}, err
}

func (s *ServerSocket) Close() error {
	return s.ln.Close()
}
