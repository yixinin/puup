package proxy

import (
	"context"
	"io"
	"net"

	"github.com/yixinin/puup/net/conn"
)

func Copy(src, dst net.Conn) error {
	defer func() {
		src.Close()
		dst.Close()
	}()

	conn.GoFunc(context.TODO(), func(ctx context.Context) error {
		_, err := io.Copy(dst, src)
		return err
	})
	conn.GoFunc(context.TODO(), func(ctx context.Context) error {
		_, err := io.Copy(src, dst)
		return err
	})

	return nil
}
