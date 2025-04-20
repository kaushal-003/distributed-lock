package clientlib

import (
	"context"
	"fmt"
	"io"
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

func (c *RPCClient) LockAcquire() error {
	var stream pb.DistributedLock_LockAcquireClient
	var err error
	retryCount := 0
	maxRetries := 5

	for {
		// Create new stream if we don't have one
		if stream == nil {
			stream, err = c.client.LockAcquire(context.Background(), &pb.LockRequest{
				ClientId: c.ClientID,
			})
			if err != nil {
				if retryCount >= maxRetries {
					return fmt.Errorf("failed to acquire lock after %d attempts: %v", maxRetries, err)
				}
				fmt.Printf("Failed to acquire lock stream: %v. Retrying...\n", err)
				time.Sleep(2 * time.Second)
				retryCount++
				continue
			}
			retryCount = 0
		}

		// Receive response from stream
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				// Server closed the stream, which means our request was processed
				fmt.Println("Server closed the stream, retrying...")
				stream = nil
				continue
			}

			fmt.Printf("Error receiving from stream: %v. Re-establishing connection...\n", err)
			stream = nil
			time.Sleep(2 * time.Second)
			continue
		}

		switch resp.StatusCode {
		case 200: // Lock acquired
			c.hasLock = true
			c.count = int(resp.Counter)
			fmt.Println("Lock acquired successfully!")
			return nil

		case 201: // In queue
			fmt.Println("You're in the queue. Waiting for lock to be granted...")
			// Continue waiting for updates

		default:
			fmt.Printf("Unexpected status code: %d\n", resp.StatusCode)
			stream = nil
			time.Sleep(2 * time.Second)
		}
	}
}

func (c *RPCClient) LockRelease() error {
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
