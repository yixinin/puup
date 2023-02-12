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

func GetPostSdpURL(serverAddr string) string {
	return fmt.Sprintf("%s/sdp", serverAddr)
}

type PostCandidateReq struct {
	Name      string               `json:"name"`
	Id        string               `json:"id"`
	Type      webrtc.SDPType       `json:"type"`
	Candidate *webrtc.ICECandidate `json:"ice"`
}

func GetPostCandidateURL(serverAddr string) string {
	return fmt.Sprintf("%s/candidate", serverAddr)
}

type OfflineReq struct {
	Name string `form:"name"`
	Id   string `form:"id"`
}

func GetOfflineURL(serverAddr, backendName, id string) string {
	var vals = url.Values{}
	vals.Add("name", backendName)
	vals.Add("id", id)
	return fmt.Sprintf("%s/offline?%s", serverAddr, vals.Encode())
}

type FetchReq struct {
	Name string         `form:"name"`
	Type webrtc.SDPType `form:"type"`
	Id   string         `form:"id"`
}

func GetFetchURL(serverAddr, backendName, id string) string {
	var vals = url.Values{}
	vals.Add("name", backendName)
	vals.Add("id", id)
	return fmt.Sprintf("%s/fetch?%s", serverAddr, vals.Encode())
}

type FetchAck struct {
	Id         string                    `json:"id"`
	Sdp        webrtc.SessionDescription `json:"sdp"`
	Candidates []*webrtc.ICECandidate    `json:"ices"`
}
