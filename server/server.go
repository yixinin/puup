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

	// cluster map[string]*Cluster

	sessions map[string]Session
}

func (s *Server) AddBackend(name string, id string, conn *websocket.Conn) {
	s.Lock()
	defer s.Unlock()
	sess, ok := s.sessions[name]
	if !ok {
		sess = Session{
			ClusterName: name,
			Backends:    make(map[string]*Client),
			Frontends:   make(map[string]*Client),
		}
		s.sessions[name] = sess
	}
	sess.Backends[id] = &Client{
		Id:    id,
		conn:  conn,
		Peers: make(map[string]string),
	}
}
func (s *Server) GetBackends(name string) []string {
	s.RLock()
	defer s.RUnlock()
	sess, ok := s.sessions[name]
	if !ok {
		return nil
	}
	var ids = make([]string, len(sess.Backends))
	for k := range sess.Backends {
		ids = append(ids, k)
	}
	return ids
}

func (s *Server) GetBackend(name string, id string) (*Client, bool) {
	s.RLock()
	defer s.RUnlock()
	sess, ok := s.sessions[name]
	if !ok {
		return nil, false
	}

	b, ok := sess.Backends[id]
	return b, ok
}

func (s *Server) DelBackend(name, id string) {
	s.Lock()
	defer s.Unlock()
	sess, ok := s.sessions[name]
	if !ok {
		return
	}
	delete(sess.Backends, id)
	if len(sess.Backends) == 0 {
		delete(s.sessions, name)
	}
}

func (s *Server) AddFrontend(name string, id string, conn *websocket.Conn) bool {
	s.Lock()
	defer s.Unlock()
	sess, ok := s.sessions[name]
	if !ok {
		return false
	}
	sess.Frontends[id] = &Client{
		Id:    id,
		conn:  conn,
		Peers: make(map[string]string),
	}
	return true
}

func (s *Server) GetFrontend(name string, id string) (*Client, bool) {
	s.RLock()
	defer s.RUnlock()
	sess, ok := s.sessions[name]
	if !ok {
		return nil, false
	}
	f, ok := sess.Frontends[id]
	return f, ok
}

func (s *Server) DelFrontend(name, id string) {
	s.Lock()
	defer s.Unlock()
	sess, ok := s.sessions[name]
	if !ok {
		return
	}
	delete(sess.Frontends, id)
}

func NewServer() *Server {
	return &Server{
		sessions: make(map[string]Session),
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
