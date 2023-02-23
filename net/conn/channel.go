package conn

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

type ChanStatus string

const (
	Opening ChanStatus = "opening"
	Idle    ChanStatus = "idle"
	Active  ChanStatus = "active"
	Closed  ChanStatus = "closed"
)

type Channel struct {
	status ChanStatus
	Type   webrtc.SDPType
	label  *Label

	laddr, raddr net.Addr
	dc           *webrtc.DataChannel

	open  chan struct{}
	close chan struct{}

	accept  chan ReadWriterReleaser
	release chan ReadWriterReleaser

	batchSize int

	recvData chan []byte
	sendData chan []byte
}

func NewOfferChannel(sname, cid string, dc *webrtc.DataChannel, label *Label, release chan ReadWriterReleaser) *Channel {
	ch := newChannel(dc, webrtc.SDPTypeOffer, release, label)
	ch.laddr = NewClientAddr(cid, label)
	ch.raddr = NewServerAddr(sname, label)
	return ch
}

func NewAnswerChannel(sname, cid string, dc *webrtc.DataChannel, label *Label, accept, release chan ReadWriterReleaser) *Channel {
	ch := newChannel(dc, webrtc.SDPTypeAnswer, release, label)
	ch.accept = accept
	ch.raddr = NewClientAddr(cid, label)
	ch.laddr = NewServerAddr(sname, label)
	return ch
}

func newChannel(dc *webrtc.DataChannel, typ webrtc.SDPType, release chan ReadWriterReleaser, label *Label) *Channel {
	ch := &Channel{
		batchSize: 512,
		status:    Idle,
		Type:      typ,
		dc:        dc,
		label:     label,
		release:   release,
		open:      make(chan struct{}),
		close:     make(chan struct{}),
	}

	dc.OnMessage(ch.OnMessage)
	dc.OnOpen(func() {
		logrus.Debugf("channel %s opend", ch.Label().String())
		select {
		case <-ch.open:
			return
		default:
			close(ch.open)
		}
	})

	dc.OnClose(func() {
		select {
		case <-ch.close:
			return
		default:
			close(ch.close)
		}
	})
	return ch
}

func (c *Channel) OnMessage(msg webrtc.DataChannelMessage) {
	select {
	case <-c.close:
		return
	case <-c.open:
		if c.TakeConn() {
			if c.Type == webrtc.SDPTypeAnswer && c.accept != nil {
				logrus.Debugf("channel %s accept", c.Label().String())
				c.accept <- c
			}
		}
		var size = len(msg.Data)
		logrus.Debugf("%s recv %d data", c.dc.Label(), size)
		for i := 0; i < len(msg.Data); i += c.batchSize {
			c.recvData <- msg.Data[i:min(i+c.batchSize, size)]
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *Channel) TakeConn() bool {
	if c.status != Idle {
		return false
	}
	logrus.Debugf("channel %s taken", c.Label().String())
	c.status = Active
	c.sendData = make(chan []byte, 10)
	c.recvData = make(chan []byte, 10)
	GoFunc(context.TODO(), func(ctx context.Context) error {
		return c.loopWrite(ctx)
	})
	return true
}
func (c *Channel) Release() {
	logrus.Debugf("%s %s release call", c.dc.Label(), c.status)
	if c.status != Active {
		return
	}
	c.status = Idle
	c.release <- c
	go func() {
		for {
			select {
			case <-c.recvData:
				time.Sleep(1 * time.Millisecond)
			case <-c.sendData:
				time.Sleep(1 * time.Millisecond)
			default:
				close(c.recvData)
				close(c.sendData)
				return
			}
		}
	}()
}

func (c *Channel) Close() error {
	c.status = Closed
	select {
	case <-c.close:
		return nil
	default:
	}
	close(c.close)
	return c.dc.Close()
}

func (c *Channel) Label() *Label {
	return c.label
}
func (c *Channel) LocalAddr() net.Addr {
	return c.laddr
}
func (c *Channel) RemoteAddr() net.Addr {
	return c.raddr
}

func (c *Channel) Read(data []byte) (int, error) {
	select {
	case <-c.close:
		return 0, io.EOF
	case b := <-c.recvData:
		n := copy(data, b)
		logrus.Debugf("read %d", n)
		return n, nil
	}
}

func (c *Channel) Write(data []byte) (int, error) {
	select {
	case <-c.close:
		return 0, net.ErrClosed
	case c.sendData <- data:
		return len(data), nil
	}
}

func (c *Channel) loopWrite(ctx context.Context) error {
	<-c.open
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.close:
			return nil
		case data, ok := <-c.sendData:
			if !ok {
				return nil
			}
			err := c.dc.Send(data)
			if err != nil {
				logrus.Errorf("send data error")
			}
			logrus.Debugf("%s send data %d", c.dc.Label(), len(data))
		}
	}
}
