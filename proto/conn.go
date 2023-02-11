package proto

import "github.com/pion/webrtc/v3"

type GetConnectionInfoReq struct {
	Name string `form:"name"`
	Key  string `form:"key"`
}

type GetConnectionInfoAck struct {
	Sdp        []byte                 `json:"sdp"`
	Candidates []*webrtc.ICECandidate `json:"icd"`
	// Resend     bool                   `json:"rs"`
}

type OfflineReq struct {
	Name string `form:"name"`
	Key  string `form:"key"`
}

type KeepAliveReq struct {
	Name string `form:"name"`
}

type KeepaliveAck struct {
	Keys []string `json:"ks"`
}
