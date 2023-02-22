package conn

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
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
		release:    make(chan ReadWriterReleaser, 1),
		close:      make(chan struct{}),
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
			logrus.Debug(dc.Label(), "released")
			p.Lock()
			p.chs[dc.Label().String()] = dc
			p.Unlock()
		}
	}
}

func (p *ChannelPool) Get(ct ChannelType, labels ...string) (ReadWriterReleaser, error) {
	p.RLock()
	defer p.RUnlock()
	var key string
	defer delete(p.chs, key)
	var ch ReadWriterReleaser
	switch len(labels) {
	case 1:
		key = labels[0]
		if ch = p.chs[key]; ch == nil {
			return nil, stderr.New("not found")
		}
		if ch.TakeConn() {
			return ch, nil
		}
	default:
		for key, ch = range p.chs {
			if ch.TakeConn() {
				return ch, nil
			}
		}
	}
	for i := 0; i < 5; i++ {
		atomic.AddUint64(&p.idx, 1)
		var label = NewLabel(ct, p.idx)
		dc, err := p.pc.CreateDataChannel(label.String(), nil)
		if err != nil {
			return nil, err
		}
		ch = NewOfferChannel(p.serverName, p.clientId, dc, label, p.release)
		if ch.TakeConn() {
			return ch, nil
		}
	}

	return nil, stderr.New("cannot take conn")
}

func (p *ChannelPool) OnChannelOpen(dc *webrtc.DataChannel) error {
	p.Lock()
	defer p.Unlock()
	label, err := parseLabel(dc.Label())
	if err != nil {
		return err
	}
	p.chs[dc.Label()] = NewAnswerChannel(p.serverName, p.clientId, dc, label, p.accept, p.release)
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
func (p *ChannelPool) Close() error {
	select {
	case <-p.close:
		return nil
	default:
	}
	close(p.close)
	return nil
}
