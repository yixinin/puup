package conn

import (
	"context"
	"net"
	"sync"
	"sync/atomic"

	"github.com/pion/webrtc/v3"
	"github.com/yixinin/puup/stderr"
)

type ChannelPool struct {
	sync.RWMutex

	Type  webrtc.SDPType
	CType ChannelType
	chs   map[string]*Channel
	pc    *webrtc.PeerConnection

	accept chan ReadWriterReleaser
	idx    uint64

	close chan struct{}
}

func NewChannelPool(pc *webrtc.PeerConnection, t webrtc.SDPType, ctype ChannelType) *ChannelPool {
	pool := &ChannelPool{
		Type:   t,
		CType:  ctype,
		chs:    make(map[string]*Channel, 8),
		pc:     pc,
		accept: make(chan ReadWriterReleaser),
	}
	return pool
}

func (p *ChannelPool) loop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (p *ChannelPool) GetChannel(labels ...string) (*Channel, error) {
	switch len(labels) {
	case 1:
		p.RLock()
		defer p.RUnlock()
		ch, _ := p.chs[labels[0]]
		if ch == nil {
			return nil, stderr.New("not found")
		}
		return ch, nil
	case 0:
		p.Lock()
		defer p.Unlock()
	default:
		p.RLock()
		defer p.RUnlock()
		for _, v := range p.chs {
			return v, nil
		}
	}
	atomic.AddUint64(&p.idx, 1)
	label := NewLabel(p.Type, p.CType, p.idx) // offer.web:1
	dc, err := p.pc.CreateDataChannel(label.String(), nil)
	if err != nil {
		return nil, err
	}
	ch := NewOfferChannel(dc, label)
	p.chs[label.String()] = ch
	return ch, nil
}

func (p *ChannelPool) OnChannel(dc *webrtc.DataChannel) {
	p.Lock()
	defer p.Unlock()
	p.chs[dc.Label()] = NewAnswerChannel(dc, p.accept)
}

func (p *ChannelPool) Accept() (ReadWriterReleaser, error) {
	select {
	case <-p.close:
		return nil, net.ErrClosed
	case conn := <-p.accept:
		return conn, nil
	}
}
