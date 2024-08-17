package goshield

import (
	"net"
	"time"
)

// Backend 表示一个后端服务器
type Backend struct {
	Address string
	Weight  int
}

// LoadBalancer 表示一个负载均衡器
type LoadBalancer interface {
	AddBackend(backend Backend)
	RemoveBackend(backend *Backend)
	NextBackend() (*Backend, error)
	HealthCheck()
}

// TCPProxy 表示一个TCP代理
type TCPProxy interface {
	Start() error
	Stop()
	HealthCheck()
}

// Config 存储负载均衡器的配置
type Config struct {
	ListenAddr          string
	HealthCheckInterval time.Duration
	MaxConnections      int
	SessionTimeout      time.Duration
}

// Server 表示一个服务器
type Server struct {
	Config Config
	Proxy  TCPProxy
	Conns  map[net.Conn]struct{}
}

func NewServer(config Config) *Server {
	return &Server{
		Config: config,
		Conns:  make(map[net.Conn]struct{}),
	}
}

func (s *Server) Start() error {
	return s.Proxy.Start()
}

func (s *Server) Stop() {
	s.Proxy.Stop()
}
