package main

import (
	"context"
	pb "distributed-lock/proto"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"google.golang.org/grpc"
)

type LockServer struct {
	pb.UnimplementedDistributedLockServer
	mu            sync.Mutex
	lockholder    string
	waitqueue     []waitingclient // wait queue
	isholdinglock bool
}

type waitingclient struct {
	client_id      string
	client_ip_port string
}

func NewLockServer() *LockServer {
	s := &LockServer{
		waitqueue:     make([]waitingclient, 0),
		lockholder:    "",
		isholdinglock: false,
	}
	s.initializeFiles()
	return s
}

func (s *LockServer) initializeFiles() {
	for i := 0; i < 100; i++ {
		fileName := fmt.Sprintf("file_%d", i)
		os.WriteFile(fileName, []byte{}, 0644)
	}
}

func (s *LockServer) Init(ctx context.Context, req *pb.InitRequest) (*pb.InitResponse, error) {
	clientID := req.ClientName
	if clientID == "" {
		clientID = fmt.Sprintf("client_%d", len(s.waitqueue)+1)
	}
	log.Printf("Client %s initialized", clientID) // Changed from "connected" to "initialized"
	return &pb.InitResponse{ClientId: clientID, Success: 1}, nil
}

func Response(ctx context.Context, req *pb.LockRequest) (*pb.LockResponse, error) {
	return &pb.LockResponse{Success: 1}, nil
}

func (s *LockServer) LockAcquire(ctx context.Context, req *pb.LockRequest) (*pb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	client_id := req.ClientId
	client_ip_port := req.ClientIpPort

	// Only add to queue if not already in queue or holding the lock
	found := false
	for _, client := range s.waitqueue {
		if client.client_id == client_id {
			found = true
			break
		}
	}

	if !found && s.lockholder != client_id {
		s.waitqueue = append(s.waitqueue, waitingclient{client_id: client_id, client_ip_port: client_ip_port})
		log.Printf("Client %s added to wait queue", client_id)

		// If no one is holding the lock, pass it to the next client
		if !s.isholdinglock {
			s.PassLocktoNext()
		}
	}

	return &pb.Empty{}, nil
}

func (s *LockServer) EmptyQueue() {
	if s.isholdinglock == false {
		if len(s.waitqueue) > 0 {
			s.PassLocktoNext()
		}
	}
}

func (s *LockServer) PassLocktoNext() {
	if len(s.waitqueue) == 0 {
		s.isholdinglock = false
		s.lockholder = ""
		log.Println("No clients in the queue. Lock is now free.")
		return
	}

	nextClient := s.waitqueue[0]
	s.waitqueue = s.waitqueue[1:]
	s.lockholder = nextClient.client_id
	s.isholdinglock = true
	log.Printf("Client %s acquired the lock", s.lockholder)

	// Notify the client that it has acquired the lock
	client_ip_port := "localhost:50052"
	conn, err := grpc.Dial(client_ip_port, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Printf("Failed to connect to client %s at %s: %v", nextClient.client_id, nextClient.client_ip_port, err)
		return
	}
	defer conn.Close()

	client := pb.NewDistributedLockClient(conn)
	_, err = client.LockAcquire(context.Background(), &pb.LockRequest{ClientId: nextClient.client_id})
	if err != nil {
		log.Printf("Failed to notify client %s about lock acquisition: %v", nextClient.client_id, err)
	}
}

func (s *LockServer) LockRelease(ctx context.Context, req *pb.LockRequest) (*pb.LockResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lockholder == req.ClientId {
		if len(s.waitqueue) == 0 {
			s.lockholder = ""
			s.isholdinglock = false
			log.Printf("Client %s released the lock", req.ClientId)
			return &pb.LockResponse{Success: 1}, nil
		} else {
			s.isholdinglock = false
			s.EmptyQueue()
			return &pb.LockResponse{Success: 1}, nil
		}
	} else {
		log.Printf("Client %s does not have the lock", req.ClientId)
		return &pb.LockResponse{Success: 0}, nil
	}
}

func (s *LockServer) AppendFile(ctx context.Context, req *pb.AppendRequest) (*pb.AppendResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lockholder != req.ClientId {
		return &pb.AppendResponse{Success: 1}, nil
	}

	if !strings.HasPrefix(req.FileName, "file_") {
		return &pb.AppendResponse{Success: 1}, nil
	}

	err := os.WriteFile(req.FileName, req.Data, 0644)
	if err != nil {
		return &pb.AppendResponse{Success: 1}, nil
	}
	return &pb.AppendResponse{Success: 0}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	lockServer := NewLockServer()
	s := grpc.NewServer()
	pb.RegisterDistributedLockServer(s, lockServer)

	log.Printf("Server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
