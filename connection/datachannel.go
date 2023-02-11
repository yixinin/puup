package connection

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

type DataChannel struct {
	sync.RWMutex

	peerType PeerType

	backendName string
	fontendName string

	dc    *webrtc.DataChannel
	open  chan struct{}
	close chan error

	recvChan  chan []byte
	dataEvent chan string

	ended  bool
	status DcStatus
}

func (c *DataChannel) Label() string {
	return c.dc.Label()
}

func (c *DataChannel) GetStatus() DcStatus {
	return c.status
}

func (c *DataChannel) SetStatus(status DcStatus) bool {
	c.Lock()
	defer c.Unlock()
	if status == c.status {
		return false
	}
	c.status = status
	return true
}

func (c *DataChannel) IsClose() bool {
	c.RLock()
	defer c.RUnlock()

	return c.status == Closed
}
func (c *DataChannel) IsEnd() bool {
	c.RLock()
	defer c.RUnlock()
	return c.IsClose() || c.ended
}

func NewDataChannel(dc *webrtc.DataChannel, laddr, raddr net.Addr, dataEvent chan string) *DataChannel {
	dcConn := &DataChannel{
		dc:        dc,
		open:      make(chan struct{}, 1),
		close:     make(chan error, 1),
		recvChan:  make(chan []byte, 1024),
		dataEvent: dataEvent,
		status:    Opening,
	}
	dc.OnOpen(func() {
		logrus.Infof("channel %s opened", dc.Label())
		if dcConn.SetStatus(Idle) {
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

func (c *DataChannel) LocalAddr() net.Addr {
	switch c.peerType {
	case Offer:
		switch c.chanType {
		case Ssh:
			return NewSshAddr(c.fontendName)
		case File:
			return NewFileAddr(c.fontendName, idx)
		}
		return NewLabelAddr(c.fontendName, c.dc.Label())
	case Answer:
		return NewLabelAddr(c.backendName, c.dc.Label())
	}
	return nil
}
func (c *DataChannel) RemoteAddr() net.Addr {
	switch c.peerType {
	case Offer:
		return NewLabelAddr(c.backendName, c.dc.Label())
	case Answer:
		return NewLabelAddr(c.fontendName, c.dc.Label())
	}
	return nil
}

func (c *DataChannel) Read(data []byte) (n int, err error) {
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

func (c *DataChannel) Write(data []byte) (int, error) {
	select {
	case <-c.close:
		return 0, net.ErrClosed
	case <-c.open:
	}
	logrus.Debugf("write %d data to %s", len(data), c.dc.Label())
	err := c.dc.Send(data)
	return len(data), err
}

func (c *DataChannel) OnMessage(msg webrtc.DataChannelMessage) {
	select {
	case <-c.close:
		logrus.Debugf("closed dc recieve %d msg, size: %d", c.dc.ID(), len(msg.Data))
		return
	case c.recvChan <- msg.Data:
		c.dataEvent <- c.dc.Label()
		logrus.Debugf("%s recv data %d", c.dc.Label(), len(msg.Data))
	}
}
