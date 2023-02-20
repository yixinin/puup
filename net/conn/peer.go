package conn

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/stderr"
)

type DcStatus string

const (
	Opening DcStatus = "opening"
	Idle    DcStatus = "idle"
	Active  DcStatus = "active"
	Closed  DcStatus = "closed"
)

type PeerType string

const (
	Offer  PeerType = "offer"
	Answer PeerType = "answer"
)

func (p PeerType) SdpTYpe() webrtc.SDPType {
	switch p {
	case Offer:
		return webrtc.SDPTypeOffer
	case Answer:
		return webrtc.SDPTypeAnswer
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

type ReadWriterReleaser interface {
	io.ReadWriter
	Release()
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

type Peer struct {
	sync.Mutex

	serverName string
	clientId   string

	Type   PeerType
	sigCli Signalinger
	pc     *webrtc.PeerConnection

	status DcStatus
	data   *webrtc.DataChannel

	recvData chan []byte
	accept   chan ReadWriterReleaser

	cmdChan chan DataChannelCommand

	connected chan struct{}
	open      chan struct{}

	laddr, raddr *PeerAddr
	close        chan struct{}
}

func newPeer(pc *webrtc.PeerConnection, serverName string, pt PeerType, sigClient Signalinger) *Peer {
	p := &Peer{
		serverName: serverName,
		pc:         pc,
		Type:       pt,
		sigCli:     sigClient,
		status:     Opening,
		recvData:   make(chan []byte, 10),
		cmdChan:    make(chan DataChannelCommand, 1),

		connected: make(chan struct{}, 1),
		open:      make(chan struct{}),
		close:     make(chan struct{}),
	}
	go p.loop()
	return p
}

func NewOfferPeer(pc *webrtc.PeerConnection, serverName string, sigClient Signalinger) (*Peer, error) {
	p := newPeer(pc, serverName, Offer, sigClient)
	p.clientId = strings.ReplaceAll(uuid.NewString(), "-", "")

	dc, err := pc.CreateDataChannel("keepalive", nil)
	if err != nil {
		return nil, err
	}

	GoFunc(context.TODO(), func(ctx context.Context) error {
		return p.loopKeepalive(ctx, dc)
	})

	return p, nil
}
func NewAnswerPeer(pc *webrtc.PeerConnection, serverName, id string, sigClient Signalinger, accept chan ReadWriterReleaser) *Peer {
	p := newPeer(pc, serverName, Answer, sigClient)
	p.clientId = id
	p.accept = accept
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

func (p *Peer) setStatus(status DcStatus) bool {
	p.Lock()
	defer p.Unlock()
	if status == p.status {
		return false
	}
	p.status = status
	return true
}

func (p *Peer) loop() {
	for {
		select {
		case <-p.close:
			return
		case cmd := <-p.cmdChan:
			switch cmd.Cmd {
			case CmdConnect:

			case CmdDisConnect:

			case CmdEOF:
			}

		}
	}
}

func (p *Peer) loopCommand(ctx context.Context, dc *webrtc.DataChannel) error {
	var ch = make(chan error)
	defer close(ch)

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		var cmd DataChannelCommand
		err := json.Unmarshal(msg.Data, &cmd)
		if err != nil {
			select {
			case ch <- err:
			case <-ch:
			}
		}
		p.cmdChan <- cmd
	})
	return <-ch
}

func (p *Peer) GetConn() (ReadWriterReleaser, error) {
	t := time.NewTimer(2 * time.Minute)
	defer t.Stop()
	select {
	case <-p.open:
		return p, nil
	case <-t.C:
		return nil, context.DeadlineExceeded
	}
}

func (p *Peer) loopDataChannel(ctx context.Context, dc *webrtc.DataChannel) error {
	dc.OnOpen(func() {
		close(p.open)
		p.setStatus(Idle)
		var id = int(*p.data.ID())
		p.laddr = NewPeerAddr(p.serverName, id, p.Type)
		switch p.Type {
		case Offer:
			p.raddr = NewPeerAddr(p.serverName, id, Answer)
		case Answer:
			p.raddr = NewPeerAddr(p.serverName, id, Offer)
		}
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		select {
		case <-p.close:
		case p.recvData <- msg.Data:
			if p.setStatus(Active) {
				p.accept <- p
			}
		}
	})

	dc.OnClose(func() {
		p.setStatus(Closed)
		p.Close()
	})
	return nil
}
func (p *Peer) ClientId() string {
	return p.clientId
}

func (p *Peer) handleChannel(dc *webrtc.DataChannel) {
	switch ChannelType(dc.Label()) {
	case Keepalive:
		GoFunc(context.TODO(), func(ctx context.Context) error {
			return p.loopKeepalive(ctx, dc)
		})
		return
	case Cmd:
		GoFunc(context.TODO(), func(ctx context.Context) error {
			return p.loopCommand(ctx, dc)
		})

	default:
		p.data = dc
		GoFunc(context.TODO(), func(ctx context.Context) error {
			return p.loopDataChannel(ctx, dc)
		})
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
		p.handleChannel(dc)
	})
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		err := p.sigCli.SendCandidate(ctx, p.clientId, p.Type.SdpTYpe(), c)
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
		dc, err := p.pc.CreateDataChannel("data", nil)
		if err != nil {
			return err
		}
		p.handleChannel(dc)

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

func (p *Peer) Release() {
	p.setStatus(Idle)
}

func (p *Peer) LocalAddr() net.Addr {
	return p.laddr
}
func (p *Peer) RemoteAddr() net.Addr {
	return p.raddr
}
