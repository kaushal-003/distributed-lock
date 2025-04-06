package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	pb "distributed-lock/distributed-lock/proto"

	"google.golang.org/grpc"
)

type pendingRequest struct {
	clientID string
	response chan int32
}

type PersistentState struct {
	LockHolder   string
	PendingQueue []string
	QueueMap     map[string]bool
	Counter      int32
}

type LockServer struct {
	pb.UnimplementedDistributedLockServer
	mu               sync.Mutex
	lockHolder       string
	pendingQueue     []pendingRequest
	queueMap         map[string]bool
	fileMu           sync.Mutex
	clientCounter    int
	counter          int32
	lockTimeout      time.Duration
	lockTimer        *time.Timer
	lastLockActivity time.Time
}

func (s *LockServer) saveState() {
	state := PersistentState{
		LockHolder:   s.lockHolder,
		PendingQueue: make([]string, len(s.pendingQueue)),
		QueueMap:     s.queueMap,
		Counter:      s.counter,
	}

	for i, req := range s.pendingQueue {
		state.PendingQueue[i] = req.clientID
	}

	// Open or create the file, truncate it to overwrite
	file, err := os.OpenFile("lockserver_state.gob", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Error opening/creating state file: %v", err)
		return
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(state); err != nil {
		log.Printf("Failed to encode state: %v", err)
		return
	}

	log.Println("Server state successfully saved.")
}

func (s *LockServer) loadState() {
	file, err := os.Open("lockserver_state.gob")
	if err != nil {
		log.Println("No previous state found, starting fresh.")
		return
	}
	defer file.Close()

	var state PersistentState
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&state); err != nil {
		log.Printf("Failed to decode state: %v", err)
		return
	}

	s.lockHolder = state.LockHolder
	s.queueMap = state.QueueMap
	s.counter = state.Counter

	s.pendingQueue = make([]pendingRequest, len(state.PendingQueue))
	for i, clientID := range state.PendingQueue {
		s.pendingQueue[i] = pendingRequest{
			clientID: clientID,
			response: make(chan int32, 1),
		}
	}

	log.Println("Recovery completed")
}

func NewLockServer() *LockServer {
	for i := 0; i < 100; i++ {
		filename := fmt.Sprintf("file_%d", i)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			os.WriteFile(filename, []byte{}, 0644)
		}
	}
	s := &LockServer{
		pendingQueue:     make([]pendingRequest, 0),
		queueMap:         make(map[string]bool),
		lockTimeout:      20 * time.Second,
		lastLockActivity: time.Now(),
	}

	s.loadState()
	if s.lockHolder != "" {
		s.lockTimer = time.AfterFunc(s.lockTimeout, s.timeoutLock)
	}
	return s
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

	//log.Println("Size of pendingQueue:", len(s.pendingQueue))

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
	s.saveState()
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

	s.lastLockActivity = time.Now()
	if s.lockTimer != nil {
		s.lockTimer.Stop()
	}
	s.lockTimer = time.AfterFunc(s.lockTimeout, s.timeoutLock)

	next.response <- 200
	close(next.response)
}

func (s *LockServer) timeoutLock() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lockHolder == "" {
		return
	}

	if time.Since(s.lastLockActivity) < s.lockTimeout {
		s.lockTimer.Reset(s.lockTimeout)
		return
	}

	// resp, _:=s.Timeout(context.Background(), &pb.Empty{})
	// if resp.Success{
	// 	log.Printf("Lock timeout for client %s", s.lockHolder)
	// }
	log.Printf("Lock timeout for client %s", s.lockHolder)
	delete(s.queueMap, s.lockHolder)
	s.lockHolder = ""

	if len(s.pendingQueue) > 0 {
		s.grantLock()
	}
}

func (s *LockServer) LockRelease(ctx context.Context, req *pb.LockRequest) (*pb.LockResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clientID := req.ClientId

	if s.lockHolder != clientID {
		return &pb.LockResponse{Success: false, StatusCode: 203}, nil
	}

	log.Printf("Client %s released the lock", clientID)
	s.lockHolder = ""
	delete(s.queueMap, clientID)

	if s.lockTimer != nil {
		s.lockTimer.Stop()
		s.lockTimer = nil
	}

	if len(s.pendingQueue) > 0 {
		s.grantLock()
	}
	s.saveState()
	return &pb.LockResponse{Success: true}, nil
}

func (s *LockServer) AppendFile(ctx context.Context, req *pb.AppendRequest) (*pb.AppendResponse, error) {
	s.fileMu.Lock()
	defer s.fileMu.Unlock()

	s.mu.Lock()
	lockHolder := s.lockHolder
	s.mu.Unlock()

	if lockHolder != req.ClientId {
		return &pb.AppendResponse{Success: false, Counter: s.counter, StatusCode: 203}, nil
	}

	filename := req.Filename
	count := req.Counter

	if count == s.counter {
		return &pb.AppendResponse{Success: false, Counter: s.counter, StatusCode: 201}, nil
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return &pb.AppendResponse{Success: false, Counter: s.counter, StatusCode: 201}, nil
	}
	defer file.Close()

	if _, err := file.Write(req.Data); err != nil {
		return &pb.AppendResponse{Success: false, Counter: s.counter, StatusCode: 201}, nil
	}
	s.counter = count
	s.saveState()
	return &pb.AppendResponse{Success: true, Counter: s.counter, StatusCode: 200}, nil
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
