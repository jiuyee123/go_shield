package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	goshield "github.com/jiuyee123/go/shield"
)

func main() {
	config := goshield.Config{
		ListenAddr:          ":8080",
		HealthCheckInterval: 10 * time.Second,
		MaxConnections:      1000,
		SessionTimeout:      30 * time.Minute,
	}

	lb := goshield.NewRoundRobinLoadBalancer()

	// 添加后端服务器
	lb.AddBackend(goshield.Backend{Address: "localhost:8081", Weight: 1})
	lb.AddBackend(goshield.Backend{Address: "localhost:8082", Weight: 1})
	lb.AddBackend(goshield.Backend{Address: "localhost:8083", Weight: 1})

	proxy, err := goshield.NewTCPLoadBalancer(config, lb, goshield.DefaultTokenBucket())
	if err != nil {
		log.Fatalf("Failed to create TCP load balancer: %v", err)
	}

	server := goshield.NewServer(config)
	server.Proxy = proxy

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Server stopped: %v", err)
			cancel()
		}
	}()

	// 开始健康检查
	go func() {
		ticker := time.NewTicker(config.HealthCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				server.Proxy.HealthCheck()
			case <-ctx.Done():
				return
			}
		}
	}()

	// 优雅关闭
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	cancel()
	server.Stop()
	log.Println("Server stopped")
}
