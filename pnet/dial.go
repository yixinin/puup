package pnet

import (
	"net"
)

var peerClient *PeersClient

func init() {
	peerClient = NewPeersClient()
}

func Dial(serverAddr, backendName string) (net.Conn, error) {
	return peerClient.Dial(serverAddr, backendName)
}
