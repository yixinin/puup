package config

import (
	"os"

	"github.com/yixinin/puup/net/conn"
	"github.com/yixinin/puup/proxy"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Type       conn.PeerType     `yaml:"type"`
	ServerName string            `yaml:"server_name"`
	SigAddr    string            `yaml:"sig_addr"`
	Proxy      []proxy.ProxyPort `yaml:"proxy"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var c = new(Config)
	err = yaml.Unmarshal(data, c)
	return c, err
}
