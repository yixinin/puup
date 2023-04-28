package conn

import (
	"context"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/proto"
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
	logrus.Debugf("send %s sdp", offer.Type)
	var packet = proto.Packet{
		From: proto.Client{
			ClientId: p.sig.Id(),
			PeerId:   p.Id,
		},
		To: proto.Client{
			PeerId:   p.RemoteId,
			ClientId: p.RemoteClientId,
		},
		Sdp: &offer,
	}
	if err := p.sig.SendPacket(ctx, packet); err != nil {
		return stderr.Wrap(err)
	}
	return nil
}

func (p *Peer) SendAnswer(ctx context.Context) error {
	answer, err := p.pc.CreateAnswer(nil)
	if err != nil {
		return err
	}
	logrus.Debugf("send %s sdp", answer.Type)
	if err := p.pc.SetLocalDescription(answer); err != nil {
		return stderr.Wrap(err)
	}
	var packet = proto.Packet{
		From: proto.Client{
			ClientId: p.sig.Id(),
			PeerId:   p.Id,
		},
		To: proto.Client{
			PeerId:   p.RemoteId,
			ClientId: p.RemoteClientId,
		},
		Sdp: &answer,
	}
	if err := p.sig.SendPacket(ctx, packet); err != nil {
		return stderr.Wrap(err)
	}
	return nil
}

func (p *Peer) WaitAnswer(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.connected:
			return nil
		case sdp, ok := <-p.sig.RemoteSdp(p.RemoteId):
			if !ok {
				return nil
			}
			logrus.Debugf("recv %s sdp, %s < %s", sdp.Type, p.pc.ConnectionState(), webrtc.PeerConnectionStateConnected)
			if p.pc.ConnectionState() < webrtc.PeerConnectionStateConnected {
				err := p.pc.SetRemoteDescription(sdp)
				if err != nil {
					return stderr.Wrap(err)
				}
			}
		case ice := <-p.sig.RemoteIceCandidates(p.RemoteId):
			logrus.Debugf("recv ice, %s < %s", p.pc.ICEConnectionState(), webrtc.ICEConnectionStateConnected)
			if p.pc.ICEConnectionState() < webrtc.ICEConnectionStateConnected {
				err := p.pc.AddICECandidate(ice.ToJSON())
				if err != nil {
					return stderr.Wrap(err)
				}
			}
		}
	}
}
func (p *Peer) PollOffer(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.connected:
			return nil
		case sdp, ok := <-p.sig.RemoteSdp(p.RemoteId):
			if !ok {
				return nil
			}
			logrus.Debugf("recv %s sdp, %s < %s", sdp.Type, p.pc.ConnectionState(), webrtc.PeerConnectionStateConnected)
			if p.pc.ConnectionState() < webrtc.PeerConnectionStateConnected {
				err := p.pc.SetRemoteDescription(sdp)
				if err != nil {
					return stderr.Wrap(err)
				}
				if err := p.SendAnswer(ctx); err != nil {
					return err
				}
			}
		case ice := <-p.sig.RemoteIceCandidates(p.RemoteId):
			logrus.Debugf("recv ice, %s < %s", p.pc.ICEConnectionState(), webrtc.ICEConnectionStateConnected)
			if p.pc.ICEConnectionState() < webrtc.ICEConnectionStateConnected {
				err := p.pc.AddICECandidate(ice.ToJSON())
				if err != nil {
					return stderr.Wrap(err)
				}
			}
		}
	}
}
