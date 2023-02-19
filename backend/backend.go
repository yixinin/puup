package backend

import (
	"context"
	"net"

	// This is required to use H264 video encoder
	_ "github.com/pion/mediadevices/pkg/driver/camera" // This is required to register camera adapter
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/net"
)

type Backend struct {
	lis *net.Listener

	rtc   *RtcServer
	web   *WebServer
	ssh   Server
	file  Server
	proxy Server

	sshConn   chan net.Conn
	fileConn  chan net.Conn
	webConn   chan net.Conn
	proxyConn chan net.Conn

	close chan struct{}
}

func NewBackend(sigAddr, sigAddr string) (*Backend, error) {
	b := &Backend{
		lis:       net.NewListener(sigAddr, sigAddr),
		sshConn:   make(chan net.Conn, 1),
		fileConn:  make(chan net.Conn, 1),
		proxyConn: make(chan net.Conn, 1),
		webConn:   make(chan net.Conn, 1),
	}

	var proxy, err = NewProxy()
	if err != nil {
		return nil, err
	}
	b.proxy = proxy

	b.web = NewWebServer(b)
	b.file = NewFileServer()
	b.ssh = NewSshServer()

	return b, nil
}
func (b *Backend) Accept() (net.Conn, error) {
	conn, ok := <-b.webConn
	if ok {
		return conn, nil
	}
	return nil, net.ErrClosed
}

func (b *Backend) Listen() error {
	for {
		conn, err := b.lis.Accept()
		if err != nil {
			return err
		}
		addr := conn.RemoteAddr().(*net.LabelAddr)
		switch addr.Type {
		case net.Ssh:
			b.sshConn <- conn
		case net.File:
			b.fileConn <- conn
		case net.Web:
			b.webConn <- conn
		case net.Proxy:
			b.proxyConn <- conn
		default:
			logrus.Errorln("unkown conn")
		}
	}
}

func (b *Backend) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case conn := <-b.fileConn:
			go func() {
				b.file.Serve(conn)
			}()
		case conn := <-b.sshConn:
			go func() {
				b.ssh.Serve(conn)
			}()
		case conn := <-b.proxyConn:
			go func() {
				b.proxy.Serve(conn)
			}()
		}
	}
}

func (b *Backend) Addr() net.Addr {
	return b.lis.Addr()
}

func (b *Backend) Close() error {
	return nil
}
