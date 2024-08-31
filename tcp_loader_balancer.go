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
	listener        net.Listener
	lb              LoadBalancer
	config          Config
	clients         map[string]*Client
	mu              sync.Mutex
	done            chan struct{}
	bucket          *TokenBucket
	timeoutInterval int
}

func NewTCPLoadBalancer(config Config, lb LoadBalancer, bucket *TokenBucket) (*TCPLoadBalancer, error) {
	listener, err := net.Listen("tcp", config.ListenAddr)
	if err != nil {
		return nil, err
	}

	return &TCPLoadBalancer{
		listener:        listener,
		lb:              lb,
		config:          config,
		clients:         make(map[string]*Client),
		done:            make(chan struct{}),
		bucket:          bucket,
		timeoutInterval: 50,
	}, nil
}

type Client struct {
	conn       net.Conn
	lastActive time.Time
}

func (t *TCPLoadBalancer) Start() error {
	go t.startHeartbeatChecker()
	go t.acceptConnections()
	return nil
}

func (t *TCPLoadBalancer) Stop() {
	close(t.done)
	t.listener.Close()

	t.mu.Lock()
	for _, client := range t.clients {
		client.conn.Close()
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

			t.removeClient(conn.RemoteAddr().String())
			select {
			case <-t.done:
				return
			default:
				log.Printf("Error accepting connection: %v", err)
				return
			}
		}

		t.mu.Lock()
		if len(t.clients) >= t.config.MaxConnections {
			t.mu.Unlock()
			conn.Close()
			continue
		}
		t.clients[conn.RemoteAddr().String()] = &Client{
			conn:       conn,
			lastActive: time.Now(),
		}
		t.mu.Unlock()

		go t.handleConnection(conn)
	}
}

func (t *TCPLoadBalancer) startHeartbeatChecker() {
	ticker := time.NewTicker(60 * time.Second)

	for range ticker.C {
		t.mu.Lock()
		for addr, client := range t.clients {
			if time.Since(client.lastActive) > time.Duration(t.timeoutInterval*int(time.Second)) {
				fmt.Println("Closing inactive connection:", addr)
				client.conn.Close()
				delete(t.clients, addr)
			}
		}
		t.mu.Unlock()
	}
}

func (t *TCPLoadBalancer) removeClient(addr string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.clients, addr)
	fmt.Println("Client removed:", addr)
}

func (t *TCPLoadBalancer) handleConnection(clientConn net.Conn) {
	defer func() {
		clientConn.Close()
		t.mu.Lock()
		delete(t.clients, clientConn.RemoteAddr().String())
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

	for {
		// Step 1: 读取客户端请求的version信息
		var version uint8
		err = binary.Read(clientConn, binary.BigEndian, &version)
		if err != nil {
			if err == io.EOF {
				log.Printf("Client closed the connection")
				break
			}
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

		if client, ok := t.clients[clientConn.RemoteAddr().String()]; ok {
			t.mu.Lock()
			client.lastActive = time.Now()
			t.mu.Unlock()
		}

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
			continue // 跳过当前请求，但继续处理后续请求
		}

		// 转发请求到后端
		err = binary.Write(backendConn, binary.BigEndian, version)
		if err != nil {
			fmt.Println("Failed to forward version to backend:", err)
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

		// 从后端读取响应并转发回客户端
		for {
			responseBuffer := make([]byte, 4096)
			n, err := backendConn.Read(responseBuffer)
			if err != nil {
				if err == io.EOF {
					break // 连接已经关闭，停止读取
				}
				fmt.Println("Failed to read response from backend:", err)
				return
			}

			// 将读取到的数据写回客户端
			_, err = clientConn.Write(responseBuffer[:n])
			if err != nil {
				fmt.Println("Failed to send response to client:", err)
				return
			}

			// 如果读取到的数据小于 buffer 大小，说明数据读取完毕
			if n < len(responseBuffer) {
				break
			}
		}
	}
}
