package net

import (
	"context"
	"net"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/ice"
	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/stderr"
)

type Listener struct {
	sync.RWMutex

	sigClient *conn.SignalingClient

	onClose chan string
	onConn  chan *conn.Conn

	peers map[string]*conn.Peer

	isClose bool
	close   chan struct{}
}

func NewListener(serverName, sigAddr string) *Listener {
	lis := &Listener{
		sigClient: conn.NewSignalingClient(conn.Answer, sigAddr, serverName),

		onClose: make(chan string, 1),
		onConn:  make(chan *conn.Conn, 10),
		peers:   make(map[string]*conn.Peer, 1),
		close:   make(chan struct{}, 1),
	}
	go lis.loop()
	return lis
}

func (l *Listener) Accept() (net.Conn, error) {
	for {
		select {
		case <-l.close:
			return nil, net.ErrClosed
		case cc := <-l.onConn:
			p, ok := l.peers[cc.ClientId()]
			if !ok {
				return nil, stderr.New("peer is invalid")
			}
			conn := NewConn(cc, p.ReleaseChan())
			return conn, nil
		}
	}
}

func (l *Listener) AddPeer(key string, p *conn.Peer) {
	l.Lock()
	defer l.Unlock()
	l.peers[key] = p
}

func (l *Listener) GetPeer(key string) (*conn.Peer, bool) {
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
FOR:
	for {
		select {
		case <-l.close:
			return
		case clientId := <-l.sigClient.NewClient():
			if _, ok := l.GetPeer(clientId); ok {
				continue FOR
			}
			pc, err := webrtc.NewPeerConnection(ice.Config)
			if err != nil {
				logrus.Errorf("new peer sig failed:%v", err)
				continue
			}

			p := conn.NewAnswerPeer(pc, clientId, l.sigClient, l.onConn)
			l.AddPeer(clientId, p)
			go func() {
				if err := p.Connect(context.TODO()); err != nil {
					logrus.Errorf("peer connect failed:%v", err)
					l.DelPeer(clientId)
					return
				}
			}()
		case key := <-l.onClose:
			l.DelPeer(key)
			l.sigClient.Offline(context.Background(), key)
		}
	}
}
