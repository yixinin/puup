package net

import (
	"bufio"
	"errors"
	"io"
	"net/http"

	"github.com/yixinin/puup/stderr"
)

type Transport struct {
	client   *PeersClient
	sigAddr  string
	backName string
}

func NewTransport(sigAddr, name string) (http.RoundTripper, error) {
	var wt = &Transport{
		sigAddr:  sigAddr,
		backName: name,
	}
	wt.client = NewPeersClient()
	return wt, nil
}

func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	c, err := t.client.DialWeb(t.sigAddr, t.backName)
	if err != nil {
		return nil, err
	}
	conn, ok := c.(*Conn)
	if !ok {
		return nil, stderr.Wrap(errors.New("unknown conn"))
	}
	defer func() {
		if err != nil {
			conn.Release()
		}
	}()

	err = req.Write(conn)
	if err != nil {
		return nil, err
	}
	rd := bufio.NewReader(conn)
	resp, err = http.ReadResponse(rd, req)
	if err != nil {
		return nil, err
	}
	resp.Body = &RespConnCloser{
		ReadCloser: resp.Body,
		Release:    conn.Release,
	}
	return resp, err
}

type RespConnCloser struct {
	Release func()
	io.ReadCloser
}

func (r *RespConnCloser) Close() error {
	err := r.ReadCloser.Close()
	if r.Release != nil {
		r.Release()
	}
	return err
}
