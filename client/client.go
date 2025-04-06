package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	pb "distributed-lock/distributed-lock/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RPCClient struct {
	conn     *grpc.ClientConn
	client   pb.DistributedLockClient
	ClientID string
	hasLock  bool
	count    int
}

func New(serverAddr, clientName string) (*RPCClient, error) {
	for {
		conn, err := grpc.Dial(serverAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock())
		if err != nil {
			fmt.Printf("Failed to connect to server: %v. Retrying in 2 secs...\n", err)
			time.Sleep(2 * time.Second)
			continue
		}

		client := pb.NewDistributedLockClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		initResp, err := client.InitConnection(ctx, &pb.InitRequest{
			ClientName: clientName,
		})
		if err != nil {
			conn.Close()
			fmt.Printf("Init failed: %v. Retrying in 2 secs...\n", err)
			time.Sleep(2 * time.Second)
			continue
		}

		return &RPCClient{
			conn:     conn,
			client:   client,
			ClientID: initResp.ClientId,
			hasLock:  false,
		}, nil
	}
}

// func (c *RPCClient) LockAcquire() error {
// 	if c.hasLock {
// 		return fmt.Errorf("you already hold the lock")
// 	}

// 	isInQueue := false
// 	var stream pb.DistributedLock_LockAcquireClient
// 	var err error

// 	for {
// 		if !isInQueue {
// 			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
// 			defer cancel()
// 			stream, err = c.client.LockAcquire(ctx, &pb.LockRequest{
// 				ClientId: c.ClientID,
// 			})
// 			if err != nil {
// 				fmt.Printf("Attempt failed to acquire lock, retrying in 2 secs...\n")
// 				time.Sleep(2 * time.Second)
// 				continue
// 			}
// 		}

// 		resp, err := stream.Recv()
// 		fmt.Println(resp.StatusCode)
// 		if err != nil {
// 			// fmt.Printf("Error receiving stream: %v. Retrying...\n", err)
// 			time.Sleep(2 * time.Second)
// 			continue
// 		}

// 		if resp.StatusCode == 200 {
// 			c.hasLock = true
// 			c.count = int(resp.Counter)
// 			fmt.Println("Lock acquired successfully!")
// 			return nil
// 		} else if resp.StatusCode == 201 {
// 			isInQueue = true
// 			fmt.Println("You're in the queue. Waiting for lock to be granted...")
// 		} else {
// 			time.Sleep(2 * time.Second)
// 		}
// 		time.Sleep(2 * time.Second)
// 	}
// }

func (c *RPCClient) LockAcquire() error {
	// if c.hasLock {
	// 	return fmt.Errorf("you already hold the lock")
	// }

	var stream pb.DistributedLock_LockAcquireClient
	var err error
	isInQueue := false

	for {
		if !isInQueue {
			stream, err = c.client.LockAcquire(context.Background(), &pb.LockRequest{
				ClientId: c.ClientID,
			})
			if err != nil {
				fmt.Printf("Attempt failed to acquire lock: %v. Retrying in 2 secs...\n", err)
				time.Sleep(2 * time.Second)
				continue
			}
		}

		resp, err := stream.Recv()
		if err != nil {
			fmt.Printf("Error receiving from stream: %v. Retrying...\n", err)
			time.Sleep(2 * time.Second)
			isInQueue = false
			continue
		}

		fmt.Println(resp.StatusCode)
		if resp.StatusCode == 200 {
			c.hasLock = true
			c.count = int(resp.Counter)
			fmt.Println("Lock acquired successfully!")
			return nil
		} else if resp.StatusCode == 201 {
			isInQueue = true
			fmt.Println("You're in the queue. Waiting for lock to be granted...")
		} else {
			time.Sleep(2 * time.Second)
		}
	}

}

func (c *RPCClient) LockRelease() error {
	// if !c.hasLock {
	// 	return fmt.Errorf("you don't hold the lock")
	// }

	fmt.Println("Releasing lock...")
	resp, err := c.client.LockRelease(context.Background(), &pb.LockRequest{
		ClientId: c.ClientID,
	})
	if err != nil {
		return fmt.Errorf("release failed: %v", err)
	}

	if resp.StatusCode == 203 {
		c.hasLock = false
		return fmt.Errorf("you don't hold the lock")
	}

	if !resp.Success {
		return fmt.Errorf("server failed to acknowledge release")
	}

	c.hasLock = false
	fmt.Println("Lock released successfully!")
	return nil
}

func (c *RPCClient) AppendFile(fileName, data string) error {
	// if !c.hasLock {
	// 	return fmt.Errorf("you must acquire the lock first")
	// }

	fmt.Printf("Writing to %s: %s\n", fileName, data)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		resp, err := c.client.AppendFile(ctx, &pb.AppendRequest{
			ClientId: c.ClientID,
			Filename: fileName,
			Data:     []byte(data),
			Counter:  int32(c.count + 1),
		})
		if resp.StatusCode == 203 {
			c.hasLock = false
			return fmt.Errorf("you don't hold the lock")
		}

		if err != nil {
			time.Sleep(2 * time.Second)
		}

		if !resp.Success {
			time.Sleep(2 * time.Second)
		}

		if resp.Success {
			fmt.Println("Data written successfully!")
			break
		}

	}
	c.count++
	return nil
}

func (c *RPCClient) Close() error {
	if c.hasLock {
		fmt.Println("Auto-releasing lock before closing...")
		if err := c.LockRelease(); err != nil {
			fmt.Printf("Warning: failed to release lock - %v\n", err)
		}
	}
	return c.conn.Close()
}

func printMenu() {
	fmt.Println(`
Available commands:
1. acquire   - Request the lock (blocks until acquired)
2. write     <file> <data> - Write data to file
3. release   - Release the lock
4. exit      - Close connection and exit`)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: client <server-ip:port> [client-name]")
		os.Exit(1)
	}

	serverAddr := os.Args[1]
	clientName := ""
	if len(os.Args) > 2 {
		clientName = os.Args[2]
	}

	client, err := New(serverAddr, clientName)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	fmt.Printf("Connected to server as client ID: %s\n", client.ClientID)
	reader := bufio.NewReader(os.Stdin)

	for {
		printMenu()
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		parts := strings.SplitN(input, " ", 3)
		cmd := parts[0]

		switch cmd {
		case "acquire":
			if err := client.LockAcquire(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "write":
			if len(parts) < 3 {
				fmt.Println("Usage: write <file> <data>")
				continue
			}
			data := parts[2]
			if len(parts) > 3 {
				data = strings.Join(parts[2:], " ")
			}
			if err := client.AppendFile(parts[1], data); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "release":
			if err := client.LockRelease(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "exit":
			fmt.Println("Closing connection...")
			err := client.Close()
			if err != nil {
				fmt.Printf("Warning: failed to close connection - %v\n", err)
			}
			return

		default:
			fmt.Println("Invalid command. Type 'help' for available commands")
		}
	}
}
