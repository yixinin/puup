package server

import (
	"time"

	"github.com/pion/webrtc/v3"
)

type Session struct {
	Id string

	offer     chan webrtc.SessionDescription
	answer    chan webrtc.SessionDescription
	offerIce  chan *webrtc.ICECandidate
	answerIce chan *webrtc.ICECandidate

	close chan struct{}
}

func NewSession(id string) *Session {
	return &Session{
		Id:        id,
		offer:     make(chan webrtc.SessionDescription, 1),
		offerIce:  make(chan *webrtc.ICECandidate, 10),
		answer:    make(chan webrtc.SessionDescription, 1),
		answerIce: make(chan *webrtc.ICECandidate, 10),
		close:     make(chan struct{}),
	}
}

func (s *Session) IsClose() bool {
	select {
	case <-s.close:
		return true
	default:
		return false
	}
}

func (s *Session) Close() error {
	if s.IsClose() {
		return nil
	}
	close(s.close)
	go func() {
		<-time.After(time.Second)
		close(s.answer)
		close(s.offer)
		close(s.answerIce)
		close(s.offerIce)
	}()

	return nil
}
