package proxy

import (
	"context"
	"fmt"
	"net"

	pnet "github.com/yixinin/puup/net"
	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/stderr"
)

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

		raddr, ok := rconn.RemoteAddr().(*conn.LabelAddr)
		if !ok {
			return stderr.New("unknown addr")
		}

		port := raddr.ProxyPort()
		lconn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
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
