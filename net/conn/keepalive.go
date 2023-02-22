package conn

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/stderr"
)

func (p *Peer) loopCommand(ctx context.Context, dc *webrtc.DataChannel) error {
	var ch = make(chan error)
	defer close(ch)

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		var cmd DataChannelCommand
		err := json.Unmarshal(msg.Data, &cmd)
		if err != nil {
			select {
			case ch <- err:
			case <-ch:
			}
		}
		p.cmdChan <- cmd
	})
	return <-ch
}

func (p *Peer) loopKeepalive(ctx context.Context, dc *webrtc.DataChannel) error {
	if dc == nil {
		return stderr.New("data channel is nil ")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {

	})
	var open = make(chan struct{})
	dc.OnOpen(func() {
		close(open)
	})

	dc.OnClose(func() {
		cancel()
	})
	// wait data open
	t := time.NewTimer(time.Minute)
	defer t.Stop()
	select {
	case <-t.C:
		return stderr.Wrap(context.DeadlineExceeded)
	case <-open:
	}
	t.Stop()

	var tk = time.NewTicker(30 * time.Second)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.close:
			return nil
		case <-tk.C:
			err := dc.Send([]byte{':', ':'})
			if err != nil {
				logrus.Errorf("send keep alive error:%v", err)
			}
		}
	}
}
