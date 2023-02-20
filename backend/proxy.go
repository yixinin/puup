package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/config"
	pnet "github.com/yixinin/puup/net"
	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/stderr"
)

type Proxy struct {
	localAddr  string
	sigAddr    string
	serverName string
	ports      map[uint16]struct{}
}

func NewProxy(cfg *config.Config, pt conn.PeerType) (*Proxy, error) {
	var ports = make(map[uint16]struct{})
	for _, v := range cfg.ProxyBack.Ports {
		ports[v] = struct{}{}
	}

	return &Proxy{
		sigAddr:    cfg.SigAddr,
		serverName: fmt.Sprintf("%s.proxy", cfg.ServerName),
		ports:      ports,
	}, nil
}
func (p *Proxy) Run(ctx context.Context) error {
	return p.runBackward(p.ports)
}

type ProxyHeader struct {
	Port uint16 `json:"port"`
}

func (p *Proxy) runBackward(ports map[uint16]struct{}) error {
	lis := pnet.NewListener(p.sigAddr, p.serverName)
	for {
		rconn, err := lis.Accept()
		if err != nil {
			return stderr.Wrap(err)
		}

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
		logrus.Debugf("proxy %s, on port: %d, start to copy data", rconn.RemoteAddr(), port)
		lconn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", p.localAddr, port))
		if err != nil {
			return stderr.Wrap(err)
		}

		conn.GoFunc(context.TODO(), func(ctx context.Context) error {
			defer func() {
				rconn.(*pnet.Conn).Release()
			}()
			return Copy(lconn, rconn)
		})
	}
}
func Copy(src, dst net.Conn) error {
	defer func() {
		src.Close()
		dst.Close()
	}()

	conn.GoFunc(context.TODO(), func(ctx context.Context) error {
		_, err := io.Copy(dst, src)
		return stderr.Wrap(err)
	})
	conn.GoFunc(context.TODO(), func(ctx context.Context) error {
		_, err := io.Copy(src, dst)
		return stderr.Wrap(err)
	})

	return nil
}
