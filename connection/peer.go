package connection

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/proto"
)

type PeerType string

const (
	Offer  PeerType = "offer"
	Answer PeerType = "answer"
)

func (p PeerType) Url() string {
	switch p {
	case Offer:
		return "front"
	case Answer:
		return "back"
	}
	panic("unexpect peer type")
}

type Command string

const (
	CmdConnect    Command = "connnect"
	CmdDisConnect Command = "disconnect"
	CmdEOF        Command = "EOF"
)

type DataChannelCommand struct {
	Cmd   Command
	Label string
}

type Peer struct {
	sync.Mutex

	Type   PeerType
	sigcli Signalinger

	dataEvent chan string

	pc               *webrtc.PeerConnection
	connReleaseEvent chan string

	// cmd     *webrtc.DataChannel
	cmdChan chan DataChannelCommand

	connected chan struct{}

	actives map[string]*DataChannel
	idles   map[string]*DataChannel

	close   chan struct{}
	isClose bool

	idx uint32
}

func NewOfferPeer(pc *webrtc.PeerConnection, sigClient Signalinger) (*Peer, error) {
	p := newPeer(pc, Offer)
	dc, err := pc.CreateDataChannel("keepalive", nil)
	if err != nil {
		return nil, err
	}
	go p.keepalive(context.Background(), dc)
	p.sigcli = sigClient
	go p.loopDataEvent(nil)
	return p, nil
}
func NewAnswerPeer(pc *webrtc.PeerConnection, sigClient Signalinger, onConn chan net.Conn) *Peer {
	p := newPeer(pc, Answer)
	p.sigcli = sigClient
	go p.loopDataEvent(onConn)
	return p
}

func newPeer(pc *webrtc.PeerConnection, pt PeerType) *Peer {
	p := &Peer{
		pc:               pc,
		Type:             pt,
		connReleaseEvent: make(chan string),
		cmdChan:          make(chan DataChannelCommand, 1),

		actives:   make(map[string]*DataChannel),
		idles:     make(map[string]*DataChannel),
		connected: make(chan struct{}, 1),
		close:     make(chan struct{}),
		dataEvent: make(chan string, 1024),
	}
	go p.loop()
	return p
}
func (p *Peer) keepalive(ctx context.Context, dc *webrtc.DataChannel) {
	if dc == nil {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		logrus.Debugf("%s recv keepalive %s", dc.Label(), msg.Data)
	})
	var open = make(chan struct{})
	dc.OnOpen(func() {
		close(open)
	})

	dc.OnClose(func() {
		cancel()
	})
	<-open
	var tk = time.NewTicker(30 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.close:
			return
		case <-tk.C:
			logrus.Debugf("send keepalive to %s", dc.Label())
			err := dc.Send([]byte{':', ':'})
			if err != nil {
				logrus.Errorf("send keep alive error:%v", err)
			}
		}
	}
}

func (p *Peer) loop() {
	for {
		select {
		case <-p.close:
			return
		case label := <-p.connReleaseEvent:
			func() {
				p.Lock()
				defer p.Unlock()

				v, ok := p.actives[label]
				if ok {
					delete(p.actives, label)
					p.idles[label] = v
				}
			}()
		case cmd := <-p.cmdChan:
			func() {
				p.Lock()
				defer p.Unlock()
				_, ok := p.actives[cmd.Label]
				if !ok {
					logrus.Warning("cmd for idle conn, ignore.")
					return
				}

				switch cmd.Cmd {
				case CmdConnect:
				case CmdDisConnect:
				case CmdEOF:
				}
			}()
		}
	}
}

func (p *Peer) Connect() error {
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	pc := p.pc

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		if dc == nil {
			return
		}
		logrus.Infof("data channel %s created", dc.Label())
		if dc.Label() == "keepalive" {
			go p.keepalive(context.Background(), dc)
			return
		}
		p.Lock()
		defer p.Unlock()
		laddr, raddr, err := AddrFromLabel(backendName, frontendName, dc.Label())
		if err != nil {
			logrus.Errorf("parse label error:%v", err)
			return
		}
		switch raddr.Type {
		case Keepalive:
			go p.keepalive(context.Background(), dc)
			return
		}
		d := NewDataChannel(dc, laddr, raddr, p.dataEvent)
		p.idles[dc.Label()] = d
	})
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		err := p.sigcli.SendCandidate(ctx, c)
		if err != nil {
			logrus.Errorf("send candidate error:%v", err)
		}
	})

	pc.OnConnectionStateChange(func(pcs webrtc.PeerConnectionState) {
		logrus.Infof("connection state changed :%s", pcs)
		switch pcs {
		case webrtc.PeerConnectionStateConnected:
			p.connected <- struct{}{}
		case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateDisconnected, webrtc.PeerConnectionStateClosed:
			p.Close()
		}
	})

	switch p.Type {
	case Offer:
		if err := p.SendOffer(ctx); err != nil {
			return err
		}
		if err := p.WaitAnswer(ctx); err != nil {
			return err
		}
	case Answer:
		return p.PollOffer(ctx)
	}
	return nil
}

func (p *Peer) loopDataEvent(onConn chan net.Conn) {
	for {
		select {
		case <-p.close:
			return
		case key := <-p.dataEvent:
			p.Lock()
			conn, ok := p.actives[key]
			if ok {
				conn.dataEvent <- key
			}
			p.Unlock()
			if onConn != nil {
				conn := p.getIdleDataChannel(key)
				if conn != nil {
					onConn <- conn
					conn.dataEvent <- struct{}{}
				}
			}
		}
	}
}

func (p *Peer) CreateConn(label string, laddr, raddr net.Addr) (*DataChannel, error) {
	// create new channel
	wdc, err := p.pc.CreateDataChannel(label, nil)
	if err != nil {
		return nil, err
	}

	dc := NewDataChannel(wdc, laddr, raddr, p.dataEvent)
	p.Lock()
	defer p.Unlock()
	p.actives[dc.dc.Label()] = dc
	return conn, nil
}

func (p *Peer) GetWebConn(label string) (*Conn, error) {
	// try get session from idles
	if conn := p.getIdleDataChannel(label); conn != nil {
		return conn, nil
	}
	idx := atomic.AddUint32(&p.idx, 1)
	label = fmt.Sprintf("web.%d", idx)
	laddr := NewWebAddr(frontendName, uint64(idx))
	raddr := NewWebAddr(backendName, uint64(idx))
	return p.CreateConn(label, laddr, raddr)
}
func (p *Peer) GetProxyConn(port uint16) (*Conn, error) {
	label := fmt.Sprintf("%s.%d", proto.Proxy, port)
	// try get session from idles
	if conn := p.getIdleDataChannel(label); conn != nil {
		return conn, nil
	}
	return p.CreateConn(label)
}
func (p *Peer) GetSshConn() (*Conn, error) {
	label := string(proto.Ssh)
	// try get session from idles
	if conn := p.getIdleDataChannel(label); conn != nil {
		return conn, nil
	}
	return p.CreateConn(label)
}

func (c *Peer) getIdleDataChannel(label string) *DataChannel {
	c.Lock()
	defer c.Unlock()
	if len(c.idles) == 0 {
		return nil
	}
	var dc *DataChannel
	if label != "" {
		dc = c.idles[label]
	} else {
		for k, v := range c.idles {
			if !v.IsClose() {
				label = k
				dc = v
				break
			}
		}
	}

	if dc != nil {
		c.actives[label] = dc
		return dc
	}
	return nil
}

func (p *Peer) IsClose() bool {
	p.Lock()
	defer p.Unlock()
	return p.isClose
}

func (p *Peer) Close() error {
	p.Lock()
	defer p.Unlock()
	if p.isClose {
		return nil
	}
	p.isClose = true
	err := p.pc.Close()
	close(p.close)
	return err
}
