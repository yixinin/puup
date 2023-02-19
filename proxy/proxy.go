package proxy

import (
	"context"
	"os"

	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/stderr"
	"gopkg.in/yaml.v3"
)

type Port struct {
	Local  uint16 `yaml:"local"`
	Remote uint16 `yaml:"remote,omitempty"`
}
type Config struct {
	Ports []Port `yaml:"ports"`
}
type Proxy struct {
	Type       conn.PeerType
	sigAddr    string
	serverName string
	ports      map[uint16]uint16
}

func NewProxy(filename string, pt conn.PeerType) (*Proxy, error) {
	var c = new(Config)
	var data, err = os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, c)
	if err != nil {
		return nil, err
	}
	var ports = make(map[uint16]uint16)
	for _, v := range c.Ports {
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
