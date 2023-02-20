package server

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/pion/webrtc/v3"
	"github.com/yixinin/puup/middles"
)

type ClientInfo struct {
	Candidates map[string]*webrtc.ICECandidate
	Sdp        []byte
}

func NewClientInfo() *ClientInfo {
	return &ClientInfo{
		Candidates: make(map[string]*webrtc.ICECandidate),
	}
}

type SdpPair struct {
	Back  *ClientInfo
	Front *ClientInfo
}

type Server struct {
	sync.Mutex
	backends map[string]*Backend
}

func (s *Server) GetBackend(name string) *Backend {
	s.Lock()
	defer s.Unlock()
	b, ok := s.backends[name]
	if !ok {
		b = NewBackend(name)
		s.backends[name] = b
	}
	return b
}

func (s *Server) DelBackend(name string) {
	s.Lock()
	defer s.Unlock()

	delete(s.backends, name)
}

func NewServer() *Server {
	return &Server{
		backends: make(map[string]*Backend),
	}
}

func (s *Server) Run(ctx context.Context) error {
	e := gin.New()
	e.Use(gin.Recovery())
	e.Use(middles.Cors)
	g := e.Group("/api", middles.Logging())

	g.POST("/sdp", s.PostSdp)
	g.POST("/candidate", s.PostCandidate)
	g.GET("/fetch", s.Fetch)
	g.HEAD("/offline", s.Offline)

	return e.Run(":8080")
}
