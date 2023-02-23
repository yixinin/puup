package conn

import (
	"context"
	"io"
	"net"
	"strings"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
)

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
	Label() *Label
	TakeConn() bool
	Release()
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

type Peer struct {
	*ChannelPool

	serverName string
	clientId   string

	Type   webrtc.SDPType
	sigCli Signalinger
	pc     *webrtc.PeerConnection

	cmdChan chan DataChannelCommand

	connected chan struct{}
	open      chan struct{}
	close     chan struct{}
}

func newPeer(pc *webrtc.PeerConnection, serverName, clientId string, pt webrtc.SDPType, sigClient Signalinger) *Peer {
	p := &Peer{
		serverName:  serverName,
		clientId:    clientId,
		pc:          pc,
		Type:        pt,
		sigCli:      sigClient,
		cmdChan:     make(chan DataChannelCommand, 1),
		ChannelPool: NewChannelPool(serverName, clientId, pc, pt),
		connected:   make(chan struct{}, 1),
		open:        make(chan struct{}),
		close:       make(chan struct{}),
	}
	go p.loop()
	return p
}

func NewOfferPeer(pc *webrtc.PeerConnection, serverName string, sigClient Signalinger) (*Peer, error) {
	clientId := strings.ReplaceAll(uuid.NewString(), "-", "")
	p := newPeer(pc, serverName, clientId, webrtc.SDPTypeOffer, sigClient)
	dc, err := pc.CreateDataChannel("keepalive", nil)
	if err != nil {
		return nil, err
	}

	GoFunc(context.TODO(), func(ctx context.Context) error {
		return p.loopKeepalive(ctx, dc)
	})

	return p, nil
}
func NewAnswerPeer(pc *webrtc.PeerConnection, serverName, cid string, sigClient Signalinger, accept chan ReadWriterReleaser) *Peer {
	p := newPeer(pc, serverName, cid, webrtc.SDPTypeAnswer, sigClient)
	p.ChannelPool.accept = accept
	return p
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
				p.ChannelPool.OnRelease(cmd.Label)
			case CmdEOF:
			}

		}
	}
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
		if err := p.ChannelPool.OnChannelOpen(dc); err != nil {
			logrus.Errorf("add channel error:%v", err)
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
		p.handleChannel(dc)
	})
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		logrus.Debugf("send ice")
		err := p.sigCli.SendCandidate(ctx, p.clientId, p.Type, c)
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
	case webrtc.SDPTypeOffer:
		if err := p.SendOffer(ctx); err != nil {
			return err
		}
		if err := p.WaitAnswer(ctx); err != nil {
			return err
		}
	case webrtc.SDPTypeAnswer:
		return p.PollOffer(ctx)
	}
	return nil
}

func (p *Peer) IsClose() bool {
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
	close(p.close)
	return p.pc.Close()
}
