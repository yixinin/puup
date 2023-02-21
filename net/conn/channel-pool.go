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

	serverName, clientId string

	Type webrtc.SDPType
	chs  map[string]ReadWriterReleaser
	pc   *webrtc.PeerConnection

	accept  chan ReadWriterReleaser
	release chan ReadWriterReleaser

	idx uint64

	close chan struct{}
}

func NewChannelPool(serverName, clientId string, pc *webrtc.PeerConnection, t webrtc.SDPType) *ChannelPool {
	pool := &ChannelPool{
		Type:       t,
		serverName: serverName,
		clientId:   clientId,
		chs:        make(map[string]ReadWriterReleaser, 8),
		pc:         pc,
	}
	go pool.loop(context.TODO())
	return pool
}

func (p *ChannelPool) loop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case dc := <-p.release:
			p.Lock()
			p.chs[dc.Label().String()] = dc
			p.Unlock()
		}
	}
}

func (p *ChannelPool) Get(ct ChannelType, labels ...string) (ReadWriterReleaser, error) {
	p.RLock()
	defer p.RUnlock()
	switch len(labels) {
	case 1:
		ch, _ := p.chs[labels[0]]
		if ch == nil {
			return nil, stderr.New("not found")
		}
		return ch, nil
	case 0:
	default:

		for _, v := range p.chs {
			return v, nil
		}
	}
	atomic.AddUint64(&p.idx, 1)
	label := NewLabel(ct, p.idx) // offer.web:1
	dc, err := p.pc.CreateDataChannel(label.String(), nil)
	if err != nil {
		return nil, err
	}
	ch := NewOfferChannel(p.serverName, p.clientId, dc, label, p.release)
	return ch, nil
}

func (p *ChannelPool) OnChannelOpen(dc *webrtc.DataChannel) error {
	p.Lock()
	defer p.Unlock()
	label, err := parseLabel(dc.Label())
	if err != nil {
		return err
	}
	p.chs[dc.Label()] = NewAnswerChannel(p.serverName, p.clientId, dc, label, p.accept)
	return nil
}
func (p *ChannelPool) OnRelease(label string) {
	p.Lock()
	defer p.Unlock()
	ch, ok := p.chs[label]
	if ok && ch != nil {
		ch.Release()
	}
}

func (p *ChannelPool) Accept() (ReadWriterReleaser, error) {
	select {
	case <-p.close:
		return nil, net.ErrClosed
	case conn := <-p.accept:
		return conn, nil
	}
}
