package goshield

import (
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type TCPLoadBalancer struct {
	listener net.Listener
	lb       LoadBalancer
	config   Config
	conns    map[net.Conn]struct{}
	mu       sync.Mutex
	done     chan struct{}
}

func NewTCPLoadBalancer(config Config, lb LoadBalancer) (*TCPLoadBalancer, error) {
	listener, err := net.Listen("tcp", config.ListenAddr)
	if err != nil {
		return nil, err
	}

	return &TCPLoadBalancer{
		listener: listener,
		lb:       lb,
		config:   config,
		conns:    make(map[net.Conn]struct{}),
		done:     make(chan struct{}),
	}, nil
}

func (t *TCPLoadBalancer) Start() error {
	go t.acceptConnections()
	return nil
}

func (t *TCPLoadBalancer) Stop() {
	close(t.done)
	t.listener.Close()

	t.mu.Lock()
	for conn := range t.conns {
		conn.Close()
	}
	t.mu.Unlock()
}

func (t *TCPLoadBalancer) HealthCheck() {
	t.lb.HealthCheck()
}

func (t *TCPLoadBalancer) acceptConnections() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.done:
				return
			default:
				log.Printf("Error accepting connection: %v", err)
				continue
			}
		}

		t.mu.Lock()
		if len(t.conns) >= t.config.MaxConnections {
			t.mu.Unlock()
			conn.Close()
			continue
		}
		t.conns[conn] = struct{}{}
		t.mu.Unlock()

		go t.handleConnection(conn)
	}
}

func (t *TCPLoadBalancer) handleConnection(clientConn net.Conn) {
	defer func() {
		clientConn.Close()
		t.mu.Lock()
		delete(t.conns, clientConn)
		t.mu.Unlock()
	}()

	backend, err := t.lb.NextBackend()
	if err != nil {
		log.Printf("Error getting backend: %v", err)
		return
	}

	backendConn, err := net.DialTimeout("tcp", backend.Address, 5*time.Second)
	if err != nil {
		log.Printf("Error connecting to backend %s: %v", backend.Address, err)
		return
	}
	defer backendConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		io.Copy(backendConn, clientConn)
		wg.Done()
	}()

	go func() {
		io.Copy(clientConn, backendConn)
		wg.Done()
	}()

	wg.Wait()
}
