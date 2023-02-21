package conn

import (
	"net"

	"github.com/pion/webrtc/v3"
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

	accept   chan ReadWriterReleaser
	recvData chan []byte
}

func NewOfferChannel(dc *webrtc.DataChannel, label *Label) *Channel {
	ch := newChannel(dc, webrtc.SDPTypeOffer)
	ch.label = label
	ch.laddr = NewOfferAddr(label)
	return ch
}

func NewAnswerChannel(dc *webrtc.DataChannel, accept chan ReadWriterReleaser) *Channel {
	ch := newChannel(dc, webrtc.SDPTypeAnswer)
	ch.accept = accept
	return ch
}

func newChannel(dc *webrtc.DataChannel, typ webrtc.SDPType) *Channel {
	ch := &Channel{
		status:   Opening,
		Type:     typ,
		dc:       dc,
		open:     make(chan struct{}),
		close:    make(chan struct{}),
		recvData: make(chan []byte, 100),
	}
	dc.OnMessage(ch.OnMessage)
	dc.OnOpen(func() {
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
		if c.Active() {
			if c.Type == webrtc.SDPTypeAnswer && c.accept != nil {
				c.accept <- c
			}
		}
		c.recvData <- msg.Data
	}
}

func (c *Channel) Active() bool {
	if c.status != Idle {
		return false
	}
	c.status = Active
	return true
}
func (c *Channel) Release() {
	if c.status != Active {
		return
	}
	c.status = Idle
}

func (c *Channel) Close() error {
	c.status = Closed
	select {
	case <-c.close:
		return nil
	default:
	}
	close(c.close)
	return nil
}

func (c *Channel) Label() string {
	return c.dc.Label()
}
func (p *Channel) LocalAddr() net.Addr {
	return p.laddr
}
func (p *Channel) RemoteAddr() net.Addr {
	return p.raddr
}

func (p *Channel) Read(data []byte) (int, error) {
	select {
	case <-p.close:
		return 0, net.ErrClosed
	case buf := <-p.recvData:
		if len(data) < len(buf) {
			panic("read out of memeroy")
		}
		n := copy(data, buf)
		return int(n), nil
	}
}

func (p *Channel) Write(data []byte) (int, error) {
	<-p.open
	err := p.dc.Send(data)
	return len(data), err
}
