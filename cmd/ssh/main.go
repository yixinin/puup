package main

import (
	"flag"

	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/frontend"
)

const (
	sigAddr = ""
)

func main() {
	flag.Parse()
	// logrus.SetLevel(logrus.DebugLevel)
	var c = frontend.NewSshClient("http://114.115.218.1:8080")
	user, name, pass, err := frontend.GetArgsUserPass()
	if err != nil {
		logrus.Errorf("get args error:%v", err)
		return
	}

	err = c.Run(user, name, pass)
	if err != nil {
		logrus.Errorf("run error:%v", err)
	}
}
