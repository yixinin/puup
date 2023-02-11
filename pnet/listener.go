package pnet

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/ice"
)

type Listener struct {
	sync.RWMutex

	sigClient *SignalingClient

	frontIn chan string
	onClose chan string
	onConn  chan net.Conn

	peers map[string]*Peer

	isClose bool
	close   chan struct{}
}

func NewListener(name, serverAddr string) *Listener {
	lis := &Listener{
		sigClient: NewAnswerClient(serverAddr, name),
		frontIn:   make(chan string, 10),
		onClose:   make(chan string, 1),
		onConn:    make(chan net.Conn, 10),
		peers:     make(map[string]*Peer, 1),
		close:     make(chan struct{}, 1),
	}
	go lis.loop()
	return lis
}

func (l *Listener) Accept() (net.Conn, error) {
	for {
		select {
		case <-l.close:
			return nil, net.ErrClosed
		case conn := <-l.onConn:
			return conn, nil
		}
	}
}

func (l *Listener) AddPeer(key string, p *Peer) {
	l.Lock()
	defer l.Unlock()
	l.peers[key] = p
}

func (l *Listener) GetPeer(key string) (*Peer, bool) {
	l.RLock()
	defer l.RUnlock()
	p, ok := l.peers[key]
	return p, ok
}

func (l *Listener) DelPeer(key string) {
	l.Lock()
	defer l.Unlock()
	delete(l.peers, key)
}

func (l *Listener) Close() error {
	l.RLock()
	closed := l.isClose
	l.RUnlock()
	if closed {
		return nil
	}

	l.Lock()
	defer l.Unlock()

	l.isClose = true
	close(l.close)
	return nil
}
func (l *Listener) Addr() net.Addr {
	return &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 80,
	}
}

func (l *Listener) loop() {
	var kt = time.NewTicker(5 * time.Second)
	defer kt.Stop()
FOR:
	for {
		select {
		case <-kt.C:
			go l.sigClient.Keepalive(context.Background(), l.frontIn)
		case <-l.close:
			return
		case key := <-l.frontIn:
			if _, ok := l.GetPeer(key); ok {
				continue FOR
			}
			pc, err := webrtc.NewPeerConnection(ice.Config)
			if err != nil {
				logrus.Errorf("new peer connection failed:%v", err)
				continue
			}
			c := l.sigClient.Clone()
			c.FrontendKey = key
			p := NewAnswerPeer(pc, c, l.onConn)
			l.AddPeer(key, p)
			go func() {
				if err := p.Connect(); err != nil {
					logrus.Errorf("peer connect failed:%v", err)
					l.DelPeer(key)
					return
				}
			}()

		case key := <-l.onClose:
			l.DelPeer(key)
			l.sigClient.Offline(context.Background())
		}
	}
}
