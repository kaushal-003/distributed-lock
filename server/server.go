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
	response chan int32
}

type LockServer struct {
	pb.UnimplementedDistributedLockServer
	mu            sync.Mutex
	lockHolder    string
	pendingQueue  []pendingRequest
	queueMap      map[string]bool
	fileMu        sync.Mutex
	clientCounter int
	counter       int32
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
		queueMap:     make(map[string]bool),
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
		return stream.Send(&pb.LockResponse{Success: true, StatusCode: 200, Counter: 0})
	}

	if s.queueMap[clientID] {
		s.mu.Unlock()
		return stream.Send(&pb.LockResponse{Success: true, StatusCode: 201, Counter: 0})
	}

	respChan := make(chan int32, 1)
	s.queueMap[clientID] = true

	s.pendingQueue = append(s.pendingQueue, pendingRequest{
		clientID: clientID,
		response: respChan,
	})
	log.Printf("Client %s added to queue (position %d)", clientID, len(s.pendingQueue))

	if s.lockHolder == "" && len(s.pendingQueue) == 1 {
		s.grantLock()
	}

	s.mu.Unlock()

	code := <-respChan
	return stream.Send(&pb.LockResponse{Success: true, StatusCode: code, Counter: 0})
}

func (s *LockServer) grantLock() {
	if len(s.pendingQueue) == 0 {
		return
	}

	next := s.pendingQueue[0]
	s.lockHolder = next.clientID
	s.counter = 0
	//TODO:
	s.pendingQueue = s.pendingQueue[1:] //may become a bottleneck
	log.Printf("Lock granted to %s", next.clientID)

	next.response <- 200
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
	delete(s.queueMap, clientID)

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
		return &pb.AppendResponse{Success: false, Counter: s.counter}, nil
	}

	filename := req.Filename
	count := req.Counter

	if count == s.counter {
		return &pb.AppendResponse{Success: false, Counter: s.counter}, nil
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return &pb.AppendResponse{Success: false, Counter: s.counter}, nil
	}
	defer file.Close()

	if _, err := file.Write(req.Data); err != nil {
		return &pb.AppendResponse{Success: false, Counter: s.counter}, nil
	}
	s.counter = count
	return &pb.AppendResponse{Success: true, Counter: s.counter}, nil
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
