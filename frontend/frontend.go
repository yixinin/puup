package frontend

import (
	"context"
	"sync"

	"github.com/yixinin/puup/config"
	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/proxy"
)

type FrontEnd struct {
	proxy *proxy.Proxy
	file  *FileClient
}

func NewFrontEnd(filename string) (*FrontEnd, error) {
	cfg, err := config.LoadConfig(filename)
	if err != nil {
		return nil, err
	}
	f := &FrontEnd{}

	proxy, err := proxy.NewProxy(cfg.Proxy, conn.Answer)
	if err != nil {
		return nil, err
	}
	f.proxy = proxy
	f.file = NewFileClient(cfg)
	return f, nil
}

func (f *FrontEnd) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	wg.Add(2)
	conn.GoFunc(ctx, func(ctx context.Context) error {
		defer wg.Done()
		return f.file.Run(ctx)
	})
	conn.GoFunc(ctx, func(ctx context.Context) error {
		defer wg.Done()
		return f.proxy.Run(ctx)
	})
	wg.Wait()
	return nil
}
