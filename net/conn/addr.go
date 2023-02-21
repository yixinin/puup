package conn

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pion/webrtc/v3"
	"github.com/yixinin/puup/stderr"
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

type ServerAddr struct {
	Label      *Label
	ServerName string
}

func NewServerAddr(serverName string, label *Label) *ServerAddr {
	return &ServerAddr{
		Label:      label,
		ServerName: serverName,
	}
}

func (a *ServerAddr) Network() string {
	return "webrtc"
}

func (a *ServerAddr) String() string {
	return fmt.Sprintf("%s.%d", a.ServerName, a.Label.String())
}

type ClientAddr struct {
	Label    *Label
	ClientId string
}

func NewClientAddr(clientId string, label *Label) *ClientAddr {
	a := &ClientAddr{
		ClientId: clientId,
		Label:    label,
	}
	return a
}

func (a *ClientAddr) Network() string {
	return "webrtc"
}

func (a *ClientAddr) String() string {
	return fmt.Sprintf("%s.%d", a.ClientId, a.Label.String())
}

func parseLabel(label string) (*Label, error) {
	addrs := strings.Split(label, ".")
	if len(addrs) != 2 {
		return nil, stderr.New("invalid label")
	}
	addrs = strings.Split(addrs[1], ":")
	if len(addrs) != 2 {
		return nil, stderr.New("invalid label")
	}
	idx, err := strconv.Atoi(addrs[1])
	if err != nil {
		return nil, stderr.Wrap(err)
	}
	return &Label{
		t:   webrtc.SDPTypeOffer,
		ct:  ChannelType(addrs[0]),
		idx: uint64(idx),
	}, nil
}
