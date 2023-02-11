package server

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/yixinin/puup/proto"
)

func (s *Server) PostFrontCandidate(c *gin.Context) {
	var req proto.PostCandidateReq
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

	var cd = req.Candidate
	var key = fmt.Sprintf("%d%d", cd.Protocol, cd.Port)

	p.Front.Candidates[key] = cd
}

func (s *Server) PostBackCandidate(c *gin.Context) {
	var req proto.PostCandidateReq
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

	var cd = req.Candidate
	var key = fmt.Sprintf("%d%d", cd.Protocol, cd.Port)
	p.Back.Candidates[key] = cd
}
