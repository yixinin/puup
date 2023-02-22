package conn

import (
	"context"
	"errors"
	"io"
	"runtime/debug"

	"github.com/sirupsen/logrus"
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

func GoCopy(src, dst io.ReadWriteCloser) error {
	defer func() {
		src.Close()
		dst.Close()
	}()

	var ch = make(chan error, 1)
	GoFunc(context.TODO(), func(ctx context.Context) error {
		_, err := io.Copy(dst, src)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			ch <- err
		}
		return nil
	})
	GoFunc(context.TODO(), func(ctx context.Context) error {
		_, err := io.Copy(src, dst)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			ch <- err
		}
		return nil
	})
	err := <-ch
	return err
}
