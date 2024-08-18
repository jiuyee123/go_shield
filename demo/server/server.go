package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

type Server struct {
	listener net.Listener
	quit     chan struct{}
	port     int
}

func NewServer(port int) *Server {
	return &Server{
		port: port,
		quit: make(chan struct{}),
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}
	s.listener = listener
	log.Printf("Server listening on port %d", s.port)

	go s.acceptConnections()
	return nil
}

func (s *Server) Stop() {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *Server) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Printf("Error accepting connection: %v", err)
				continue
			}
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	log.Printf("New connection from %s", conn.RemoteAddr().String())
	defer conn.Close()
	for {
		var version uint8
		err := binary.Read(conn, binary.BigEndian, &version)
		if err != nil {
			log.Printf("Error reading version: %v", err)
			return
		}

		var msgType uint16
		err = binary.Read(conn, binary.BigEndian, &msgType)
		if err != nil {
			log.Printf("Error reading msgType: %v", err)
			return
		}

		// Read the remaining bytes as the message content
		var msgLength uint16
		err = binary.Read(conn, binary.BigEndian, &msgLength)
		if err != nil {
			log.Printf("Error reading message length: %v", err)
			return
		}

		if msgType == 1000 || msgType == 1001 { // 1000 是心跳消息
			// 心跳消息不需要处理

			messageData := make([]byte, msgLength)
			_, err = io.ReadFull(conn, messageData)
			if err != nil {
				log.Printf("Error reading message content: %v", err)
				return
			}

			// Deserialize the JSON message
			var message map[string]interface{}
			err = json.Unmarshal(messageData, &message)
			if err != nil {
				log.Printf("Error unmarshalling JSON: %v", err)
				return
			}
			log.Printf("Received heartbeat message %v", message)
		} else {

			messageData := make([]byte, msgLength)
			_, err = io.ReadFull(conn, messageData)
			if err != nil {
				log.Printf("Error reading message content: %v", err)
				return
			}

			// Deserialize the JSON message
			var message map[string]interface{}
			err = json.Unmarshal(messageData, &message)
			if err != nil {
				log.Printf("Error unmarshalling JSON: %v", err)
				return
			}
			log.Printf("Received request message %v", message)

			// Respond to the client
			response := map[string]interface{}{
				"message": fmt.Sprintf("Hello from server on port %d! The time is %s", s.port, time.Now().Format(time.RFC3339)),
			}
			responseData, err := json.Marshal(response)
			if err != nil {
				log.Printf("Error marshalling JSON response: %v", err)
				return
			}
			version = 1
			binary.Write(conn, binary.BigEndian, version)

			// Send response with message type
			err = binary.Write(conn, binary.BigEndian, msgType)
			if err != nil {
				log.Printf("Error writing message type: %v", err)
				return
			}

			responseLength := uint16(len(responseData))
			err = binary.Write(conn, binary.BigEndian, responseLength)
			if err != nil {
				log.Printf("Error writing message length: %v", err)
				return
			}

			_, err = conn.Write(responseData)
			if err != nil {
				log.Printf("Error writing response: %v", err)
				return
			}
		}
	}
}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Usage: go run server.go <port>")
	}

	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Invalid port number: %v", err)
	}

	server := NewServer(port)
	if err := server.Start(); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}

	// Wait for a signal to quit
	quit := make(chan os.Signal, 1)
	<-quit

	server.Stop()
	log.Println("Server stopped")
}
