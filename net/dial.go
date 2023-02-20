package net

import (
	"net"
)

var peerClient *PeersClient

func init() {
	peerClient = NewPeersClient()
}

func DialWeb(sigAddr, serverName string) (net.Conn, error) {
	return peerClient.DialWeb(sigAddr, serverName)
}
func DialSsh(sigAddr, serverName string) (net.Conn, error) {
	return peerClient.DialSsh(sigAddr, serverName)
}

func DialFile(sigAddr, serverName string) (net.Conn, error) {
	return peerClient.DialFile(sigAddr, serverName)
}

func DialProxy(sigAddr, serverName string, port uint16) (net.Conn, error) {
	return peerClient.DialProxy(sigAddr, serverName, port)
}
