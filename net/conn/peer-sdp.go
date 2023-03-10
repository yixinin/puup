package conn

import (
	"context"
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
	logrus.Debugf("send %s sdp", offer.Type)
	if err := p.sigCli.SendSdp(ctx, p.clientId, offer); err != nil {
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
	if err := p.sigCli.SendSdp(ctx, p.clientId, answer); err != nil {
		return stderr.Wrap(err)
	}
	return nil
}

func (p *Peer) WaitAnswer(ctx context.Context) error {
	defer p.sigCli.Offline(context.Background(), p.clientId)
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
		case sdp := <-p.sigCli.RemoteSdp(p.clientId):
			logrus.Debugf("recv %s sdp, %s < %s", sdp.Type, p.pc.ConnectionState(), webrtc.PeerConnectionStateConnected)
			if p.pc.ConnectionState() < webrtc.PeerConnectionStateConnected {
				err := p.pc.SetRemoteDescription(sdp)
				if err != nil {
					return stderr.Wrap(err)
				}
			}
		case ice := <-p.sigCli.RemoteIceCandidates(p.clientId):
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
	defer p.sigCli.Offline(context.Background(), p.clientId)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.connected:
			return nil
		case sdp := <-p.sigCli.RemoteSdp(p.clientId):
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
		case ice := <-p.sigCli.RemoteIceCandidates(p.clientId):
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
