package goshield

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
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
	bucket   *TokenBucket
}

func NewTCPLoadBalancer(config Config, lb LoadBalancer, bucket *TokenBucket) (*TCPLoadBalancer, error) {
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
		bucket:   bucket,
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

	// Step 1: 读取客户端请求的version信息
	var version uint8
	err = binary.Read(clientConn, binary.BigEndian, &version)
	if err != nil {
		fmt.Println("Failed to read version:", err)
		return
	}

	// Step 2: 读取客户端请求的消息类型和消息内容
	var msgType uint16
	err = binary.Read(clientConn, binary.BigEndian, &msgType)
	if err != nil {
		fmt.Println("Failed to read message type:", err)
		return
	}
	log.Printf("receive msgType: %d", msgType)
	// 限流逻辑处理
	if !t.bucket.AllowRequest() {
		log.Printf("Request rate limit exceeded")

		response := map[string]interface{}{
			"message": "error: request rate limit exceeded",
		}
		responseData, err := json.Marshal(response)
		if err != nil {
			log.Printf("Error marshalling JSON response: %v", err)
			return
		}

		binary.Write(clientConn, binary.BigEndian, version)
		binary.Write(clientConn, binary.BigEndian, msgType)
		binary.Write(clientConn, binary.BigEndian, uint16(len(responseData)))
		clientConn.Write(responseData)
		return
	}

	var msgLength uint16
	err = binary.Read(clientConn, binary.BigEndian, &msgLength)
	if err != nil {
		fmt.Println("Failed to read message length:", err)
		return
	}

	messageData := make([]byte, msgLength)
	_, err = io.ReadFull(clientConn, messageData)
	if err != nil {
		fmt.Println("Failed to read message data:", err)
		return
	}

	// Step 3: 解析消息内容
	var message map[string]interface{}
	err = json.Unmarshal(messageData, &message)
	if err != nil {
		fmt.Println("Failed to unmarshal JSON:", err)
		return
	}

	// 转发请求
	err = binary.Write(backendConn, binary.BigEndian, version)
	if err != nil {
		fmt.Println("Failed to forward message type to backend:", err)
		return
	}

	err = binary.Write(backendConn, binary.BigEndian, msgType)
	if err != nil {
		fmt.Println("Failed to forward message type to backend:", err)
		return
	}

	err = binary.Write(backendConn, binary.BigEndian, msgLength)
	if err != nil {
		fmt.Println("Failed to forward message length to backend:", err)
		return
	}

	_, err = backendConn.Write(messageData)
	if err != nil {
		fmt.Println("Failed to forward message data to backend:", err)
		return
	}

	// 转发服务器的响应回客户端
	go io.Copy(clientConn, backendConn)
	io.Copy(backendConn, clientConn)

	// 转发请求
	// var wg sync.WaitGroup
	// wg.Add(2)

	// go func() {
	// 	io.Copy(backendConn, clientConn)
	// 	wg.Done()
	// }()

	// go func() {
	// 	io.Copy(clientConn, backendConn)
	// 	wg.Done()
	// }()

	// wg.Wait()
}
