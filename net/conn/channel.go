package conn

import (
	"context"
	"net"

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
		status:   Idle,
		Type:     typ,
		dc:       dc,
		label:    label,
		release:  release,
		open:     make(chan struct{}),
		close:    make(chan struct{}),
		recvData: make(chan []byte, 100),
		sendData: make(chan []byte, 100),
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
	GoFunc(context.TODO(), func(ctx context.Context) error {
		return ch.loopWrite(ctx)
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
		logrus.Debugf("channel %s recv data %d", c.Label().String(), len(msg.Data))
		c.recvData <- msg.Data
	}
}

func (c *Channel) TakeConn() bool {
	if c.status != Idle {
		logrus.Debugf("channel %s take fail, %s", c.Label().String(), c.status)
		return false
	}
	logrus.Debugf("channel %s taken", c.Label().String())
	c.status = Active
	return true
}
func (c *Channel) Release() {
	logrus.Debugf("%s %s release call", c.dc.Label(), c.status)
	if c.status != Active {
		return
	}
	c.status = Idle
	c.release <- c
	for {
		// drop buffed data
		select {
		case data := <-c.recvData:
			logrus.Debugf("%s drop recv data %d", c.dc.Label(), len(data))
		case data := <-c.sendData:
			logrus.Debugf("%s drop send data %d", c.dc.Label(), len(data))
		default:
			return
		}
	}
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
		return 0, net.ErrClosed
	case buf := <-c.recvData:
		if len(data) < len(buf) {
			panic("read out of memeroy")
		}
		n := copy(data, buf)
		return int(n), nil
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
		case data := <-c.sendData:
			err := c.dc.Send(data)
			if err != nil {
				logrus.Errorf("send data error")
			}
			logrus.Debugf("%s send data %d", c.dc.Label(), len(data))
		}
	}
}
