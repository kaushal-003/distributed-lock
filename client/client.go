package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	pb "distributed-lock/distributed-lock/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RPCClient struct {
	conn     *grpc.ClientConn
	client   pb.DistributedLockClient
	ClientID string
	hasLock  bool
}

func New(serverAddr, clientName string) (*RPCClient, error) {
	conn, err := grpc.Dial(serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}

	client := pb.NewDistributedLockClient(conn)
	initResp, err := client.InitConnection(context.Background(), &pb.InitRequest{
		ClientName: clientName,
	})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("init failed: %v", err)
	}

	return &RPCClient{
		conn:     conn,
		client:   client,
		ClientID: initResp.ClientId,
		hasLock:  false,
	}, nil
}

func (c *RPCClient) LockAcquire() error {
	if c.hasLock {
		return fmt.Errorf("you already hold the lock")
	}

	fmt.Printf("Requesting lock (Client ID: %s)...\n", c.ClientID)
	stream, err := c.client.LockAcquire(context.Background(), &pb.LockRequest{
		ClientId: c.ClientID,
	})
	if err != nil {
		return fmt.Errorf("lock request failed: %v", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive lock response: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("server denied lock request")
	}

	c.hasLock = true
	fmt.Println("Lock acquired successfully!")
	return nil
}

func (c *RPCClient) LockRelease() error {
	if !c.hasLock {
		return fmt.Errorf("you don't hold the lock")
	}

	fmt.Println("Releasing lock...")
	resp, err := c.client.LockRelease(context.Background(), &pb.LockRequest{
		ClientId: c.ClientID,
	})
	if err != nil {
		return fmt.Errorf("release failed: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("server failed to acknowledge release")
	}

	c.hasLock = false
	fmt.Println("Lock released successfully!")
	return nil
}

func (c *RPCClient) AppendFile(fileName, data string) error {
	if !c.hasLock {
		return fmt.Errorf("you must acquire the lock first")
	}

	fmt.Printf("Writing to %s: %s\n", fileName, data)
	resp, err := c.client.AppendFile(context.Background(), &pb.AppendRequest{
		ClientId: c.ClientID,
		Filename: fileName,
		Data:     []byte(data),
	})
	if err != nil {
		return fmt.Errorf("write failed: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("server failed to write data")
	}

	fmt.Println("Data written successfully!")
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
4. status    - Show current lock status
5. exit      - Close connection and exit`)
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
