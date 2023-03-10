package conn

import (
	"bufio"
	"io"
	"net"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/stderr"
)

type ChanStatus string

const (
	Opening ChanStatus = "opening"
	Idle    ChanStatus = "idle"
	Active  ChanStatus = "active"
	Closed  ChanStatus = "closed"
)

type ChannelReader struct {
	batchSize int
	recvData  chan []byte
}

func (c *ChannelReader) Release() {
	close(c.recvData)
}

func NewChannelReader(batchSize int) *ChannelReader {
	return &ChannelReader{
		batchSize: batchSize,
		recvData:  make(chan []byte, 10),
	}
}

func (r *ChannelReader) OnData(data []byte) {
	var size = len(data)
	for i := 0; i < size; i += r.batchSize {
		end := min(i+r.batchSize, size)
		r.recvData <- data[i:end]
	}
}

func (r *ChannelReader) Read(p []byte) (int, error) {
	data, ok := <-r.recvData
	if !ok {
		return 0, io.EOF
	}
	if len(data) > len(p) {
		panic("read out of memeroy!")
	}
	return copy(p, data), nil
}

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

	rd     *ChannelReader
	buffer *bufio.Reader
	// sendData chan []byte
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
		batchSize: 4096,
		status:    Idle,
		Type:      typ,
		dc:        dc,
		label:     label,
		release:   release,
		open:      make(chan struct{}),
		close:     make(chan struct{}),
	}
	logrus.Debugf("register %s on message", ch.dc.Label())
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
		c.rd.OnData(msg.Data)
		logrus.Debugf("%s recv data %d", c.dc.Label(), len(msg.Data))
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

	var rd = NewChannelReader(c.batchSize)
	c.rd = rd
	c.buffer = bufio.NewReader(rd)

	return true
}
func (c *Channel) Release() {
	logrus.Debugf("start to release %s conn %s ", c.dc.Label(), c.status)
	if c.status != Active {
		return
	}
	c.rd.Release()
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

func (c *Channel) Read(data []byte) (n int, err error) {
	select {
	case <-c.close:
		if c.buffer.Buffered() > 0 {
			return c.buffer.Read(data)
		}
		return 0, io.EOF
	default:
		return c.buffer.Read(data)
	}
}

func (c *Channel) Write(data []byte) (int, error) {
	var size = len(data)
	writen := 0
	for i := 0; i < size; i += c.batchSize {
		writen = min(size, i+c.batchSize)
		select {
		case <-c.close:
			return 0, net.ErrClosed
		case <-c.open:
			err := c.dc.Send(data[i:writen])
			if err != nil {
				return writen, stderr.Wrap(err)
			}
		}
	}
	return writen, nil
}

// func (c *Channel) loopWrite(ctx context.Context) error {
// 	var total = 0
// 	defer func() {
// 		logrus.Infof("%s send loop end, total send: %d", c.dc.Label(), total)
// 	}()
// 	<-c.open

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return ctx.Err()
// 		case data, ok := <-c.sendData:
// 			if !ok {
// 				return nil
// 			}
// 			logrus.Debugf("%s send data %d", c.dc.Label(), len(data))
// 			total += len(data)
// 			if c.wrf != nil {
// 				c.wrf.Write(data)
// 			}
// 			err := c.dc.Send(data)
// 			if err != nil {
// 				logrus.Errorf("send data error")
// 			}
// 		}
// 	}
// }
