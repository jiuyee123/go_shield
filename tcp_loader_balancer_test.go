package goshield

import (
	"net"
	"testing"
	"time"
)

// Mocking the net.Listener interface
type MockListener struct {
	acceptErr error
}

func (ml *MockListener) Accept() (net.Conn, error) {
	if ml.acceptErr != nil {
		return nil, ml.acceptErr
	}
	return &MockConn{}, nil
}

func (ml *MockListener) Close() error {
	return nil
}

func (ml *MockListener) Addr() net.Addr {
	return nil
}

// Mocking the net.Conn interface
type MockConn struct{}

func (mc *MockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (mc *MockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (mc *MockConn) Close() error                       { return nil }
func (mc *MockConn) LocalAddr() net.Addr                { return nil }
func (mc *MockConn) RemoteAddr() net.Addr               { return nil }
func (mc *MockConn) SetDeadline(t time.Time) error      { return nil }
func (mc *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (mc *MockConn) SetWriteDeadline(t time.Time) error { return nil }

type MockLoadBalancer struct {
	backends []Backend
	index    int
}

func (lb MockLoadBalancer) AddBackend(backend Backend) {

}

func (lb MockLoadBalancer) RemoveBackend(backend *Backend) {

}

func (lb MockLoadBalancer) NextBackend() (*Backend, error) {
	return nil, nil
}

func (lb MockLoadBalancer) HealthCheck() {

}

// Test NewTCPLoadBalancer function
func TestNewTCPLoadBalancer(t *testing.T) {
	mockListener := &MockListener{}
	config := Config{ListenAddr: "localhost:8080"}
	lb := MockLoadBalancer{}
	bucket := DefaultTokenBucket()

	tcpLB, err := NewTCPLoadBalancer(config, lb, bucket)
	tcpLB.listener = mockListener
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if tcpLB.listener == nil {
		t.Fatalf("Expected listener to be initialized")
	}

	if tcpLB.lb == nil {
		t.Fatalf("Expected load balancer to be initialized")
	}
}

// Test Start function
func TestTCPLoadBalancerStart(t *testing.T) {
	mockListener := &MockListener{}
	config := Config{ListenAddr: "localhost:8080"}
	lb := NewRoundRobinLoadBalancer()
	bucket := DefaultTokenBucket()

	tcpLB, _ := NewTCPLoadBalancer(config, lb, bucket)
	tcpLB.listener = mockListener

	err := tcpLB.Start()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// Test Stop function
func TestTCPLoadBalancerStop(t *testing.T) {
	mockListener := &MockListener{}
	config := Config{ListenAddr: "localhost:8080"}

	lb := NewRoundRobinLoadBalancer()
	bucket := DefaultTokenBucket()

	tcpLB, _ := NewTCPLoadBalancer(config, lb, bucket)
	tcpLB.listener = mockListener

	tcpLB.Start()
	tcpLB.Stop()

	if len(tcpLB.conns) != 0 {
		t.Fatalf("Expected all connections to be closed")
	}
}
