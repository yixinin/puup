package frontend

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/net"
	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/stderr"
	"golang.org/x/term"
)

type SshHeader struct {
	User string `json:"user,omitempty"`
	Pass string `json:"pass,omitempty"`
	Key  []byte `json:"key,omitempty"`
}

type SshClient struct {
	sigAddr string
}

func NewSshClient(sigAddr string) *SshClient {
	return &SshClient{
		sigAddr: sigAddr,
	}
}

func (c *SshClient) Run(user, name, pass string) error {
	rconn, err := net.Dial(c.sigAddr, name, conn.Ssh)
	if err != nil {
		return err
	}
	if pass == "" {
		pass, err = GetUserPass()
		if err != nil {
			return err
		}
	}
	var req = SshHeader{
		User: user,
		Pass: pass,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	logrus.Debugf("login with:%s", data)
	_, err = rconn.Write(data)
	if err != nil {
		return err
	}

	fileDescriptor := int(os.Stdin.Fd())
	if term.IsTerminal(fileDescriptor) {
		originalState, err := term.MakeRaw(fileDescriptor)
		if err != nil {
			return stderr.Wrap(err)
		}
		defer term.Restore(fileDescriptor, originalState)
	}

	return conn.GoCopy(os.Stdin, rconn)
}

func GetArgsUserPass() (user, name, pass string, err error) {
	ss := flag.Args()
	var sss = make([]string, 0, len(ss))
	for _, v := range ss {
		v = strings.TrimSpace(v)
		if v != "" {
			sss = append(sss, v)
		}
	}

	var username string
	switch len(sss) {
	case 0:
		err = errors.New("no user")
		return
	case 1:
		username = sss[0]
	case 2:
		username = sss[0]
		pass = sss[1]
	}

	ss = strings.Split(username, "@")
	switch len(ss) {
	case 1:
		user = ss[0]
		name = "pi"
	case 2:
		user = ss[0]
		name = ss[1]
	}

	if pass != "" {
		return
	}
	return
}

func GetUserPass() (pass string, err error) {
	fmt.Println("Password:")
	buf, _, err := bufio.NewReader(os.Stdin).ReadLine()
	if err != nil {
		return
	}
	pass = string(buf)
	return
}
