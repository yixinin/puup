package backend

import "net"

type Server interface {
	Serve(conn net.Conn) error
}
