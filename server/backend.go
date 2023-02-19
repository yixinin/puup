package server

import "sync"

type Backend struct {
	sync.RWMutex

	sigAddr  string
	sessions map[string]*Session
}

func NewBackend(name string) *Backend {
	return &Backend{
		sigAddr:  name,
		sessions: make(map[string]*Session),
	}
}

func (b *Backend) GetSession(id string) *Session {
	b.Lock()
	defer b.Unlock()

	sess, ok := b.sessions[id]
	if ok {
		return sess
	}
	sess = NewSession(id)
	b.sessions[id] = sess
	return sess
}

func (b *Backend) DelSession(id string) {
	b.Lock()
	defer b.Unlock()
	delete(b.sessions, id)
}

func (b *Backend) RandSession() *Session {
	b.RLock()
	defer b.RUnlock()
	for _, sess := range b.sessions {
		return sess
	}
	return nil
}
