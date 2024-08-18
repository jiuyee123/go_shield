package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/jiuyee123/go/shield/demo"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Usage: go run client.go <host> <port>")
	}

	addr := fmt.Sprintf("%s:%s", os.Args[1], os.Args[2])
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("Error connecting to server: %v", err)
	}
	defer conn.Close()

	go func() {
		// 模拟发送消息
		for {
			// 发送hello请求
			sendRequest(conn, demo.Func_Hello_Client, map[string]interface{}{
				"message": "hello",
				"time":    time.Now(),
			})
			time.Sleep(20 * time.Microsecond)
		}
	}()

	// 启动心跳包发送
	go startHeartbeat(conn)

	// 等待接收服务器的响应
	for {
		handleServerResponse(conn)
	}
}

var version uint8 = 1

// 发送请求
func sendRequest(conn net.Conn, msgType uint16, data map[string]interface{}) {
	// 将消息序列化为JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal JSON: %v", err)
		return
	}

	// 发送版本信息
	err = binary.Write(conn, binary.BigEndian, version)
	if err != nil {
		log.Printf("Failed to write version type: %v", err)
		return
	}

	// 发送消息类型
	err = binary.Write(conn, binary.BigEndian, msgType)
	if err != nil {
		log.Printf("Failed to write message type: %v", err)
		return
	}

	// 发送消息长度
	msgLength := uint16(len(jsonData))
	err = binary.Write(conn, binary.BigEndian, msgLength)
	if err != nil {
		log.Printf("Failed to write message length: %v", err)
		return
	}

	// 发送消息内容
	_, err = conn.Write(jsonData)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
		return
	}

	log.Printf("Sent message type %d with content: %v", msgType, data)
}

// 处理服务器响应
func handleServerResponse(conn net.Conn) {

	var version uint8
	err := binary.Read(conn, binary.BigEndian, &version)
	if err != nil {
		log.Printf("Error reading version: %v", err)
		return
	}

	// 读取消息类型
	var msgType uint16
	err = binary.Read(conn, binary.BigEndian, &msgType)
	if err != nil {
		log.Printf("Failed to read message type: %v", err)
		return
	}

	// 读取消息长度
	var msgLength uint16
	err = binary.Read(conn, binary.BigEndian, &msgLength)
	if err != nil {
		log.Printf("Failed to read message length: %v", err)
		return
	}

	// 读取消息内容
	messageData := make([]byte, msgLength)
	_, err = conn.Read(messageData)
	if err != nil {
		log.Printf("Failed to read message content: %v", err)
		return
	}

	// 反序列化JSON
	var message map[string]interface{}
	err = json.Unmarshal(messageData, &message)
	if err != nil {
		log.Printf("Failed to unmarshal JSON: %v", err)
		return
	}

	log.Printf("Received message type %d with content: %v", msgType, message)
}

// 启动心跳包发送
func startHeartbeat(conn net.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sendRequest(conn, demo.Func_HeartBeat_Client, map[string]interface{}{
				"heartbeat": "ping",
			})
		}
	}
}
