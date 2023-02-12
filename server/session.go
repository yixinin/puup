package server

import (
	"time"

	"github.com/pion/webrtc/v3"
)

type Backend struct {
	backendName string
	sess        map[string]Session
}

func (s *Backend) StartNewSession(id string) {
	var sess = NewSession(id)
}

type Session struct {
	Id   string
	fsdp chan webrtc.SessionDescription
	bsdp chan webrtc.SessionDescription
	bice chan webrtc.ICECandidate
	fice chan webrtc.ICECandidate
}

func NewSession(id string) *Session {
	return &Session{
		Id:   id,
		bsdp: make(chan webrtc.SessionDescription, 1),
		bice: make(chan webrtc.ICECandidate, 10),
		fsdp: make(chan webrtc.SessionDescription, 1),
		fice: make(chan webrtc.ICECandidate, 10),
	}
}
func (s *Session) StateChange(state string) {

}
func (s *Session) RecvFrontendSdp(sdp webrtc.SessionDescription) {
	s.sdp <- sdp
}
func NewBackend(name string) *Session {
	return &Session{
		backendName: name,
		alive:       uint64(time.Now().Unix()),
		sessions:    make(map[string]*SdpPair),
		pendings:    make(map[string]*SdpPair),
	}
}

func (b *Session) Alive() bool {
	var now = uint64(time.Now().Unix())
	b.RLock()
	defer b.RUnlock()

	return now-b.alive <= 60*2
}
func (b *Session) KeepAlive() {
	b.Lock()
	defer b.Unlock()

	b.alive = uint64(time.Now().Unix())
}

func (b *Session) Session(key string) (*SdpPair, bool) {
	b.RLock()
	defer b.RUnlock()
	v, ok := b.sessions[key]
	return v, ok
}

func (b *Session) Sessions(f func(k string, v *SdpPair)) {
	b.RLock()
	defer b.RUnlock()

	for k, v := range b.sessions {
		f(k, v)
	}
}

func (b *Session) PreConnect(key string, p *SdpPair) {
	b.Lock()
	defer b.Unlock()

	b.pendings[key] = p
}

func (b *Session) Pending(key string) (*SdpPair, bool) {
	b.RLock()
	defer b.RUnlock()
	v, ok := b.pendings[key]
	return v, ok
}

func (b *Session) Pendings(f func(k string, v *SdpPair)) {
	b.RLock()
	defer b.RUnlock()

	for k, v := range b.pendings {
		f(k, v)
	}
}

func (b *Session) Connect(key string) *SdpPair {
	b.Lock()
	defer b.Unlock()

	p, ok := b.pendings[key]
	if !ok {
		return nil
	}

	b.sessions[key] = p
	delete(b.pendings, key)
	return p
}

func (b *Session) Offline(key string) {
	b.Lock()
	defer b.Unlock()

	delete(b.pendings, key)
	delete(b.sessions, key)
}
