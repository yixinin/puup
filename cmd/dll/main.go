package main

import (
	"net/http"
	"os"

	"github.com/yixinin/puup/frontend"
	pnet "github.com/yixinin/puup/net"
)

var sigAddr, serverName string

var file *frontend.FileClient
var hc *http.Client

func main() {
	tp, err := pnet.NewTransport(sigAddr, serverName)
	if err != nil {
		panic(err)
	}
	hc = &http.Client{
		Transport: tp,
	}
	file = frontend.NewFileClient(cfg)
}

func Uploadfile(filename string) string {
	fs, err := os.Open(filename)
	if err != nil {
		return err.Error()
	}
	file.CopyFile(file)
}
