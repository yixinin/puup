package frontend

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/config"
	pnet "github.com/yixinin/puup/net"
	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/stderr"
)

type ProxyClient struct {
	Type       webrtc.SDPType
	sigAddr    string
	serverName string
	ports      map[uint16]uint16
}

func NewProxy(cfg *config.Config, pt webrtc.SDPType) (*ProxyClient, error) {
	var ports = make(map[uint16]uint16)
	for _, v := range cfg.ProxyFront {
		ports[v.Local] = v.Remote
	}
	return &ProxyClient{
		Type:       pt,
		sigAddr:    cfg.SigAddr,
		serverName: cfg.ServerName,
		ports:      ports,
	}, nil
}
func (p *ProxyClient) Run(ctx context.Context) error {
	return p.runForwards()
}

func (p *ProxyClient) runForwards() error {
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
func (p *ProxyClient) runForward(localPort, remotePort uint16) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", localPort))
	if err != nil {
		return stderr.Wrap(err)
	}
	for {
		lconn, err := lis.Accept()
		if err != nil {
			return stderr.Wrap(err)
		}
		rconn, err := pnet.Dial(p.sigAddr, p.serverName, conn.Proxy)
		if err != nil {
			return stderr.Wrap(err)
		}
		var header = make([]byte, 2)
		binary.BigEndian.PutUint16(header, remotePort)
		logrus.Debugf("proxy %s, on port: %d, write header:%v", rconn.RemoteAddr(), remotePort, header)
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
