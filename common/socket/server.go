package socket

import (
	Err "distribuidos/tp2/common/errors"
	"net"
)

type ServerSocket struct {
	skt net.Listener
}

func NewServer(address string) (ServerSocket, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return ServerSocket{}, Err.Ctx("Error on listen syscall of socket.NewServer", err)
	}

	return ServerSocket{
		skt: listener,
	}, nil
}

func (self *ServerSocket) Accept() (TCPConnection, error) {
	conn, err := self.skt.Accept()

	if err != nil {
		return TCPConnection{}, Err.Ctx("Error on accept syscall of socket.Accept", err)
	}

	return newTCPConnection(conn), nil
}

func (self *ServerSocket) Close() error {
	err := self.skt.Close()

	if err != nil {
		return Err.Ctx("Error on close syscall of socket.Close", err)
	}

	return nil
}
