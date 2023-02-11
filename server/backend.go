package server

import (
	"sync"
	"time"
)

type Backend struct {
	sync.RWMutex
	name     string
	alive    uint64
	sessions map[string]*SdpPair
	pendings map[string]*SdpPair
}

func NewBackend(name string) *Backend {
	return &Backend{
		name:     name,
		alive:    uint64(time.Now().Unix()),
		sessions: make(map[string]*SdpPair),
		pendings: make(map[string]*SdpPair),
	}
}

func (b *Backend) Alive() bool {
	var now = uint64(time.Now().Unix())
	b.RLock()
	defer b.RUnlock()

	return now-b.alive <= 60*2
}
func (b *Backend) KeepAlive() {
	b.Lock()
	defer b.Unlock()

	b.alive = uint64(time.Now().Unix())
}

func (b *Backend) Session(key string) (*SdpPair, bool) {
	b.RLock()
	defer b.RUnlock()
	v, ok := b.sessions[key]
	return v, ok
}

func (b *Backend) Sessions(f func(k string, v *SdpPair)) {
	b.RLock()
	defer b.RUnlock()

	for k, v := range b.sessions {
		f(k, v)
	}
}

func (b *Backend) PreConnect(key string, p *SdpPair) {
	b.Lock()
	defer b.Unlock()

	b.pendings[key] = p
}

func (b *Backend) Pending(key string) (*SdpPair, bool) {
	b.RLock()
	defer b.RUnlock()
	v, ok := b.pendings[key]
	return v, ok
}

func (b *Backend) Pendings(f func(k string, v *SdpPair)) {
	b.RLock()
	defer b.RUnlock()

	for k, v := range b.pendings {
		f(k, v)
	}
}

func (b *Backend) Connect(key string) *SdpPair {
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

func (b *Backend) Offline(key string) {
	b.Lock()
	defer b.Unlock()

	delete(b.pendings, key)
	delete(b.sessions, key)
}
