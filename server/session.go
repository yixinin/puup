package server

type Session struct {
	ClusterName string
	Backends    map[string]*Client
	Frontends   map[string]*Client
}
