package backend

import (
	"context"
	"net"
	"strings"

	"github.com/pion/webrtc/v3"
	"github.com/yixinin/puup/pnet"
	"github.com/yixinin/puup/proto"
)

type Backend struct {
	lis *pnet.Listener

	ssh   *SshServer
	file  *FileServer
	web   *WebServer
	proxy *Proxy

	sshConn   chan net.Conn
	fileConn  chan net.Conn
	webConn   chan net.Conn
	proxyConn chan net.Conn

	video *webrtc.PeerConnection
}

func NewBackend(ssh, file, web, proxy bool) *Backend {
	b := &Backend{}

	return b
}

func (b *Backend) Run(ctx context.Context) error {
	for {
		conn, err := b.lis.Accept()
		if err != nil {
			return err
		}

		addrs := strings.Split(conn.RemoteAddr().String(), ":")
		name := addrs[0]
		switch proto.GetChannelType(addrs[1]) {
		case proto.Ssh:
		case proto.Proxy:
		case proto.File:
		case proto.Web:
		}

	}
}

func (b *Backend) loop() {

}
