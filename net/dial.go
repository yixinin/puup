package net

import (
	"net"
)

var peerClient *PeersClient

func init() {
	peerClient = NewPeersClient()
}

func Dial(sigAddr, serverName string) (net.Conn, error) {
	return peerClient.Dial(sigAddr, serverName)
}
