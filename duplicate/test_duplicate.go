package main

import (
	"distributed-lock/clientlib"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	// Start Client 1
	client1, _ := clientlib.New("localhost:5000", "client1")
	defer client1.Close()

	fmt.Println("Client 1 acquiring lock...")
	client1.LockAcquire()
	fmt.Println("Client 1 appending A...")
	client1.AppendFile("file_4", "A")

	// Simulate duplicate release: delay real release
	fmt.Println("Simulating delayed lock release...")
	go func() {
		time.Sleep(3 * time.Second)
		fmt.Println("Client 1 retrying lock release (duplicate)...")
		client1.LockRelease() // this one is the delayed retry
	}()

	time.Sleep(1 * time.Second) // before the retry fires
	fmt.Println("Client 1 releasing lock (real release)...")
	client1.LockRelease() // release should succeed

	// Start Client 2
	client2, _ := clientlib.New("localhost:5000", "client2")
	defer client2.Close()

	fmt.Println("Client 2 acquiring lock...")
	client2.LockAcquire()
	fmt.Println("Client 2 appending B...")
	client2.AppendFile("file_4", "B")

	// Append another B to simulate repeated usage while holding the lock
	fmt.Println("Client 2 appending B again...")
	client2.AppendFile("file_4", "B")

	fmt.Println("Client 2 releasing lock...")
	client2.LockRelease()

	// Wait for Client 1's delayed release to happen (which server should ignore)
	time.Sleep(2 * time.Second)

	// Client 1 reacquires lock and appends A again
	fmt.Println("Client 1 reacquiring lock...")
	client1.LockAcquire()
	fmt.Println("Client 1 appending A again...")
	client1.AppendFile("file_4", "A")
	client1.LockRelease()

	// Final file check
	time.Sleep(2 * time.Second)
	fmt.Println("🔍 Final file check...")

	resp, err := http.Get("http://localhost:8080/read/4")
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		FileID int      `json:"file_id"`
		Lines  []string `json:"lines"`
	}
	json.Unmarshal(body, &result)

	fmt.Printf("📄 File content: %v\n", result.Lines)

	expected := []string{"A", "B", "B", "A"}
	match := len(result.Lines) == len(expected)
	for i := range expected {
		if result.Lines[i] != expected[i] {
			match = false
			break
		}
	}

	if match {
		fmt.Println("✅ PASS: File content is correct: [A B B A]")
	} else {
		fmt.Println("❌ FAIL: Unexpected file content:", result.Lines)
	}
}
