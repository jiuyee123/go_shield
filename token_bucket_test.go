package goshield

import (
	"sync"
	"testing"
	"time"
)

func TestNewTokenBucket(t *testing.T) {
	rate := 10
	capacity := 20
	tb := NewTokenBucket(rate, capacity)

	if tb.rate != rate {
		t.Fatalf("Expected rate to be %d, got %d", rate, tb.rate)
	}
	if tb.capacity != capacity {
		t.Fatalf("Expected capacity to be %d, got %d", capacity, tb.capacity)
	}

	if tb.tokens != capacity {
		t.Fatalf("Expected tokens to be %d, got %d", capacity, tb.tokens)
	}
}

func TestDefaultTokenBucket(t *testing.T) {
	tb := DefaultTokenBucket()

	if tb.rate != 5 {
		t.Fatalf("Expected default rate to be 5, got %d", tb.rate)
	}

	if tb.capacity != 10 {
		t.Fatalf("Expected default capacity to be 10, got %d", tb.capacity)
	}

	if tb.tokens != 10 {
		t.Fatalf("Expected default tokens to be 10, got %d", tb.tokens)
	}
}

func TestAllowRequest(t *testing.T) {
	tb := NewTokenBucket(10, 10)

	// Test allowing a single request
	if !tb.AllowRequest() {
		t.Fatalf("Expected AllowRequest to return true")
	}

	// Test draining the bucket
	for i := 0; i < 9; i++ {
		tb.AllowRequest()
	}

	if tb.AllowRequest() {
		t.Fatalf("Expected AllowRequest to return false after draining the bucket")
	}

	// Test token refill after some time
	time.Sleep(200 * time.Millisecond) // Wait for 0.2 seconds (should add 2 tokens if rate is 10/s)
	if !tb.AllowRequest() {
		t.Fatalf("Expected AllowRequest to return true after token refill")
	}
}

func TestAllowRequestConcurrent(t *testing.T) {
	tb := NewTokenBucket(10, 10)

	var wg sync.WaitGroup
	successCount := 0
	mutex := sync.Mutex{}

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tb.AllowRequest() {
				mutex.Lock()
				successCount++
				mutex.Unlock()
			}
		}()
	}

	wg.Wait()

	if successCount > 10 {
		t.Fatalf("Expected successCount to be at most 10, got %d", successCount)
	}
}
