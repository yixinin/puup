package proto

import (
	"fmt"
	"strings"
)

const (
	Ssh   ChannelType = "ssh"
	Proxy ChannelType = "proxy"
	File  ChannelType = "file"
	Web   ChannelType = "http"
)

type ChannelType string

func FileLabel(index int) string {
	return fmt.Sprintf("%s.%d", File, index)
}

func HttpLabel(index int) string {
	return fmt.Sprintf("%s.%d", Web, index)
}

func ProxyLabel(port uint16) string {
	return fmt.Sprintf("%s.%d", Proxy, port)
}

func SshLabel() string {
	return string(Ssh)
}

func GetChannelType(label string) ChannelType {
	return ChannelType(strings.Split(label, ".")[0])
}
