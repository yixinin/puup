package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type ProxyPort struct {
	Local  uint16 `yaml:"local"`
	Remote uint16 `yaml:"remote,omitempty"`
}
type ProxyBack struct {
	Addr  string   `yaml:"addr"`
	Ports []uint16 `yaml:"ports"`
}

type Config struct {
	Type       string      `yaml:"type"`
	ServerName string      `yaml:"server_name"`
	SigAddr    string      `yaml:"sig_addr"`
	ProxyBack  *ProxyBack  `yaml:"proxy_back"`
	ProxyFront []ProxyPort `yaml:"proxy_front"`
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
