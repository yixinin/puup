package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/yixinin/puup/proto"
)

func (s *Server) PostFrontSdp(c *gin.Context) {
	var req proto.PostSdpReq
	c.MustBindWith(&req, binding.JSON)

	defer c.String(200, "")

	b, _ := s.MustGetBackend(req.Name)
	p, ok := b.Session(req.Key)
	if !ok {
		p, ok = b.Pending(req.Key)
		if !ok {
			p = &SdpPair{
				Back:  NewClientInfo(),
				Front: NewClientInfo(),
			}
			b.PreConnect(req.Key, p)
		}
	}

	p.Front.Sdp = req.Sdp
}

func (s *Server) PostBackSdp(c *gin.Context) {
	var req proto.PostSdpReq
	c.MustBindWith(&req, binding.JSON)
	defer c.String(200, "")

	b, ok := s.GetBackend(req.Name)
	if !ok {
		return
	}
	p, ok := b.Session(req.Key)
	if !ok {
		p, ok = b.Pending(req.Key)
		if !ok {
			p = &SdpPair{
				Back:  NewClientInfo(),
				Front: NewClientInfo(),
			}
			b.PreConnect(req.Key, p)
		}
	}
	p.Back.Sdp = req.Sdp
}
