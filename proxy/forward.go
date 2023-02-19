package proxy

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/sirupsen/logrus"
	pnet "github.com/yixinin/puup/net"
	"github.com/yixinin/puup/stderr"
)

func (p *Proxy) runForwards() error {
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
	return nil
}
func (p *Proxy) runForward(localPort, remotePort uint16) error {
	cli := pnet.NewPeersClient()
	err := cli.Connect(p.sigAddr, p.serverName)
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
		rconn, err := cli.DialProxy(p.sigAddr, p.serverName, remotePort)
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
