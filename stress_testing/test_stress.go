package main

import (
	"distributed-lock/clientlib"
	"fmt"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup

	// Connect all clients
	client1, _ := clientlib.New("localhost:5000", "client1")
	time.Sleep(1 * time.Second)
	client2, _ := clientlib.New("localhost:5000", "client2")
	time.Sleep(1 * time.Second)
	client3, _ := clientlib.New("localhost:5000", "client3")
	time.Sleep(1 * time.Second)
	defer client1.Close()
	defer client2.Close()
	defer client3.Close()

	// Client 1
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("[Client 1] Acquiring lock...")
		err := client1.LockAcquire()
		if err != nil {
			fmt.Printf("[Client 1] Lock acquire failed: %v\n", err)
			return
		}
		time.Sleep(2 * time.Second)
		fmt.Println("[Client 1] Lock acquired. Appending 'A' 20 times.")
		for i := 0; i < 20; i++ {
			time.Sleep(300 * time.Millisecond)
			err := client1.AppendFile("file_51", "A")
			if err != nil {
				fmt.Printf("[Client 1] Append %d failed: %v\n", i+1, err)
			} else {
				fmt.Printf("[Client 1] Successfully appended A (%d/20)\n", i+1)
			}
		}
		client1.LockRelease()
		fmt.Println("[Client 1] Lock released.")
	}()

	// Delay to allow client 1 to finish and release the lock
	time.Sleep(8 * time.Second)

	// Client 2
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("[Client 2] Acquiring lock...")
		client2.LockAcquire()
		fmt.Println("[Client 2] Lock acquired. Appending 'B' 20 times.")

		for i := 0; i < 10; i++ {
			time.Sleep(300 * time.Millisecond)
			err := client2.AppendFile("file_51", "B")
			if err != nil {
				fmt.Printf("[Client 2] Append %d failed: %v\n", i+1, err)
			} else {
				fmt.Printf("[Client 2] Successfully appended B (%d/20)\n", i+1)
			}
		}

		// Simulate primary failure
		fmt.Println("[Simulate] Server 1 (Primary) crashes now...")
		// You would trigger this manually or with some signal in a real test

		// Let time pass for failover
		time.Sleep(10 * time.Second)
		fmt.Println("[Info] Failover should have occurred by now. Continuing appends with new primary.")

		for i := 10; i < 20; i++ {
			time.Sleep(300 * time.Millisecond)
			err := client2.AppendFile("file_51", "B")
			if err != nil {
				fmt.Printf("[Client 2] Append %d failed: %v\n", i+1, err)
			} else {
				fmt.Printf("[Client 2] Successfully appended B (%d/20)\n", i+1)
			}
		}
		client2.LockRelease()
		fmt.Println("[Client 2] Lock released.")
	}()

	// Delay to ensure client 2 gets headstart
	time.Sleep(3 * time.Second)

	// Client 3
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(15 * time.Second) // Ensures Client 2's mid-progress
		fmt.Println("[Client 3] Waiting and acquiring lock...")
		client3.LockAcquire()
		fmt.Println("[Client 3] Lock acquired. Appending 'C' 20 times.")
		for i := 0; i < 20; i++ {
			time.Sleep(300 * time.Millisecond)
			err := client3.AppendFile("file_51", "C")
			if err != nil {
				fmt.Printf("[Client 3] Append %d failed: %v\n", i+1, err)
			} else {
				fmt.Printf("[Client 3] Successfully appended C (%d/20)\n", i+1)
			}
		}
		client3.LockRelease()
		fmt.Println("[Client 3] Lock released.")
	}()

	// Wait for all clients
	wg.Wait()

	fmt.Println("[Test Complete] File should contain 20 'A', 20 'B', and 20 'C' in atomic groups.")
}
