package main

import (
	"errors"
	"flag"
	"strings"

	"github.com/sirupsen/logrus"
)

type ScpMode string

const (
	Push ScpMode = "push"
	Pull ScpMode = "pull"
)

func main() {
	flag.Parse()
	local, remote, name, mode, err := GetFile()
	if err != nil {
		logrus.Errorf("parse args error:%v", err)
		return
	}

}

func GetFile() (localFilename, remoteFilename, name string, mode ScpMode, err error) {
	args := flag.Args()

	// scp file.ext pi:app/file.ext
	// scp pi:app/file.ext file.ext

	ss := strings.Split(args[0], ":")
	switch len(ss) {
	case 1:
		localFilename = ss[0]
		mode = Push
	case 2:
		name = ss[0]
		remoteFilename = ss[1]
		mode = Pull
	}

	ss = strings.Split(args[1], ":")
	switch len(ss) {
	case 1:
		localFilename = ss[0]
		if mode != Pull {
			err = errors.New("scp push args error")
			return
		}
		return
	case 2:
		name = ss[0]
		remoteFilename = ss[1]
		if mode != Push {
			err = errors.New("scp pull args error")
			return
		}
		return
	}

	err = errors.New("scp args error")
	return
}
