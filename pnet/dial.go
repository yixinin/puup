package pnet

import (
	"net"

	"github.com/yixinin/puup/connection"
)

var peerClient *connection.PeersClient

func init() {
	peerClient = connection.NewPeersClient()
}

func Dial(serverAddr, backendName string) (net.Conn, error) {
	return peerClient.Dial(serverAddr, backendName)
}
