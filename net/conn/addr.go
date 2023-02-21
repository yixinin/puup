package conn

import (
	"fmt"

	"github.com/pion/webrtc/v3"
)

type ChannelType string

const (
	Keepalive ChannelType = "keepalive"
	Cmd       ChannelType = "command"
	Web       ChannelType = "web"
	Proxy     ChannelType = "proxy"
	Ssh       ChannelType = "ssh"
	File      ChannelType = "file"
)

func (t ChannelType) String() string {
	switch t {
	case Cmd, Keepalive, Web, Proxy, Ssh, File:
		return string(t)
	}
	return "unknown"
}

type Label struct {
	t   webrtc.SDPType
	ct  ChannelType
	idx uint64
}

func NewLabel(t webrtc.SDPType, ct ChannelType, idx uint64) *Label {
	return &Label{
		t:   t,
		ct:  ct,
		idx: idx,
	}
}
func (l *Label) String() string {
	return fmt.Sprintf("%s.%s:%d", l.t, l.ct, l.idx)
}

type OfferAddr struct {
	Label      *Label
	ServerName string
}

func NewOfferAddr(serverName string, label *Label) *OfferAddr {
	return &OfferAddr{
		Label:      label,
		ServerName: serverName,
	}
}

func (a *OfferAddr) Network() string {
	return "webrtc"
}

func (a *OfferAddr) String() string {
	return fmt.Sprintf("%s.%d", a.ServerName, a.Label.String())
}

func NewAnswerAddr(serverName, label string) *PeerAddr {
	return &PeerAddr{}
}

func parseLabel() {

}
