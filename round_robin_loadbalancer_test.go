package goshield

import (
	"testing"
)

func TestNewRoundRobinLoadBalancer(t *testing.T) {
	loadBalancer := NewRoundRobinLoadBalancer()
	if loadBalancer == nil {
		t.Fatal("Failed to create round robin load balancer")
	}

	if loadBalancer.backends == nil {
		t.Fatal("Failed to create backends")
	}

	if len(loadBalancer.backends) != 0 {
		t.Fatal("Failed to create backends")
	}
}

func TestAddBackend(t *testing.T) {
	loadBalancer := NewRoundRobinLoadBalancer()
	loadBalancer.AddBackend(Backend{})
	if len(loadBalancer.backends) != 1 {
		t.Fatal("Failed to add backend")
	}

	loadBalancer.AddBackend(Backend{Address: "localhost"})
	if len(loadBalancer.backends) != 2 {
		t.Fatal("Failed to add backend")
	}

	if loadBalancer.backends[1].Address != "localhost" {
		t.Fatal("Wrong Add Bacnkend.")
	}

}

func TestRemoveBackend(t *testing.T) {
	loadBalancer := NewRoundRobinLoadBalancer()
	loadBalancer.AddBackend(Backend{Address: "localhost"})
	loadBalancer.RemoveBackend(&Backend{Address: "localhost"})
	if len(loadBalancer.backends) != 0 {
		t.Fatal("Failed to remove backend")
	}
}

func TestNextBackend(t *testing.T) {
	loadBalancer := NewRoundRobinLoadBalancer()
	loadBalancer.AddBackend(Backend{Address: "localhost1"})
	b, err := loadBalancer.NextBackend()
	if err != nil {
		t.Fatal("Failed to get next backend")
	}

	if b.Address != "localhost1" {
		t.Fatal("Wrong Next Backend.")
	}
	loadBalancer.AddBackend(Backend{Address: "localhost2"})
	loadBalancer.AddBackend(Backend{Address: "localhost3"})
	b, err = loadBalancer.NextBackend()
	if err != nil {
		t.Fatal("Failed to get next backend")
	}

	if b.Address != "localhost1" {
		t.Fatal("Wrong Next Backend.")
	}

	b, err = loadBalancer.NextBackend()
	if err != nil {
		t.Fatal("Failed to get next backend")
	}

	if b.Address != "localhost2" {
		t.Fatal("Wrong Next Backend.")
	}
}
