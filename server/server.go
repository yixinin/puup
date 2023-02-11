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
	sync.RWMutex
	backends map[string]*Backend
}

func (s *Server) GetBackend(name string) (*Backend, bool) {
	s.RLock()
	defer s.RUnlock()
	b, ok := s.backends[name]
	return b, ok
}

func (s *Server) MustGetBackend(name string) (*Backend, bool) {
	s.Lock()
	defer s.Unlock()
	b, ok := s.backends[name]
	if !ok {
		b = NewBackend(name)
		s.backends[name] = b
	}
	return b, ok
}

func (s *Server) AddBackend(name string, b *Backend) {
	s.Lock()
	defer s.Unlock()
	s.backends[name] = b
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
	g := e.Group("/api")
	g.Use(middles.CommonLogInterceptor())

	g.POST("/sdp/back", s.PostBackSdp)
	g.POST("/sdp/front", s.PostFrontSdp)
	g.POST("/candidate/front", s.PostFrontCandidate)
	g.POST("/candidate/back", s.PostBackCandidate)
	g.GET("/conninfo/back", s.GetBackConnectionInfo)
	g.GET("/conninfo/front", s.GetFrontConnectionInfo)
	g.GET("/keepalive/back", s.Keepalive)
	g.HEAD("/offline", s.Offline)

	return e.Run(":8080")
}
