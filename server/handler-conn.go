package server

import (
	"github.com/gin-gonic/gin"
	"github.com/yixinin/puup/proto"
)

func map2slice[K comparable, V any](m map[K]V) []V {
	var s = make([]V, 0, len(m))
	for _, v := range m {
		s = append(s, v)
	}
	return s
}

func (s *Server) GetBackConnectionInfo(c *gin.Context) {
	var req proto.GetConnectionInfoReq
	c.BindQuery(&req)

	var ack = new(proto.GetConnectionInfoAck)
	defer c.JSON(200, ack)

	b, ok := s.GetBackend(req.Name)
	if !ok {
		return
	}

	p, ok := b.Session(req.Key)
	if !ok {
		p = b.Connect(req.Key)
	}

	if p != nil {
		ack.Candidates = map2slice(p.Front.Candidates)
		ack.Sdp = p.Front.Sdp
	}
}

func (s *Server) GetFrontConnectionInfo(c *gin.Context) {
	var req proto.GetConnectionInfoReq
	c.BindQuery(&req)
	var ack = new(proto.GetConnectionInfoAck)
	defer c.JSON(200, ack)

	b, ok := s.GetBackend(req.Name)
	if !ok {
		return
	}
	v, ok := b.Session(req.Key)
	if ok {
		ack.Candidates = map2slice(v.Back.Candidates)
		ack.Sdp = v.Back.Sdp
	}
}

func (s *Server) Keepalive(c *gin.Context) {
	var req proto.KeepAliveReq
	c.BindQuery(&req)
	var ack = new(proto.KeepaliveAck)

	defer c.JSON(200, ack)

	b, exist := s.MustGetBackend(req.Name)
	if !exist {
		return
	}
	b.KeepAlive()
	b.Pendings(func(key string, p *SdpPair) {
		ack.Keys = append(ack.Keys, key)
	})
}

func (s *Server) Offline(c *gin.Context) {
	var req proto.OfflineReq
	c.BindQuery(&req)
	b, ok := s.GetBackend(req.Name)
	if !ok {
		return
	}
	b.Offline(req.Key)
}
