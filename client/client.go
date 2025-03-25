package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	pb "distributed-lock/proto"

	"google.golang.org/grpc"
)

type RPCClient struct {
	conn     *grpc.ClientConn
	client   pb.DistributedLockClient
	ClientID string
	ClientIP string
	hasLock  bool
	lockCh   chan bool
}

func New(serverAddr, clientName, clientIP string) (*RPCClient, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}

	client := pb.NewDistributedLockClient(conn)
	initResp, err := client.Init(context.Background(), &pb.InitRequest{ClientName: clientName})
	if err != nil {
		return nil, fmt.Errorf("init failed: %v", err)
	}

	return &RPCClient{
		conn:     conn,
		client:   client,
		ClientID: initResp.ClientId,
		ClientIP: clientIP,
		hasLock:  false,
		lockCh:   make(chan bool, 1),
	}, nil
}

func (c *RPCClient) LockAcquire() error {
	if c.hasLock {
		return fmt.Errorf("you already hold the lock")
	}

	_, err := c.client.LockAcquire(context.Background(), &pb.LockRequest{ClientId: c.ClientID, ClientIpPort: c.ClientIP})
	if err != nil {
		return err
	}

	fmt.Println("Waiting for lock...")
	<-c.lockCh // Wait for lock to be granted
	c.hasLock = true
	fmt.Println("Lock acquired!")
	return nil
}

func (c *RPCClient) ListenForLockGrant() {
	for {
		time.Sleep(1 * time.Second) // Polling interval
		resp, err := c.client.LockAcquire(context.Background(), &pb.LockRequest{ClientId: c.ClientID})
		if err == nil && resp != nil {
			c.lockCh <- true
			break
		}
	}
}

func (c *RPCClient) LockRelease() error {
	if !c.hasLock {
		return fmt.Errorf("you don't hold the lock")
	}

	resp, err := c.client.LockRelease(context.Background(), &pb.LockRequest{ClientId: c.ClientID})
	if err != nil {
		return err
	}
	if resp.Success == 0 {
		return fmt.Errorf("lock release failed")
	}
	c.hasLock = false
	fmt.Println("Lock released!")
	return nil
}

func (c *RPCClient) AppendFile(fileName, data string) error {
	if !c.hasLock {
		return fmt.Errorf("you need to acquire the lock first")
	}

	resp, err := c.client.AppendFile(context.Background(), &pb.AppendRequest{
		ClientId: c.ClientID,
		FileName: fileName,
		Data:     []byte(data),
	})
	if err != nil {
		return err
	}
	if resp.Success != 0 {
		return fmt.Errorf("write failed")
	}
	fmt.Printf("Successfully wrote to %s\n", fileName)
	return nil
}

func (c *RPCClient) Close() error {
	if c.hasLock {
		if err := c.LockRelease(); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}
	return c.conn.Close()
}

func printMenu() {
	fmt.Println(`\nAvailable commands:
1. acquire - Request the lock (blocks until acquired)
2. write <file> <data> - Write data to file
3. release - Release the lock
5. exit - Close connection and exit`)
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: client <server-ip:port> <client-ip:port> [client-name]")
		os.Exit(1)
	}

	serverAddr := os.Args[1]
	clientIP := os.Args[2]
	clientName := ""
	if len(os.Args) > 3 {
		clientName = os.Args[3]
	}

	client, err := New(serverAddr, clientName, clientIP)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	fmt.Printf("Connected as client %s\n", client.ClientID)
	reader := bufio.NewReader(os.Stdin)

	go client.ListenForLockGrant()

	for {
		printMenu()
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		parts := strings.Split(input, " ")

		switch parts[0] {
		case "acquire":
			if err := client.LockAcquire(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "write":
			if len(parts) < 3 {
				fmt.Println("Usage: write <file> <data>")
				continue
			}
			if err := client.AppendFile(parts[1], strings.Join(parts[2:], " ")); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "release":
			if err := client.LockRelease(); err != nil {
				fmt.Printf("Error: %v\n", err)
			}

		case "exit":
			fmt.Println("Exiting...")
			return

		default:
			fmt.Println("Invalid command")
		}
	}
}
