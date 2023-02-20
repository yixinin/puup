package config

import (
	"os"

	"github.com/yixinin/puup/net/conn"
	"gopkg.in/yaml.v2"
)

type ProxyPort struct {
	Local  uint16 `yaml:"local"`
	Remote uint16 `yaml:"remote,omitempty"`
}
type Config struct {
	Type       conn.PeerType `yaml:"type"`
	ServerName string        `yaml:"server_name"`
	SigAddr    string        `yaml:"sig_addr"`
	Proxy      []ProxyPort   `yaml:"proxy"`
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
