package main

import (
	"distributed-lock/clientlib"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type FileResponse struct {
	FileID int      `json:"file_id"`
	Lines  []string `json:"lines"`
}

func main() {
	// Start Client 1
	client1, _ := clientlib.New("localhost:5000", "client1")
	defer client1.Close()

	// Acquire lock
	fmt.Println("Client 1 acquiring lock...")
	client1.LockAcquire()

	// Simulate delay before append
	fmt.Println("Client 1 acquired lock, simulating delay before append...")
	go func() {
		time.Sleep(25 * time.Second) // Delay beyond lock timeout
		fmt.Println("Client 1 attempting delayed append")
		err := client1.AppendFile("file_3", "A")
		if err != nil {
			fmt.Printf("Client 1 Append Error: %v\n", err)
		}
	}()

	// Let server timeout client1's lock
	time.Sleep(22 * time.Second)

	// Start Client 2
	client2, _ := clientlib.New("localhost:5000", "client2")
	defer client2.Close()

	fmt.Println("Client 2 acquiring lock after Client 1 timeout...")
	client2.LockAcquire()

	fmt.Println("Client 2 appending B...")
	client2.AppendFile("file_3", "B")

	// Give time for Client 1's delayed append
	time.Sleep(10 * time.Second)

	fmt.Println("Test complete. File should contain only 'B'")

	resp, err := http.Get("http://localhost:8080/read/3")
	if err != nil {
		log.Fatalf("Error sending request: %v\n", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response body: %v\n", err)
	}

	// Unmarshal the JSON response into the FileResponse struct
	var fileResponse FileResponse
	err = json.Unmarshal(body, &fileResponse)
	if err != nil {
		log.Fatalf("Error unmarshaling response: %v\n", err)
	}

	// Print the file contents
	fmt.Printf("File ID: %d\n", fileResponse.FileID)
	fmt.Println("File Contents:")
	for _, line := range fileResponse.Lines {
		fmt.Println(line)
	}

}
