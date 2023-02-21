package conn

import (
	"context"
	"errors"
	"io"
	"net"
	"runtime/debug"

	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/stderr"
)

func GoFunc(ctx context.Context, f func(ctx context.Context) error) {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithField("stacks", string(debug.Stack())).Errorf("run go func panic, r:%v", r)
		}
	}()
	go func() {
		err := f(ctx)
		if err != nil {
			logrus.Errorf("run go func error:%v", err)
		}
	}()
}

func GoCopy(src, dst net.Conn) error {
	defer func() {
		src.Close()
		dst.Close()
	}()

	var ch = make(chan error, 1)
	GoFunc(context.TODO(), func(ctx context.Context) error {
		_, err := io.Copy(dst, src)
		if err != nil && !errors.Is(err, io.EOF) {
			ch <- err
		}
		return stderr.Wrap(err)
	})
	GoFunc(context.TODO(), func(ctx context.Context) error {
		_, err := io.Copy(src, dst)
		if err != nil && !errors.Is(err, io.EOF) {
			ch <- err
		}
		return stderr.Wrap(err)
	})
	err := <-ch
	return err
}
