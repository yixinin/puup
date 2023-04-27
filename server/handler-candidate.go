package server

// func (s *Server) PostCandidate(c *gin.Context) {
// 	var req proto.PostCandidateReq
// 	c.MustBindWith(&req, binding.JSON)
// 	b := s.GetBackend(req.Name)
// 	sess := b.MustGetSession(req.Id)
// 	if sess.IsClose() {
// 		c.String(200, "session closed")
// 		return
// 	}
// 	defer c.String(200, "")

// 	switch req.Type {
// 	case webrtc.SDPTypeOffer:
// 		sess.offerIce <- req.Candidate
// 	case webrtc.SDPTypeAnswer:
// 		sess.answerIce <- req.Candidate
// 	}
// }
