package backend

import (
	"context"
	"net"
	"sync"

	// This is required to use H264 video encoder
	_ "github.com/pion/mediadevices/pkg/driver/camera" // This is required to register camera adapter
	"github.com/yixinin/puup/config"
	pnet "github.com/yixinin/puup/net"
	"github.com/yixinin/puup/net/conn"
)

type Server interface {
	ServeConn(ctx context.Context, conn net.Conn) error
	Match(ctx context.Context, addr net.Addr) bool
}

type Backend struct {
	web   *WebServer
	lis   *pnet.Listener
	ssh   *SshServer
	file  *FileServer
	proxy *Proxy

	close chan struct{}
}

func NewBackend(filename string) (*Backend, error) {
	cfg, err := config.LoadConfig(filename)
	if err != nil {
		return nil, err
	}
	b := &Backend{}

	proxy, err := NewProxy(cfg, conn.Answer)
	if err != nil {
		return nil, err
	}
	b.proxy = proxy

	b.web = NewWebServer(cfg)
	b.file = NewFileServer(cfg)
	b.ssh = NewSshServer(cfg)

	return b, nil
}

func (b *Backend) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	wg.Add(4)
	conn.GoFunc(ctx, func(ctx context.Context) error {
		defer wg.Done()
		return b.web.Run(ctx)
	})
	conn.GoFunc(ctx, func(ctx context.Context) error {
		defer wg.Done()
		return b.file.Run(ctx)
	})
	conn.GoFunc(ctx, func(ctx context.Context) error {
		defer wg.Done()
		return b.ssh.Run(ctx)
	})
	conn.GoFunc(ctx, func(ctx context.Context) error {
		defer wg.Done()
		return b.proxy.Run(ctx)
	})
	wg.Wait()
	return nil
}

func (b *Backend) Close() error {
	select {
	case <-b.close:
		return nil
	default:
	}
	close(b.close)
	return nil
}
