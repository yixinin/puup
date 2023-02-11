package pnet

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/stderr"
)

func (p *Peer) SendOffer(ctx context.Context) error {
	offer, err := p.pc.CreateOffer(nil)
	if err != nil {
		return stderr.Wrap(err)
	}
	if err := p.pc.SetLocalDescription(offer); err != nil {
		return stderr.Wrap(err)
	}

	sdp, err := json.Marshal(offer)
	if err != nil {
		return stderr.Wrap(err)
	}

	if err := p.sigCient.PostSdp(ctx, sdp); err != nil {
		return stderr.Wrap(err)
	}
	return nil
}

func (p *Peer) SendAnswer(ctx context.Context) error {
	answer, err := p.pc.CreateAnswer(nil)
	if err != nil {
		return err
	}
	if err := p.pc.SetLocalDescription(answer); err != nil {
		return stderr.Wrap(err)
	}
	sdp, err := json.Marshal(answer)
	if err != nil {
		return stderr.Wrap(err)
	}

	if err := p.sigCient.PostSdp(ctx, sdp); err != nil {
		return stderr.Wrap(err)
	}
	return nil
}
func (p *Peer) WaitAnswer(ctx context.Context) error {
	var answetTick = time.NewTicker(2 * time.Second)
	defer answetTick.Stop()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.connected:
			return nil
		case <-answetTick.C:
			info, err := p.sigCient.GetConnectionInfo(ctx)
			if err != nil {
				return stderr.Wrap(err)
			}
			if len(info.Sdp) > 0 {
				var answer webrtc.SessionDescription
				if err := json.Unmarshal(info.Sdp, &answer); err != nil {
					return stderr.Wrap(err)
				}

				desc := p.pc.RemoteDescription()
				if desc == nil || desc.SDP == "" {
					if err := p.pc.SetRemoteDescription(answer); err != nil {
						return stderr.Wrap(err)
					}
					for _, v := range info.Candidates {
						if err := p.pc.AddICECandidate(v.ToJSON()); err != nil {
							return stderr.Wrap(err)
						}
					}
				} else {
					for _, v := range info.Candidates {
						if err := p.pc.AddICECandidate(v.ToJSON()); err != nil {
							return stderr.Wrap(err)
						}
					}
				}
			}
		}
	}
}
func (p *Peer) PollOffer(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	pc := p.pc
	var tk = time.NewTicker(2 * time.Second)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.connected:
			return nil
		case <-tk.C:
			info, err := p.sigCient.GetConnectionInfo(context.Background())
			if err != nil {
				return stderr.Wrap(err)
			}
			if len(info.Sdp) > 0 {
				desc := pc.RemoteDescription()

				var s webrtc.SessionDescription
				if err := json.Unmarshal(info.Sdp, &s); err != nil {
					return stderr.Wrap(err)
				}

				if s.SDP != "" {
					if desc == nil || desc.SDP == "" {
						if err := pc.SetRemoteDescription(s); err != nil {
							logrus.Errorf("set sdp:%+v,error", s)
							return stderr.Wrap(err)
						}
						if err := p.SendAnswer(context.Background()); err != nil {
							return stderr.Wrap(err)
						}
					}
				}
				if pc.RemoteDescription() != nil {
					for _, cd := range info.Candidates {
						if err := pc.AddICECandidate(cd.ToJSON()); err != nil {
							return stderr.Wrap(err)
						}
					}
				}
			}
		}
	}
}
