package conn

import (
	"fmt"
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

func NewLocalAddr(label string) {

}

func NewWebAddr(name string, idx uint64) *LabelAddr {
	return &LabelAddr{
		Name: name,
		Type: Web,
		id:   idx,
	}
}
func NewFileAddr(name string, idx uint64) *LabelAddr {
	return &LabelAddr{
		Name: name,
		Type: File,
		id:   idx,
	}
}

func NewSshAddr(name string) *LabelAddr {
	return &LabelAddr{
		Name: name,
		Type: Ssh,
	}
}

func NewProxyAddr(name string, port uint16) *LabelAddr {
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
	return fmt.Sprintf("%s:%s", a.Name, a.Label())
}

func (a *LabelAddr) Label() string {
	switch a.Type {
	case Ssh:
		return Ssh.String()
	case Web, File:
		return fmt.Sprintf("%sx%d", a.Type, a.id)
	case Proxy:
		return fmt.Sprintf("%sx%d", a.Type, a.id)
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

func AddrFromLabel(sigAddr, clientId, label string) (*LabelAddr, *LabelAddr, error) {
	laddr := new(LabelAddr)
	raddr := new(LabelAddr)
	addrs := strings.Split(label, "x")
	ctype := ChannelType(addrs[0])

	laddr.Name = sigAddr
	laddr.Type = ctype
	raddr.Name = clientId
	raddr.Type = ctype

	var idx uint64
	if len(addrs) >= 2 {
		var err error
		idx, err = strconv.ParseUint(addrs[1], 10, 64)
		if err != nil {
			return nil, nil, stderr.Wrap(err)
		}
	}

	switch ctype {
	case Ssh, Keepalive:
		return laddr, raddr, nil
	case Web, File, Proxy:
		laddr.id = idx
		raddr.id = idx
	}
	return nil, nil, stderr.New("unknown label")
}
