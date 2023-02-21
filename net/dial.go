package net

import (
	"net"

	"github.com/yixinin/puup/net/conn"
)

var peerClient *PeersClient

func init() {
	peerClient = NewPeersClient()
}

func Dial(sigAddr, serverName string, ct conn.ChannelType) (net.Conn, error) {
	return peerClient.Dial(sigAddr, serverName, ct)
}
