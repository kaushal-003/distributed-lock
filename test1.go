package main

import (
	"distributed-lock/clientlib"
	"fmt"
	"time"
)

func main() {
	// Start Client 1
	client1, _ := clientlib.New("localhost:50051", "client1")
	defer client1.Close()

	// Acquire lock
	fmt.Println("Client 1 acquiring lock...")
	client1.LockAcquire()

	// Simulate delay before append
	fmt.Println("Client 1 acquired lock, simulating delay before append...")
	go func() {
		time.Sleep(25 * time.Second) // Delay beyond lock timeout
		fmt.Println("Client 1 attempting delayed append")
		err := client1.AppendFile("file_0", "A")
		if err != nil {
			fmt.Printf("Client 1 Append Error: %v\n", err)
		}
	}()

	// Let server timeout client1's lock
	time.Sleep(22 * time.Second)

	// Start Client 2
	client2, _ := clientlib.New("localhost:50051", "client2")
	defer client2.Close()

	fmt.Println("Client 2 acquiring lock after Client 1 timeout...")
	client2.LockAcquire()

	fmt.Println("Client 2 appending B...")
	client2.AppendFile("file_0", "B")

	// Give time for Client 1's delayed append
	time.Sleep(10 * time.Second)

	fmt.Println("Test complete. File should contain only 'B'")
}
