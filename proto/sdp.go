package proto

type PostSdpReq struct {
	Name string `json:"name"`
	Key  string `json:"key"`
	Sdp  []byte `json:"sdp"`
}
