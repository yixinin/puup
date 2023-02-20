package proxy

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
	pnet "github.com/yixinin/puup/net"
	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/stderr"
)

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
