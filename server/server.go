package server

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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
	websocket.Upgrader

	cluster map[string]*Cluster
}

func (s *Server) AddBackend(name string, id string, conn *websocket.Conn) {
	s.Lock()
	defer s.Unlock()
	c, ok := s.cluster[name]
	if !ok {
		c = NewCluster(name)
		s.cluster[name] = c
	}
	c.AddBackend(id, conn)
}
func (s *Server) GetBackends(name string) []string {
	s.RLock()
	defer s.RUnlock()
	c, ok := s.cluster[name]
	if !ok {
		return nil
	}
	var ids = make([]string, len(c.backends))
	for k := range c.backends {
		ids = append(ids, k)
	}
	return ids
}

func (s *Server) GetBackend(name string, id string) (*Client, bool) {
	s.RLock()
	defer s.RUnlock()
	c, ok := s.cluster[name]
	if !ok {
		return nil, false
	}

	return c.GetBackend(id)
}

func (s *Server) DelBackend(name, id string) {
	s.Lock()
	defer s.Unlock()
	c, ok := s.cluster[name]
	if !ok {
		return
	}
	c.DelBackend(id)
	if len(c.backends) == 0 {
		delete(s.cluster, name)
	}
}

func (s *Server) AddFrontend(name string, id string, conn *websocket.Conn) bool {
	s.Lock()
	defer s.Unlock()
	c, ok := s.cluster[name]
	if !ok {
		return false
	}
	c.AddFrontend(id, conn)
	return true
}

func (s *Server) GetFrontend(name string, id string) (*Client, bool) {
	s.RLock()
	defer s.RUnlock()
	c, ok := s.cluster[name]
	if !ok {
		return nil, false
	}
	return c.GetFrontend(id)
}

func (s *Server) DelFrontend(name, id string) {
	s.Lock()
	defer s.Unlock()
	c, ok := s.cluster[name]
	if !ok {
		return
	}
	c.DelFrontend(id)
}

func NewServer() *Server {
	return &Server{
		cluster: make(map[string]*Cluster),
	}
}

func (s *Server) Run(ctx context.Context) error {
	e := gin.New()
	e.Use(gin.Recovery())
	e.StaticFS("/web", http.Dir("dist"))
	e.Use(middles.Cors)
	g := e.Group("/api")

	// g.POST("/sdp", s.PostSdp)
	// g.POST("/candidate", s.PostCandidate)
	// g.GET("/fetch", s.Fetch)
	// g.HEAD("/offline", s.Offline)
	g.GET("/cluster", s.Cluster)

	g.Any("/signalling", s.WsSignalling)

	return e.Run(":8080")
}
