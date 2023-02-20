package net

import (
	"bufio"
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/net/conn"
)

type Conn struct {
	sync.RWMutex

	conn.ReadWriterReleaser

	isRelease bool
	close     chan struct{}
	buffer    *bufio.Reader
	rdl       time.Time
	sendPool  chan []byte
}

func NewConn(rwr conn.ReadWriterReleaser) *Conn {
	c := &Conn{
		ReadWriterReleaser: rwr,
		close:              make(chan struct{}),
		sendPool:           make(chan []byte),
	}
	conn.GoFunc(context.TODO(), func(ctx context.Context) error {
		return c.loopSend(ctx)
	})
	c.buffer = bufio.NewReader(rwr)
	return c
}

func (s *Conn) IsClose() bool {
	s.RLock()
	defer s.RUnlock()

	select {
	case <-s.close:
		return true
	default:
	}
	return false
}

func (c *Conn) SetDeadline(t time.Time) error {
	c.rdl = t
	return nil
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	c.rdl = t
	return nil
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (c *Conn) getRdl() (*time.Timer, error) {
	var timeout time.Duration
	zero := c.rdl.IsZero()
	if !zero {
		timeout = time.Until(c.rdl)
	}
	c.rdl = time.Time{}

	if timeout <= 0 && !zero {
		return nil, context.DeadlineExceeded
	}
	if timeout <= 100*time.Millisecond {
		timeout = 100 * time.Millisecond
	}
	tm := time.NewTimer(timeout)
	defer tm.Stop()

	if zero {
		tm.Stop()
	}
	return tm, nil
}

type DataInfo struct {
	N   int
	Err error
}

func (d *DataInfo) Unwrap() (int, error) {
	return d.N, d.Err
}

func (c *Conn) Read(data []byte) (n int, err error) {
	tm, err := c.getRdl()
	if err != nil {
		return 0, err
	}

	if c.buffer.Buffered() >= len(data) {
		return c.buffer.Read(data)
	}

	var ch = make(chan error)
	select {
	case <-c.close:
		return 0, net.ErrClosed
	default:
		conn.GoFunc(context.TODO(), func(ctx context.Context) error {
			defer close(ch)
			n, err = c.buffer.Read(data)
			ch <- err
			return nil
		})
		select {
		case <-tm.C:
			return 0, context.DeadlineExceeded
		case err = <-ch:
			return n, err
		}
	}
}

func (c *Conn) Write(data []byte) (int, error) {
	select {
	case <-c.close:
		return 0, net.ErrClosed
	case c.sendPool <- data:
		return len(data), nil
	}
}
func (c *Conn) loopSend(ctx context.Context) error {
	defer c.Close()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.close:
			return nil
		case data := <-c.sendPool:
			_, err := c.ReadWriterReleaser.Write(data)
			if err != nil {
				logrus.Errorf("send channel data failed, session will be closed! error:%v", err)
				return err
			}
			if errors.Is(err, webrtc.ErrConnectionClosed) {
				return net.ErrClosed
			}
		}
	}
}

func (c *Conn) Release() {
	c.Lock()
	defer c.Unlock()
	if c.isRelease {
		return
	}
	c.isRelease = true
	c.ReadWriterReleaser.Release()
}

func (c *Conn) Close() error {
	c.Release()
	if c.IsClose() {
		return nil
	}
	c.Lock()
	defer c.Unlock()
	close(c.close)

	return nil
}
