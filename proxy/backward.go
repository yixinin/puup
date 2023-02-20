package proxy

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
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

		logrus.Debugf("proxy from %s, read port", rconn.RemoteAddr())
		var header = make([]byte, 2)
		n, err := rconn.Read(header)
		if err != nil {
			return err
		}
		if n != 2 {
			return stderr.New("proxy header error")
		}
		var port uint16
		binary.BigEndian.PutUint16(header, port)
		logrus.Debugf("proxy %s, on port: %d, start to copy data", rconn.RemoteAddr(), port)
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
