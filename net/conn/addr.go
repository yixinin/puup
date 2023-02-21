package conn

import (
	"fmt"
	"strconv"
	"strings"

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
	ChannelType ChannelType
	Index       uint64
}

func NewLabel(ct ChannelType, idx uint64) *Label {
	return &Label{
		ChannelType: ct,
		Index:       idx,
	}
}
func (l *Label) String() string {
	return fmt.Sprintf("%s:%d", l.ChannelType, l.Index)
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
	return fmt.Sprintf("%s.%s", a.ClientId, a.Label.String())
}

func parseLabel(label string) (*Label, error) {
	addrs := strings.Split(label, ":")
	if len(addrs) != 2 {
		return nil, stderr.New("invalid label: " + label)
	}
	idx, err := strconv.Atoi(addrs[1])
	if err != nil {
		return nil, stderr.Wrap(err)
	}
	return &Label{
		ChannelType: ChannelType(addrs[0]),
		Index:       uint64(idx),
	}, nil
}
