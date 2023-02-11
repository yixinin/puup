package backend

import (
	"encoding/json"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yixinin/puup/pnet"
	"github.com/yixinin/puup/stderr"
	"golang.org/x/crypto/ssh"
)

type SshServer struct {
	lis *pnet.Listener
}

type SshHeader struct {
	User string `json:"user,omitempty"`
	Pass string `json:"pass,omitempty"`
	Key  []byte `json:"key,omitempty"`
}

func NewSshClient(puup, name string) *SshServer {
	lis := pnet.NewListener(name, puup)
	return &SshServer{
		lis: lis,
	}
}

func (c *SshServer) Run() error {
	return c.loop()
}

func (c *SshServer) loop() error {
	for {
		conn, err := c.lis.Accept()
		if err != nil {
			return stderr.Wrap(err)
		}
		var req SshHeader
		var header = make([]byte, 1024)
		n, err := conn.Read(header)
		if err != nil {
			return stderr.Wrap(err)
		}
		logrus.Debugf("login with:%s", header[:n])
		err = json.Unmarshal(header[:n], &req)
		if err != nil {
			return stderr.Wrap(err)
		}
		go func() {
			err = Connect(req, conn)
			if err != nil {
				logrus.Error("ssh connection failed:%v", err)
			}
			logrus.Debugf("ssh session end")
		}()
	}
}

func Connect(req SshHeader, conn net.Conn) error {
	cfg := &ssh.ClientConfig{
		Timeout:         time.Second, //ssh 连接time out 时间一秒钟, 如果ssh验证错误 会在一秒内返回
		User:            req.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //这个可以, 但是不够安全
		//HostKeyCallback: hostKeyCallBackFunc(h.Host),
	}
	if req.Pass != "" {
		if req.Pass != "-nopass" {
			cfg.Auth = append(cfg.Auth, ssh.Password(req.Pass))
		}
	}
	if len(req.Key) != 0 {
		signKey, err := ssh.ParsePrivateKey(req.Key)
		if err != nil {
			return stderr.Wrap(err)
		}
		cfg.Auth = append(cfg.Auth, ssh.PublicKeys(signKey))
	}
	client, err := ssh.Dial("tcp", "127.0.0.1:22", cfg)
	if err != nil {
		return stderr.Wrap(err)
	}
	sess, err := client.NewSession()
	if err != nil {
		return stderr.Wrap(err)
	}
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // 关闭回显
		ssh.TTY_OP_ISPEED: 14400, // 设置传输速率
		ssh.TTY_OP_OSPEED: 14400,
		ssh.IGNCR:         1,
	}
	// 请求伪终端
	err = sess.RequestPty("xterm-256color", 32, 160, modes)
	if err != nil {
		return stderr.Wrap(err)
	}
	sess.Stdout = conn
	sess.Stderr = conn
	sess.Stdin = conn

	err = sess.Shell()
	if err != nil {
		return stderr.Wrap(err)
	}
	err = sess.Wait()
	if err != nil {
		return stderr.Wrap(err)
	}
	return nil
}
