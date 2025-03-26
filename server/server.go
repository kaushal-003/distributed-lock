package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	pb "distributed-lock/distributed-lock/proto"

	"google.golang.org/grpc"
)

type pendingRequest struct {
	clientID string
	response chan bool
}

type LockServer struct {
	pb.UnimplementedDistributedLockServer
	mu            sync.Mutex
	lockHolder    string
	pendingQueue  []pendingRequest
	fileMu        sync.Mutex
	clientCounter int
}

func NewLockServer() *LockServer {

	for i := 0; i < 100; i++ {
		filename := fmt.Sprintf("file_%d", i)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			os.WriteFile(filename, []byte{}, 0644)
		}
	}
	return &LockServer{
		pendingQueue: make([]pendingRequest, 0),
	}
}

func (s *LockServer) InitConnection(ctx context.Context, req *pb.InitRequest) (*pb.InitResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clientCounter++
	clientID := req.ClientName
	if clientID == "" {
		clientID = fmt.Sprintf("client_%d", s.clientCounter)
	}

	log.Printf("New client connected: %s", clientID)
	return &pb.InitResponse{ClientId: clientID, Success: true}, nil
}

func (s *LockServer) LockAcquire(req *pb.LockRequest, stream pb.DistributedLock_LockAcquireServer) error {
	s.mu.Lock()
	clientID := req.ClientId

	if s.lockHolder == clientID {
		s.mu.Unlock()
		return stream.Send(&pb.LockResponse{Success: true})
	}

	respChan := make(chan bool, 1)

	s.pendingQueue = append(s.pendingQueue, pendingRequest{
		clientID: clientID,
		response: respChan,
	})
	log.Printf("Client %s added to queue (position %d)", clientID, len(s.pendingQueue))

	if s.lockHolder == "" && len(s.pendingQueue) == 1 {
		s.grantLock()
	}

	s.mu.Unlock()

	success := <-respChan
	return stream.Send(&pb.LockResponse{Success: success})
}

func (s *LockServer) grantLock() {
	if len(s.pendingQueue) == 0 {
		return
	}

	next := s.pendingQueue[0]
	s.lockHolder = next.clientID
	s.pendingQueue = s.pendingQueue[1:]
	log.Printf("Lock granted to %s", next.clientID)

	next.response <- true
	close(next.response)
}

func (s *LockServer) LockRelease(ctx context.Context, req *pb.LockRequest) (*pb.LockResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clientID := req.ClientId

	if s.lockHolder != clientID {
		return &pb.LockResponse{Success: false}, nil
	}

	log.Printf("Client %s released the lock", clientID)
	s.lockHolder = ""

	if len(s.pendingQueue) > 0 {
		s.grantLock()
	}

	return &pb.LockResponse{Success: true}, nil
}

func (s *LockServer) AppendFile(ctx context.Context, req *pb.AppendRequest) (*pb.AppendResponse, error) {
	s.fileMu.Lock()
	defer s.fileMu.Unlock()

	s.mu.Lock()
	lockHolder := s.lockHolder
	s.mu.Unlock()

	if lockHolder != req.ClientId {
		return &pb.AppendResponse{Success: false}, nil
	}

	filename := req.Filename

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return &pb.AppendResponse{Success: false}, nil
	}
	defer file.Close()

	if _, err := file.Write(req.Data); err != nil {
		return &pb.AppendResponse{Success: false}, nil
	}

	return &pb.AppendResponse{Success: true}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := NewLockServer()
	grpcServer := grpc.NewServer()
	pb.RegisterDistributedLockServer(grpcServer, server)

	log.Printf("Distributed Lock Server started on %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
