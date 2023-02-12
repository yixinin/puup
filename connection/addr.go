package connection

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/yixinin/puup/stderr"
)

type ChannelType string

const (
	Proxy     ChannelType = "proxy"
	Web       ChannelType = "web"
	File      ChannelType = "file"
	Ssh       ChannelType = "ssh"
	Keepalive ChannelType = "keepalive"
)

func (t ChannelType) String() string {
	switch t {
	case Proxy, Web, File, Ssh, Keepalive:
		return string(t)
	}
	return "unknown"
}

type LabelAddr struct {
	Name string
	Type ChannelType
	id   uint64
}

func NewWebAddr(name string, idx uint64) net.Addr {
	return &LabelAddr{
		Name: name,
		Type: Web,
		id:   idx,
	}
}
func NewFileAddr(name string, idx uint64) net.Addr {
	return &LabelAddr{
		Name: name,
		Type: File,
		id:   idx,
	}
}

func NewSshAddr(name string) net.Addr {
	return &LabelAddr{
		Name: name,
		Type: Ssh,
	}
}

func NewProxyAddr(name string, port uint16) net.Addr {
	return &LabelAddr{
		Name: name,
		Type: Web,
		id:   uint64(port),
	}
}

func (a *LabelAddr) Network() string {
	return "webrtc"
}

func (a *LabelAddr) String() string {
	return fmt.Sprintf("%s:%s", a.Name, a.Label)
}

func (a *LabelAddr) Label() string {
	switch a.Type {
	case Ssh:
		return Ssh.String()
	case Web, File:
		return fmt.Sprintf("%s.%d", a.Type, a.id)
	case Proxy:
		return fmt.Sprintf("%s.%d", a.Type, a.id)
	case Keepalive:
		return Keepalive.String()
	}
	return ""
}

func (a *LabelAddr) ProxyPort() uint16 {
	if a.Type == Proxy {
		return uint16(a.id)
	}
	return 0
}

func AddrFromLabel(backendName, id, label string) (*LabelAddr, *LabelAddr, error) {
	laddr := new(LabelAddr)
	raddr := new(LabelAddr)
	addrs := strings.Split(label, ".")
	t := ChannelType(addrs[0])

	laddr.Name = backendName
	laddr.Type = t
	raddr.Name = id
	raddr.Type = t

	var idx uint64
	if len(addrs) >= 2 {
		var err error
		idx, err = strconv.ParseUint(addrs[1], 10, 64)
		if err != nil {
			return nil, nil, stderr.Wrap(err)
		}
	}

	switch t {
	case Ssh, Keepalive:
		return laddr, raddr, nil
	case Web, File, Proxy:
		laddr.id = idx
		raddr.id = idx
	}
	return nil, nil, stderr.New("unknown label")
}
