package proto

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/pion/webrtc/v3"
)

type PostSdpReq struct {
	Name string                    `json:"name,omitempty"`
	Id   string                    `json:"id,omitempty"`
	Sdp  webrtc.SessionDescription `json:"sdp,omitempty"`
}

func GetPostSdpURL(sigAddr string) string {
	return fmt.Sprintf("%s/api/sdp", sigAddr)
}

type PostCandidateReq struct {
	Name      string               `json:"name,omitempty"`
	Id        string               `json:"id,omitempty"`
	Type      webrtc.SDPType       `json:"type,omitempty"`
	Candidate *webrtc.ICECandidate `json:"ice,omitempty"`
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

func GetFetchURL(sigAddr string, tp webrtc.SDPType, serverName, id string) string {
	var vals = url.Values{}
	vals.Add("name", serverName)
	vals.Add("type", strconv.Itoa(int(tp)))
	if id != "" {
		vals.Add("id", id)
	}

	return fmt.Sprintf("%s/api/fetch?%s", sigAddr, vals.Encode())
}

type FetchAck struct {
	Id         string                     `json:"id,omitempty"`
	Sdp        *webrtc.SessionDescription `json:"sdp,omitempty"`
	Candidates []*webrtc.ICECandidate     `json:"ices,omitempty"`
}

type GetClusterReq struct {
	Name string `form:"name"`
}

type GetClusterAck struct {
	Ids []string `json:"ids"`
}
