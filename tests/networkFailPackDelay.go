package main

import (
	"distributed-lock/clientlib"
	"testing"
	"time"
)

func TestNetworkFailure_PacketDelay(t *testing.T) {
	client1, _ := clientlib.New("localhost:5001", "client1")
	defer client1.Close()

	client2, _ := clientlib.New("localhost:5001", "client2")
	defer client2.Close()

	// C1 acquires the lock
	err := client1.LockAcquire()
	if err != nil {
		t.Fatalf("Client1 failed to acquire lock: %v", err)
	}

	// Simulate delay in append from C1 (network latency)
	go func() {
		time.Sleep(11 * time.Second) // simulate delay
		err := client1.AppendFile("file_1", "A")
		if err == nil {
			t.Errorf("Expected LOCK_EXPIRED error for delayed append, got none")
		}
	}()

	time.Sleep(1 * time.Second) // allow C1 append to get delayed

	// Lock should timeout and go to C2
	err = client2.LockAcquire()
	if err != nil {
		t.Fatalf("Client2 failed to acquire lock: %v", err)
	}

	err = client2.AppendFile("file_1", "B")
	if err != nil {
		t.Fatalf("Client2 failed to append: %v", err)
	}

	// Final check: file should only have B
	// (Assumes your test environment supports reading final file state)
}

func main() {
	TestNetworkFailure_PacketDelay(nil)
}
