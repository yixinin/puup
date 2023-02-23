package net

import (
	"context"
	"net"
	"sync"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/ice"
	"github.com/yixinin/puup/net/conn"
)

type Listener struct {
	sync.RWMutex

	serverName string
	sigClient  *conn.SignalingClient

	onClose chan string
	accept  chan conn.ReadWriterReleaser

	acceptWeb   chan conn.ReadWriterReleaser
	acceptSsh   chan conn.ReadWriterReleaser
	acceptFile  chan conn.ReadWriterReleaser
	acceptProxy chan conn.ReadWriterReleaser
	peers       map[string]*conn.Peer

	isClose bool
	close   chan struct{}
}

func NewListener(sigAddr, serverName string) *Listener {
	lis := &Listener{
		sigClient:   conn.NewSignalingClient(webrtc.SDPTypeAnswer, sigAddr, serverName),
		serverName:  serverName,
		onClose:     make(chan string, 1),
		accept:      make(chan conn.ReadWriterReleaser, 100),
		acceptWeb:   make(chan conn.ReadWriterReleaser, 40),
		acceptSsh:   make(chan conn.ReadWriterReleaser, 10),
		acceptFile:  make(chan conn.ReadWriterReleaser, 10),
		acceptProxy: make(chan conn.ReadWriterReleaser, 40),
		peers:       make(map[string]*conn.Peer, 1),
		close:       make(chan struct{}, 1),
	}
	go lis.loop()
	return lis
}

func (l *Listener) Accept() (net.Conn, error) {
	for {
		select {
		case <-l.close:
			return nil, net.ErrClosed
		case rwr := <-l.acceptWeb:
			conn := NewConn(rwr)
			return conn, nil
		}
	}
}
func (l *Listener) AcceptFile() (net.Conn, error) {
	for {
		select {
		case <-l.close:
			return nil, net.ErrClosed
		case rwr := <-l.acceptFile:
			conn := NewConn(rwr)
			return conn, nil
		}
	}
}
func (l *Listener) AcceptSsh() (net.Conn, error) {
	for {
		select {
		case <-l.close:
			return nil, net.ErrClosed
		case rwr := <-l.acceptSsh:
			conn := NewConn(rwr)
			return conn, nil
		}
	}
}
func (l *Listener) AcceptProxy() (net.Conn, error) {
	for {
		select {
		case <-l.close:
			return nil, net.ErrClosed
		case rwr := <-l.acceptProxy:
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
FOR:
	for {
		select {
		case <-l.close:
			return
		case dc := <-l.accept:
			switch dc.Label().ChannelType {
			case conn.Web:
				l.acceptWeb <- dc
			case conn.Ssh:
				l.acceptSsh <- dc
			case conn.File:
				l.acceptFile <- dc
			case conn.Proxy:
				l.acceptProxy <- dc
			}
		case clientId := <-l.sigClient.NewClient():
			logrus.Debugf("recv new client: %s", clientId)
			if _, ok := l.GetPeer(clientId); ok {
				logrus.Debugf("client %s already connected", clientId)
				continue FOR
			}
			pc, err := webrtc.NewPeerConnection(ice.Config)
			if err != nil {
				logrus.Errorf("new peer sig failed:%v", err)
				continue
			}

			p := conn.NewAnswerPeer(pc, l.serverName, clientId, l.sigClient, l.accept)
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
		}
	}
}
