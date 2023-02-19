package frontend

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/net"
)

func RunClient(ctx context.Context, name, puup string) {
	tp, err := net.NewTransport(puup, name)
	if err != nil {
		logrus.Error(err)
		return
	}
	hc := &http.Client{
		Transport: tp,
		Timeout:   30 * time.Second,
	}

	Get(hc, "http://localhost/hello")
	Get(hc, "http://localhost/hallo")
	tk := time.NewTicker(time.Second)
	defer tk.Stop()
	var i int
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			i++
			Get(hc, fmt.Sprintf("http://localhost/hello/%d", i))
			if i >= 100 {
				return
			}
		}
	}
}

func Get(hc *http.Client, url string) {
	resp, err := hc.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	fmt.Printf("%s, %v\n", data, err)
}
