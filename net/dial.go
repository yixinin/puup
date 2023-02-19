package net

import (
	"net"
)

var peerClient *PeersClient

func init() {
	peerClient = NewPeersClient()
}

func DialWeb(serverAddr, sigAddr string) (net.Conn, error) {
	return peerClient.DialWeb(serverAddr, sigAddr)
}
func DialSsh(serverAddr, sigAddr string) (net.Conn, error) {
	return peerClient.DialSsh(serverAddr, sigAddr)
}

func DialFile(serverAddr, sigAddr string) (net.Conn, error) {
	return peerClient.DialFile(serverAddr, sigAddr)
}

func DialProxy(serverAddr, sigAddr string, port uint16) (net.Conn, error) {
	return peerClient.DialProxy(serverAddr, sigAddr, port)
}
