package conn

import (
	"context"
	"io"
	"net"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/ice"
	"github.com/yixinin/puup/proto"
	"github.com/yixinin/puup/stderr"
)

type PeerId string
type ClientId string
type ClusterName string

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

	sig Signalinger
	pc  *webrtc.PeerConnection

	cmdChan chan DataChannelCommand

	connected chan struct{}
	open      chan struct{}
	close     chan struct{}
}

func newPeer(pc *webrtc.PeerConnection, rid, rcid string, pt webrtc.SDPType, sig Signalinger) *Peer {
	p := &Peer{
		ChannelPool: NewChannelPool(pc, rid, rcid, pt),

		sig: sig,
		pc:  pc,

		cmdChan:   make(chan DataChannelCommand, 1),
		connected: make(chan struct{}, 1),
		open:      make(chan struct{}),
		close:     make(chan struct{}),
	}
	go p.loop()
	return p
}

func NewOfferPeer(sig Signalinger, remoteClientId string) (*Peer, error) {
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return nil, stderr.Wrap(err)
	}

	p := newPeer(pc, "", remoteClientId, webrtc.SDPTypeOffer, sig)
	dc, err := pc.CreateDataChannel("keepalive", nil)
	if err != nil {
		return nil, err
	}

	GoFunc(context.TODO(), func(ctx context.Context) error {
		return p.loopKeepalive(ctx, dc)
	})

	return p, nil
}
func NewAnswerPeer(sig Signalinger, remoteClientId, remoteId string, accept chan ReadWriterReleaser) (*Peer, error) {
	pc, err := webrtc.NewPeerConnection(ice.Config)
	if err != nil {
		return nil, stderr.Wrap(err)
	}

	p := newPeer(pc, remoteId, remoteClientId, webrtc.SDPTypeAnswer, sig)
	p.ChannelPool.accept = accept
	return p, nil
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

func (p *Peer) OnConnectionStateChange(pcs webrtc.PeerConnectionState) {
	logrus.Infof("connection state changed :%s", pcs)
	switch pcs {
	case webrtc.PeerConnectionStateConnected:
		p.connected <- struct{}{}
	case webrtc.PeerConnectionStateFailed, webrtc.PeerConnectionStateDisconnected, webrtc.PeerConnectionStateClosed:
		p.Close()
	}
}

func (p *Peer) Connect(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	p.pc.OnConnectionStateChange(p.OnConnectionStateChange)
	if err := p.SendOffer(ctx); err != nil {
		return err
	}
	if err := p.WaitAnswer(ctx); err != nil {
		return err
	}
	return nil
}

func (p *Peer) Listen(ctx context.Context) error {
	pc := p.pc
	pc.OnConnectionStateChange(p.OnConnectionStateChange)
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
		var packet = proto.Packet{
			From: proto.Client{
				ClientId: p.sig.Id(),
				PeerId:   p.Id,
			},
			To: proto.Client{
				PeerId:   p.RemoteId,
				ClientId: p.RemoteClientId,
			},
			ICECandidate: c,
		}
		err := p.sig.SendPacket(ctx, packet)
		if err != nil {
			logrus.Errorf("send candidate error:%v", err)
		}
	})
	return p.PollOffer(ctx)
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
