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

// request queue
type pendingRequest struct {
	clientID string
	response chan int32
}

// data type to store server state in persistent storage
type PersistentState struct {
	LockHolder   string
	PendingQueue []string
	QueueMap     map[string]bool
	Counter      int32
}

// server metadata
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
	/*
		saveState serializes the current state of the LockServer and writes it to a
		file named "lockserver_state.gob".
	*/
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
	/*
		loadState attempts to restore the LockServer's state from a file named
		"lockserver_state.gob". If the file is not found, the server starts with
		a fresh state.
	*/
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
	/*
		NewLockServer initializes and returns a new instance of LockServer.
		It performs the following tasks:
	*/
	//Prepares 100 empty files named "file_0" to "file_99" if they do not exist.
	for i := 0; i < 100; i++ {
		filename := fmt.Sprintf("file_%d", i)
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			os.WriteFile(filename, []byte{}, 0644)
		}
	}

	/*
		Initializes internal data structures including the pending queue,
		queue map, lock timeout, and last lock activity timestamp.
	*/
	s := &LockServer{
		pendingQueue:     make([]pendingRequest, 0),
		queueMap:         make(map[string]bool),
		lockTimeout:      20 * time.Second,
		lastLockActivity: time.Now(),
	}
	//loads prev state if any
	s.loadState()
	if s.lockHolder != "" {
		s.lockTimer = time.AfterFunc(s.lockTimeout, s.timeoutLock)
	}
	return s
}

func (s *LockServer) InitConnection(ctx context.Context, req *pb.InitRequest) (*pb.InitResponse, error) {
	/*
		InitConnection handles the initialization of a new client connection.
		It assigns a unique client ID based on the provided client name or,
		if empty, generates one using an internal counter.
	*/
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
	/*
		LockAcquire handles a client's request to acquire the distributed lock.
		If the client already holds the lock, it immediately responds with success.
		If the client is already in the queue, it acknowledges their position.
		Otherwise, the client is added to the pending queue and must wait for
		its turn to acquire the lock.
	*/
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
	s.saveState()
	return stream.Send(&pb.LockResponse{Success: true, StatusCode: code, Counter: 0})
}

func (s *LockServer) grantLock() {
	/*
		grantLock assigns the lock to the next client in the pending queue.
		If the queue is empty, the method returns immediately.
	*/
	if len(s.pendingQueue) == 0 {
		return
	}

	next := s.pendingQueue[0]
	s.lockHolder = next.clientID

	s.counter = 0
	s.pendingQueue = s.pendingQueue[1:]
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
	/*
		timeoutLock removes access of the lock from current lockHolder
		after timeout.
	*/
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lockHolder == "" {
		return
	}

	if time.Since(s.lastLockActivity) < s.lockTimeout {
		s.lockTimer.Reset(s.lockTimeout)
		return
	}

	log.Printf("Lock timeout for client %s", s.lockHolder)
	delete(s.queueMap, s.lockHolder)
	s.lockHolder = ""

	if len(s.pendingQueue) > 0 {
		s.grantLock()
	}
}

func (s *LockServer) LockRelease(ctx context.Context, req *pb.LockRequest) (*pb.LockResponse, error) {

	/*
		LockRelease handles a client's request to release the currently held lock.
		If the requesting client is not the current lock holder, the request is
		rejected with an appropriate status code(203).
	*/
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

	/*
		AppendFile allows the current lock holder to append data to a specified file.
		The operation is permitted only if the requesting client currently holds
		the lock. It also ensures idempotency using a counter to avoid duplicate writes
	*/
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
