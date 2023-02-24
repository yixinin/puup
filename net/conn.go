package net

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/yixinin/puup/net/conn"
)

type Conn struct {
	sync.RWMutex

	conn.ReadWriterReleaser

	isRelease bool
	close     chan struct{}
	rdl       time.Time
}

func NewConn(rwr conn.ReadWriterReleaser) *Conn {
	c := &Conn{
		ReadWriterReleaser: rwr,
		close:              make(chan struct{}),
	}
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

	var ch = make(chan error)
	defer close(ch)

	conn.GoFunc(context.TODO(), func(ctx context.Context) error {
		n, err = c.ReadWriterReleaser.Read(data)
		select {
		case <-ch:
		case ch <- err:
		}
		return err
	})
	select {
	case <-tm.C:
		select {
		case <-c.close:
			return 0, net.ErrClosed
		default:
			return 0, context.DeadlineExceeded
		}
	case err = <-ch:
		return n, err
	}
}

func (c *Conn) Write(data []byte) (int, error) {
	select {
	case <-c.close:
		return 0, net.ErrClosed
	default:
		return c.ReadWriterReleaser.Write(data)
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
