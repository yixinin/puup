package conn

import (
	"net"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

type DcStatus string

const (
	Opening DcStatus = "opening"
	Idle    DcStatus = "idle"
	Active  DcStatus = "active"
	Closed  DcStatus = "closed"
)

type Conn struct {
	sync.RWMutex

	sigAddr  string
	clientId string
	status   DcStatus

	open  chan struct{}
	close chan error

	dc        *webrtc.DataChannel
	recvChan  chan []byte
	dataEvent chan string

	localAddr, remoteAddr *LabelAddr
}

func (c *Conn) ClientId() string {
	return c.clientId
}
func (c *Conn) Label() string {
	return c.dc.Label()
}

func (c *Conn) SetStatus(status DcStatus) bool {
	c.Lock()
	defer c.Unlock()
	if status == c.status {
		return false
	}
	c.status = status
	return true
}

func (c *Conn) IsClose() bool {
	c.RLock()
	defer c.RUnlock()

	return c.status == Closed
}

func NewConn(dc *webrtc.DataChannel, dataEvent chan string) *Conn {
	dcConn := &Conn{
		dc:        dc,
		open:      make(chan struct{}, 1),
		close:     make(chan error, 1),
		recvChan:  make(chan []byte, 1024),
		dataEvent: dataEvent,
		status:    Opening,
	}
	dc.OnOpen(func() {
		logrus.Infof("channel %s opened", dc.Label())
		select {
		case <-dcConn.open:
		default:
			close(dcConn.open)
		}
	})
	dc.OnClose(func() {
		if dcConn.SetStatus(Closed) {
			close(dcConn.close)
		}
	})
	dc.OnMessage(dcConn.OnMessage)
	return dcConn
}

func (c *Conn) getLocalAddr() net.Addr {
	c.RLock()
	defer c.RUnlock()
	if c.localAddr != nil {
		return c.localAddr
	}
	return nil
}

func (c *Conn) getRemoteAddr() net.Addr {
	c.RLock()
	defer c.RUnlock()
	if c.remoteAddr != nil {
		return c.remoteAddr
	}
	return nil
}

func (c *Conn) LocalAddr() net.Addr {
	if addr := c.getLocalAddr(); addr != nil {
		return addr
	}
	c.Lock()
	defer c.Unlock()
	var err error
	c.localAddr, c.remoteAddr, err = AddrFromLabel(c.sigAddr, c.clientId, c.Label())
	if err != nil {
		logrus.Errorf("parse addr error")
	}
	return c.localAddr
}
func (c *Conn) RemoteAddr() net.Addr {
	if addr := c.getRemoteAddr(); addr != nil {
		return addr
	}
	c.Lock()
	defer c.Unlock()
	var err error
	c.localAddr, c.remoteAddr, err = AddrFromLabel(c.sigAddr, c.clientId, c.Label())
	if err != nil {
		logrus.Errorf("parse addr error")
	}
	return c.remoteAddr
}

func (c *Conn) Read(data []byte) (n int, err error) {
	select {
	case <-c.open:
	case <-c.close:
		return 0, net.ErrClosed
	}

	src := <-c.recvChan
	if len(src) > len(data) {
		// TODO add buffer
		panic("read from datachannel out of memery!")
	}
	n = copy(data, src)
	return n, nil
}

func (c *Conn) Write(data []byte) (int, error) {
	select {
	case <-c.close:
		return 0, net.ErrClosed
	case <-c.open:
	}
	logrus.Debugf("write %d data to %s", len(data), c.dc.Label())
	err := c.dc.Send(data)
	return len(data), err
}

func (c *Conn) OnMessage(msg webrtc.DataChannelMessage) {
	select {
	case <-c.close:
		logrus.Debugf("closed dc recieve %d msg, size: %d", c.dc.ID(), len(msg.Data))
		return
	case c.recvChan <- msg.Data:
		c.dataEvent <- c.dc.Label()
		logrus.Debugf("%s recv data %d", c.dc.Label(), len(msg.Data))
	}
}
