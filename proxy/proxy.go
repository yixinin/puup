package proxy

import (
	"context"

	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/stderr"
)

type ProxyPort struct {
	Local  uint16 `yaml:"local"`
	Remote uint16 `yaml:"remote,omitempty"`
}

type Proxy struct {
	Type       conn.PeerType
	sigAddr    string
	serverName string
	ports      map[uint16]uint16
}

func NewProxy(cfg []ProxyPort, pt conn.PeerType) (*Proxy, error) {
	var ports = make(map[uint16]uint16)
	for _, v := range cfg {
		ports[v.Local] = v.Remote
	}
	return &Proxy{
		Type:  pt,
		ports: ports,
	}, nil
}
func (p *Proxy) Run(ctx context.Context) error {
	switch p.Type {
	case conn.Offer:
		return p.runForwards()
	case conn.Answer:
		var ports = make(map[uint16]struct{}, len(p.ports))
		for k := range p.ports {
			ports[k] = struct{}{}
		}
		return p.runBackward(ports)
	}
	return stderr.New("unknown proxy type")
}
