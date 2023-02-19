package conn

import (
	"context"
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
