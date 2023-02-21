package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/config"
	pnet "github.com/yixinin/puup/net"
	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/stderr"
)

type ProxyServer struct {
	lis       *pnet.Listener
	localAddr string
	ports     map[uint16]struct{}
}

func NewProxy(cfg *config.Config, lis *pnet.Listener) *ProxyServer {
	var ports = make(map[uint16]struct{})
	for _, v := range cfg.ProxyBack.Ports {
		ports[v] = struct{}{}
	}

	return &ProxyServer{
		lis:       lis,
		localAddr: cfg.ProxyBack.Addr,
		ports:     ports,
	}
}
func (p *ProxyServer) Run(ctx context.Context) error {
	for {
		rconn, err := p.lis.AcceptProxy()
		if err != nil {
			return stderr.Wrap(err)
		}
		if err := p.ServeConn(ctx, rconn); err != nil {
			return err
		}
	}
}

type ProxyHeader struct {
	Port uint16 `json:"port"`
}

func (p *ProxyServer) ServeConn(ctx context.Context, rconn net.Conn) error {
	logrus.Debugf("proxy from %s, read port", rconn.RemoteAddr())
	var header = make([]byte, 2)
	n, err := rconn.Read(header)
	if err != nil {
		return err
	}
	if n != 2 {
		return stderr.New("proxy header error")
	}
	port := binary.BigEndian.Uint16(header)
	if port == 0 {
		return stderr.New("proxy port error")
	}
	logrus.Debugf("proxy %s header:%v, on port: %d, start to copy data", rconn.RemoteAddr(), header, port)
	lconn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", p.localAddr, port))
	if err != nil {
		return stderr.Wrap(err)
	}

	conn.GoFunc(context.TODO(), func(ctx context.Context) error {
		defer func() {
			rconn.(*pnet.Conn).Release()
		}()
		return conn.GoCopy(lconn, rconn)
	})
	return nil
}
