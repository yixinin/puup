package proto

import "github.com/pion/webrtc/v3"

type PostCandidateReq struct {
	Name      string               `json:"name"`
	Key       string               `json:"key"`
	Candidate *webrtc.ICECandidate `json:"icd"`
}
