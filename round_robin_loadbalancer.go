package goshield

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/jiuyee123/go/shield/demo"
)

type RoundRobinLoadBalancer struct {
	backends []*Backend
	mu       sync.RWMutex
	index    int
}

func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{
		backends: make([]*Backend, 0),
	}
}

func (lb *RoundRobinLoadBalancer) AddBackend(backend Backend) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.backends = append(lb.backends, &backend)
}

func (lb *RoundRobinLoadBalancer) RemoveBackend(backend *Backend) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	for i, backend := range lb.backends {
		if backend.Address == backend.Address {
			lb.backends = append(lb.backends[:i], lb.backends[i+1:]...)
			break
		}
	}
}

func (lb *RoundRobinLoadBalancer) NextBackend() (*Backend, error) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(lb.backends) == 0 {
		return nil, errors.New("no backends available")
	}

	backend := lb.backends[lb.index]
	lb.index = (lb.index + 1) % len(lb.backends)
	return backend, nil
}

func (lb *RoundRobinLoadBalancer) HealthCheck() {
	lb.mu.RLock()
	backends := make([]*Backend, len(lb.backends))
	copy(backends, lb.backends)
	lb.mu.RUnlock()

	for _, backend := range backends {
		conn, err := net.DialTimeout("tcp", backend.Address, 5*time.Second)
		if err != nil {
			lb.RemoveBackend(backend)
		} else {
			sendBeatHeart(conn)
			conn.Close()
		}
	}
}

func sendBeatHeart(conn net.Conn) {
	var version uint8 = 1
	binary.Write(conn, binary.BigEndian, version)

	// Send response with message type
	err := binary.Write(conn, binary.BigEndian, demo.Func_HeartBeat_Server)
	if err != nil {
		log.Printf("Error writing message type: %v", err)
		return
	}

	request := map[string]interface{}{
		"srvTime": time.Now().Format("20060102150405"),
	}
	requestData, err := json.Marshal(request)
	if err != nil {
		log.Printf("Error marshalling response: %v", err)
		return
	}
	reqLength := uint16(len(requestData))
	err = binary.Write(conn, binary.BigEndian, reqLength)
	if err != nil {
		log.Printf("Error writing message length: %v", err)
		return
	}

	_, err = conn.Write(requestData)
	if err != nil {
		log.Printf("Error writing response: %v", err)
		return
	}
}
