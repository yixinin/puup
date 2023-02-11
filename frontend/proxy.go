package frontend

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/pnet"
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
	Type  pnet.PeerType
	ports map[uint16]uint16
}

func NewProxy(filename string, pt pnet.PeerType) (*Proxy, error) {
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

func (p *Proxy) runBackwords() {
	var wg sync.WaitGroup
	for local, remote := range p.ports {
		wg.Add(1)
		go func(l, r uint16) {
			defer wg.Done()
			if err := p.runForward(l, r); err != nil {
				logrus.Errorf("run forword :%d->%d error:%v", l, r, err)
			}
		}(local, remote)
	}
	wg.Wait()
}
func (p *Proxy) runForward(localPort, remotePort uint16) error {
	cli := pnet.NewPeersClient()
	err := cli.Connect(p.cfg.ServerAddr, p.cfg.BackendName)
	if err != nil {
		return stderr.Wrap(err)
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", localPort))
	if err != nil {
		return stderr.Wrap(err)
	}
	for {
		lconn, err := lis.Accept()
		if err != nil {
			return stderr.Wrap(err)
		}
		rconn, err := cli.DialProxy(p.cfg.ServerAddr, p.cfg.BackendName, remotePort)
		if err != nil {
			return stderr.Wrap(err)
		}
		go func(src, dst net.Conn) {
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				io.Copy(src, dst)
				wg.Done()
			}()
			go func() {
				io.Copy(dst, src)
				wg.Done()
			}()
			wg.Wait()
		}(lconn, rconn)
	}
}
