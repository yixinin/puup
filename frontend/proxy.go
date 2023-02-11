package frontend

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/yixinin/puup/ice"
	"github.com/yixinin/puup/pnet"
)

type Proxy struct {
	puup, name string
	port       uint16
}

func NewProxy(puup, name string, port uint16) *Proxy {
	return &Proxy{
		puup: puup,
		name: name,
		port: port,
	}
}

func (p *Proxy) Run() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", p.port))
	if err != nil {
		return err
	}
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return err
	}
	var sigCli = pnet.NewOfferClient(p.puup, p.puup)
	peer, err := pnet.NewOfferPeer(pc, sigCli)
	if err != nil {
		return err
	}
	if err := peer.Connect(); err != nil {
		return err
	}
	for {
		conn, err := lis.Accept()
		if err != nil {
			return err
		}
		dst, err := peer.GetWebConn("")
		if err != nil {
			return err
		}
		go func(src, dst net.Conn) {
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				io.Copy(conn, dst)
				wg.Done()
			}()
			go func() {
				io.Copy(dst, conn)
				wg.Done()
			}()
			wg.Wait()
		}(conn, dst)

	}
}
