package server

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/pion/webrtc/v3"
	"github.com/yixinin/puup/proto"
)

func (s *Server) PostCandidate(c *gin.Context) {
	var req proto.PostCandidateReq
	c.MustBindWith(&req, binding.JSON)
	b := s.GetBackend(req.Name)
	sess := b.GetSession(req.Id)
	if !sess.IsClose() {
		c.String(200, "connection closed")
		return
	}
	defer c.String(200, "")

	switch req.Type {
	case webrtc.SDPTypeOffer:
		sess.offerIce <- req.Candidate
	case webrtc.SDPTypeAnswer:
		sess.answerIce <- req.Candidate
	}
}
