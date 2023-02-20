package conn

import (
	"fmt"
)

type ChannelType string

const (
	Data      ChannelType = "data"
	Keepalive ChannelType = "keepalive"
	Cmd       ChannelType = "command"
)

func (t ChannelType) String() string {
	switch t {
	case Data, Cmd, Keepalive:
		return string(t)
	}
	return "unknown"
}

type PeerAddr struct {
	Type   PeerType
	Name   string
	PeerId int
}

func NewPeerAddr(name string, pid int, pt PeerType) *PeerAddr {
	return &PeerAddr{
		Name:   name,
		PeerId: pid,
		Type:   pt,
	}
}

func (a *PeerAddr) Network() string {
	return "webrtc"
}

func (a *PeerAddr) String() string {
	return fmt.Sprintf("%s.%s:%d", a.Type, a.Name, a.PeerId)
}
