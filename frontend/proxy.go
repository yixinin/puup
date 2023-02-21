package frontend

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/config"
	pnet "github.com/yixinin/puup/net"
	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/stderr"
)

type Proxy struct {
	Type       conn.PeerType
	sigAddr    string
	serverName string
	ports      map[uint16]uint16
}

func NewProxy(cfg *config.Config, pt conn.PeerType) (*Proxy, error) {
	var ports = make(map[uint16]uint16)
	for _, v := range cfg.ProxyFront {
		ports[v.Local] = v.Remote
	}
	return &Proxy{
		Type:       pt,
		sigAddr:    cfg.SigAddr,
		serverName: fmt.Sprintf("%s.proxy", cfg.ServerName),
		ports:      ports,
	}, nil
}
func (p *Proxy) Run(ctx context.Context) error {
	return p.runForwards()
}

func (p *Proxy) runForwards() error {
	var wg sync.WaitGroup
	for local, remote := range p.ports {
		wg.Add(1)
		l := local
		r := remote
		conn.GoFunc(context.TODO(), func(ctx context.Context) error {
			defer wg.Done()
			return p.runForward(l, r)
		})
	}
	wg.Wait()
	return nil
}
func (p *Proxy) runForward(localPort, remotePort uint16) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", localPort))
	if err != nil {
		return stderr.Wrap(err)
	}
	for {
		lconn, err := lis.Accept()
		if err != nil {
			return stderr.Wrap(err)
		}
		rconn, err := pnet.Dial(p.sigAddr, p.serverName)
		if err != nil {
			return stderr.Wrap(err)
		}
		logrus.Debugf("proxy %s, on port: %d, write header", rconn.RemoteAddr(), remotePort)
		var header = make([]byte, 2)
		binary.BigEndian.PutUint16(header, remotePort)
		_, err = rconn.Write(header)
		if err != nil {
			return err
		}
		logrus.Debugf("proxy %s, on port: %d, start to copy data", rconn.RemoteAddr(), remotePort)
		conn.GoFunc(context.TODO(), func(ctx context.Context) error {
			defer func() {
				rconn.(*pnet.Conn).Release()
			}()
			return conn.GoCopy(lconn, rconn)
		})
	}
}
