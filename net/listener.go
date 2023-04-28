package net

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/net/conn"
)

type Listener struct {
	sync.RWMutex

	wsURL       string
	clusterName string
	id          string

	sig conn.Signalinger

	onClose chan string
	accept  chan conn.ReadWriterReleaser

	peers map[string]*conn.Peer

	isClose bool
	close   chan struct{}
}

func NewListener(wsURL, clusterName string) *Listener {
	id := uuid.NewString()
	lis := &Listener{
		id:          id,
		sig:         conn.NewWsBackendSigClient(id, wsURL, clusterName),
		clusterName: clusterName,
		onClose:     make(chan string, 1),
		accept:      make(chan conn.ReadWriterReleaser, 100),
		peers:       make(map[string]*conn.Peer, 1),
		close:       make(chan struct{}, 1),
	}
	go func() {
		if err := lis.sig.Run(context.Background()); err != nil {
			logrus.Errorf("sig disconnected:%v", err)
		}
		lis.sig.Close(context.Background())
	}()
	go lis.loop()
	return lis
}

func (l *Listener) Accept() (net.Conn, error) {
	for {
		select {
		case <-l.close:
			return nil, net.ErrClosed
		case rwr := <-l.accept:
			conn := NewConn(rwr)
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
	tk := time.NewTicker(5 * time.Second)
FOR:
	for {
		select {
		case <-l.close:
			return
		case <-tk.C:
			if l.sig.IsClose() {
				l.sig = conn.NewWsBackendSigClient(l.id, l.wsURL, l.clusterName)
				go func() {
					if err := l.sig.Run(context.TODO()); err != nil {
						logrus.Errorf("client run error:%v", err)
					}
					l.sig.Close(context.Background())
				}()
			}
		case remoteId := <-l.sig.NewPeer():
			logrus.Debugf("recv new client: %s", remoteId)
			if _, ok := l.GetPeer(remoteId); ok {
				logrus.Debugf("client %s already connected", remoteId)
				continue FOR
			}

			p, err := conn.NewAnswerPeer(l.sig, "", remoteId, l.accept)
			if err != nil {
				logrus.Debugf("new peer error:%v", err)
				return
			}

			l.AddPeer(remoteId, p)
			go func() {
				if err := p.Listen(context.TODO()); err != nil {
					logrus.Errorf("peer connect failed:%v", err)
					l.DelPeer(remoteId)
					return
				}
			}()
		case key := <-l.onClose:
			l.DelPeer(key)
		}
	}
}
