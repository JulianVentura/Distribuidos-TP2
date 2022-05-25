package socket

import (
	Err "distribuidos/tp2/common/errors"
	"net"
)

func NewClient(address string) (*TCPConnection, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, Err.Ctx("Error on connect syscall of socket.NewClient", err)
	}

	return &TCPConnection{
		skt: conn,
	}, nil
}
