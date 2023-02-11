package pnet

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

type Conn struct {
	sync.RWMutex
	*DataChannel

	isClose bool

	isRelease bool
	release   chan string
	close     chan struct{}
	// new data received
	dataEvent chan struct{}

	rdl      time.Time
	buffer   *bufio.Reader
	sendPool chan []byte
}

func NewConn(dc *DataChannel, release chan string) *Conn {
	sess := &Conn{
		DataChannel: dc,
		release:     release,
		close:       make(chan struct{}),
		dataEvent:   make(chan struct{}, 1),
		buffer:      bufio.NewReader(dc),
		sendPool:    make(chan []byte),
	}
	dc.setStatus(Active)
	go sess.loop()
	return sess
}

func (c *Conn) IsEOF() bool {
	c.RLock()
	defer c.RUnlock()
	return c.ended
}

func (s *Conn) IsClose() bool {
	s.RLock()
	defer s.RUnlock()

	return s.isClose
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

	if c.buffer.Buffered() > 0 {
		return c.buffer.Read(data)
	}

	if c.IsEOF() {
		return 0, io.EOF
	}

	select {
	case <-c.close:
		return 0, net.ErrClosed
	case <-tm.C:
		return 0, context.DeadlineExceeded
	case <-c.dataEvent:
		return c.buffer.Read(data)
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
func (c *Conn) loop() {
	for {
		select {
		case <-c.close:
			return
		case data := <-c.sendPool:
			_, err := c.DataChannel.Write(data)
			if err != nil {
				logrus.Errorf("send channel data failed, session will be closed! error:%v", err)
				c.Close()
			}
			if errors.Is(err, webrtc.ErrConnectionClosed) {
				return
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
	c.release <- c.dc.Label()
}

func (c *Conn) Close() error {
	c.Release()
	c.Lock()
	defer c.Unlock()
	if c.isClose {
		return nil
	}
	c.isClose = true
	close(c.close)

	c.setStatus(Idle)
	return nil
}
