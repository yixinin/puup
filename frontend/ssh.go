package frontend

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/net"
	"github.com/yixinin/puup/stderr"
	"golang.org/x/term"
)

type SshHeader struct {
	User string `json:"user,omitempty"`
	Pass string `json:"pass,omitempty"`
	Key  []byte `json:"key,omitempty"`
}

type SshClient struct {
	sigAddr    string
	serverName string
}

func NewSshClient() *SshClient {
	return &SshClient{}
}

func (c *SshClient) Run(user, name, pass string) error {
	conn, err := net.DialSsh(c.sigAddr, c.serverName)
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
	_, err = conn.Write(data)
	if err != nil {
		return err
	}

	var read = func() error {
		_, err := io.Copy(os.Stdout, conn)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		return nil
	}
	var wirte = func() error {
		_, err := io.Copy(conn, os.Stdin)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		return nil
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		err := read()
		if err != nil {
			logrus.Errorf("read error:%v", err)
		}
	}()
	go func() {
		defer wg.Done()
		err := wirte()
		if err != nil {
			logrus.Errorf("write error:%v", err)
		}
	}()
	fileDescriptor := int(os.Stdin.Fd())
	if term.IsTerminal(fileDescriptor) {
		originalState, err := term.MakeRaw(fileDescriptor)
		if err != nil {
			return stderr.Wrap(err)
		}
		defer term.Restore(fileDescriptor, originalState)
	}
	wg.Wait()
	return nil
}

func GetArgsUserPass() (user, name, pass string, err error) {
	ss := flag.Args()
	fmt.Println(ss)
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
