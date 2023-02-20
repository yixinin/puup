package proto

import (
	"fmt"
	"net/url"

	"github.com/pion/webrtc/v3"
)

type PostSdpReq struct {
	Name string                    `json:"name"`
	Id   string                    `json:"id"`
	Sdp  webrtc.SessionDescription `json:"sdp"`
}

func GetPostSdpURL(sigAddr string) string {
	return fmt.Sprintf("%s/api/sdp", sigAddr)
}

type PostCandidateReq struct {
	Name      string               `json:"name"`
	Id        string               `json:"id"`
	Type      webrtc.SDPType       `json:"type"`
	Candidate *webrtc.ICECandidate `json:"ice"`
}

func GetPostCandidateURL(sigAddr string) string {
	return fmt.Sprintf("%s/api/candidate", sigAddr)
}

type OfflineReq struct {
	Name string `form:"name"`
	Id   string `form:"id"`
}

func GetOfflineURL(sigAddr, serverName, id string) string {
	var vals = url.Values{}
	vals.Add("name", serverName)
	vals.Add("id", id)
	return fmt.Sprintf("%s/api/offline?%s", sigAddr, vals.Encode())
}

type FetchReq struct {
	Name string         `form:"name"`
	Type webrtc.SDPType `form:"type"`
	Id   string         `form:"id"`
}

func GetFetchURL(sigAddr, serverName, id string) string {
	var vals = url.Values{}
	vals.Add("name", serverName)
	vals.Add("id", id)
	return fmt.Sprintf("%s/api/fetch?%s", sigAddr, vals.Encode())
}

type FetchAck struct {
	Id         string                     `json:"id"`
	Sdp        *webrtc.SessionDescription `json:"sdp"`
	Candidates []*webrtc.ICECandidate     `json:"ices"`
}
