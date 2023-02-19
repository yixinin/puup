package conn

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/proto"
	"github.com/yixinin/puup/stderr"
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

	sigAddr  string
	clientId string

	Type PeerType

	connIdx uint32

	sigCli Signalinger
	pc     *webrtc.PeerConnection

	onDataIn         chan string
	connReleaseEvent chan string
	cmdChan          chan DataChannelCommand
	connected        chan struct{}

	actives map[string]*Conn
	idles   map[string]*Conn

	close chan struct{}
}

func NewOfferPeer(pc *webrtc.PeerConnection, sigClient Signalinger) (*Peer, error) {
	p := newPeer(pc, Offer)
	p.sigCli = sigClient
	dc, err := pc.CreateDataChannel("keepalive", nil)
	if err != nil {
		return nil, err
	}

	GoFunc(context.TODO(), func(ctx context.Context) error {
		return p.loopKeepalive(ctx, dc)
	})

	GoFunc(context.TODO(), func(ctx context.Context) error {
		return p.loopDataEvent(context.TODO(), nil)
	})
	return p, nil
}
func NewAnswerPeer(pc *webrtc.PeerConnection, id string, sigClient Signalinger, onConn chan *Conn) *Peer {
	p := newPeer(pc, Answer)
	p.sigCli = sigClient
	go p.loopDataEvent(context.TODO(), onConn)
	return p
}

func newPeer(pc *webrtc.PeerConnection, pt PeerType) *Peer {
	p := &Peer{
		pc:               pc,
		Type:             pt,
		connReleaseEvent: make(chan string),
		cmdChan:          make(chan DataChannelCommand, 1),

		actives:   make(map[string]*Conn),
		idles:     make(map[string]*Conn),
		connected: make(chan struct{}, 1),
		close:     make(chan struct{}),
		onDataIn:  make(chan string, 1024),
	}
	go p.loop()
	return p
}
func (p *Peer) loopKeepalive(ctx context.Context, dc *webrtc.DataChannel) error {
	if dc == nil {
		return stderr.New("data channel is nil ")
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
	// wait data open
	t := time.NewTimer(time.Minute)
	defer t.Stop()
	select {
	case <-t.C:
		return stderr.Wrap(context.DeadlineExceeded)
	case <-open:
	}
	t.Stop()

	var tk = time.NewTicker(30 * time.Second)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.close:
			return nil
		case <-tk.C:
			logrus.Debugf("send keepalive to %s", dc.Label())
			err := dc.Send([]byte{':', ':'})
			if err != nil {
				logrus.Errorf("send keep alive error:%v", err)
			}
		}
	}
}

func (p *Peer) ReleaseChan() chan string {
	return p.connReleaseEvent
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

				conn, ok := p.actives[label]
				if ok {
					conn.SetStatus(Idle)
					delete(p.actives, label)
					p.idles[label] = conn
				}
			}()
		case cmd := <-p.cmdChan:
			func() {
				p.Lock()
				defer p.Unlock()
				switch cmd.Cmd {
				case CmdConnect:
					_, ok := p.actives[cmd.Label]
					if !ok {
						logrus.Warning("cmd for idle conn, ignore.")
						return
					}
				case CmdDisConnect:
					_, ok := p.idles[cmd.Label]
					if !ok {
						logrus.Warning("cmd for active conn, ignore.")
						return
					}
				case CmdEOF:
				}
			}()
		}
	}
}

func (p *Peer) Connect(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	pc := p.pc

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		if dc == nil {
			return
		}
		logrus.Infof("data channel %s created", dc.Label())
		if dc.Label() == string(Keepalive) {
			GoFunc(ctx, func(ctx context.Context) error {
				return p.loopKeepalive(ctx, dc)
			})
			return
		}

		p.Lock()
		defer p.Unlock()

		d := NewConn(dc, p.onDataIn)
		p.idles[dc.Label()] = d
	})
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		err := p.sigCli.SendCandidate(ctx, p.clientId, c)
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

func (p *Peer) loopDataEvent(ctx context.Context, onConn chan *Conn) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.close:
			return nil
		case key := <-p.onDataIn:
			p.Lock()
			conn, ok := p.actives[key]
			if ok {
				conn.dataEvent <- key
			}
			p.Unlock()
			if onConn != nil {
				conn := p.getIdle(key)
				if conn != nil {
					onConn <- conn
					conn.dataEvent <- key
				}
			}
		}
	}
}

func (p *Peer) CreateConn(label string) (*Conn, error) {
	// create new channel
	dc, err := p.pc.CreateDataChannel(label, nil)
	if err != nil {
		return nil, err
	}

	conn := NewConn(dc, p.onDataIn)
	p.Lock()
	defer p.Unlock()
	conn.SetStatus(Active)
	p.actives[conn.Label()] = conn
	return conn, nil
}

func (p *Peer) CreateConnWithAddr(label string, local, remote *LabelAddr) (*Conn, error) {
	conn, err := p.CreateConn(label)
	if err != nil {
		return nil, err
	}
	conn.localAddr = local
	conn.remoteAddr = remote
	return conn, nil
}

func (p *Peer) GetWebConn(labels ...string) (*Conn, error) {
	// try get session from idles
	if len(labels) > 0 && labels[0] != "" {
		if conn := p.getIdle(labels[0]); conn != nil {
			return conn, nil
		}
	}

	idx := atomic.AddUint32(&p.connIdx, 1)
	label := fmt.Sprintf("webx%d", idx)
	laddr := NewWebAddr(p.sigAddr, uint64(idx))
	raddr := NewWebAddr(p.clientId, uint64(idx))
	return p.CreateConnWithAddr(label, laddr, raddr)
}

func (p *Peer) GetFileConn(labels ...string) (*Conn, error) {
	// try get session from idles
	if len(labels) > 0 && labels[0] != "" {
		if conn := p.getIdle(labels[0]); conn != nil {
			return conn, nil
		}
	}

	idx := atomic.AddUint32(&p.connIdx, 1)
	label := fmt.Sprintf("filex%d", idx)
	laddr := NewWebAddr(p.sigAddr, uint64(idx))
	raddr := NewWebAddr(p.clientId, uint64(idx))
	return p.CreateConnWithAddr(label, laddr, raddr)
}
func (p *Peer) GetProxyConn(port uint16) (*Conn, error) {
	label := fmt.Sprintf("%sx%d", proto.Proxy, port)
	// try get session from idles
	if conn := p.getIdle(label); conn != nil {
		return conn, nil
	}
	return p.CreateConn(label)
}
func (p *Peer) GetSshConn() (*Conn, error) {
	label := string(proto.Ssh)
	// try get session from idles
	if conn := p.getIdle(label); conn != nil {
		return conn, nil
	}
	return p.CreateConn(label)
}

func (c *Peer) getIdle(label string) *Conn {
	c.Lock()
	defer c.Unlock()
	if len(c.idles) == 0 {
		return nil
	}
	var conn *Conn
	if label != "" {
		conn = c.idles[label]
	} else {
		for k, v := range c.idles {
			if !v.IsClose() {
				label = k
				conn = v
				break
			}
		}
	}

	if conn != nil {
		delete(c.idles, label)
		conn.SetStatus(Active)
		c.actives[label] = conn
		return conn
	}
	return nil
}

func (p *Peer) IsClose() bool {
	p.Lock()
	defer p.Unlock()
	select {
	case <-p.close:
		return true
	default:
		return false
	}
}

func (p *Peer) Close() error {
	if p.IsClose() {
		return nil
	}
	p.Lock()
	defer p.Unlock()
	err := p.pc.Close()
	close(p.close)
	return err
}
