package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"sync"
	"time"

	pb "distributed-lock/distributed-lock/proto"

	"google.golang.org/grpc"
)

// request queue
type pendingRequest struct {
	clientID    string
	response    chan int32
	queuenumber int32
}

// data type to store server state in persistent storage
type PersistentState struct {
	LockHolder   string
	PendingQueue []string
	QueueMap     map[string]bool
	Counter      int32
}

type Mixed struct {
	StrVal string
	IntVal int
}

// server metadata
type LockServer struct {
	pb.UnimplementedDistributedLockServer
	mu                sync.Mutex
	lockHolder        string
	pendingQueue      []pendingRequest
	queueMap          map[string]bool
	fileMu            sync.Mutex
	clientCounter     int
	counter           int32
	lockTimeout       time.Duration
	lockTimer         *time.Timer
	lastLockActivity  time.Time
	queueindex        int32
	selfIp            string
	peers             []string
	lastHeartbeatTime time.Time
	leaderIp          string
	clientnumber      int
}

func (s *LockServer) GetQueueIndex(ctx context.Context, req *pb.Empty) (*pb.GetQueueIndexResponse, error) {
	return &pb.GetQueueIndexResponse{Index: s.queueindex}, nil
}

func isReachable(addr string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (s *LockServer) Heartbeat(ctx context.Context, req *pb.Empty) (*pb.Empty, error) {
	s.mu.Lock()
	s.lastHeartbeatTime = time.Now()
	s.mu.Unlock()
	return &pb.Empty{}, nil
}

func (s *LockServer) SendandReceiveHeartbeat() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		currentLeader := s.leaderIp
		selfIp := s.selfIp
		lastHB := s.lastHeartbeatTime
		s.mu.Unlock()

		if selfIp == currentLeader {
			for _, peer := range s.peers {
				if peer == selfIp {
					continue
				}
				if !isReachable(peer) {
					continue
				}
				conn, err := grpc.Dial(peer, grpc.WithInsecure())
				if err != nil {
					log.Printf("Warning: cannot connect to %s for heartbeat: %v", peer, err)
					continue
				}
				client := pb.NewDistributedLockClient(conn)
				_, err = client.Heartbeat(context.Background(), &pb.Empty{})
				if err != nil {
					log.Printf("Warning: heartbeat failed to %s: %v", peer, err)
				}
				conn.Close()
			}
		} else {

			elapsed := time.Since(lastHB)
			log.Printf("Time elapsed since last heartbeat from leader (%s): %v", currentLeader, elapsed)

			if elapsed > 8*time.Second {
				newLeader := s.electLeader()
				s.mu.Lock()
				s.leaderIp = newLeader
				s.lastHeartbeatTime = time.Now()
				s.mu.Unlock()
				log.Printf("Leader elected: %s", newLeader)

				if newLeader == selfIp {
					s.notifyPeers(newLeader)
				}
			}
		}
	}
}

func (s *LockServer) electLeader() string {
	available := []Mixed{}

	if isReachable(s.selfIp) {
		available = append(available, Mixed{StrVal: s.selfIp, IntVal: int(s.queueindex)})
	}

	for _, peer := range s.peers {
		if isReachable(peer) {
			conn, err := grpc.Dial(peer, grpc.WithInsecure())
			if err != nil {
				log.Printf("Warning: cannot connect to %s to get last committed index: %v", peer, err)
				continue
			}
			client := pb.NewDistributedLockClient(conn)
			resp, err := client.GetQueueIndex(context.Background(), &pb.Empty{})
			if err != nil {
				log.Printf("Warning: cannot get last committed index from %s: %v", peer, err)
				conn.Close()
				continue
			}
			Ind := resp.Index
			log.Printf("Last committed index from %s: %d", peer, Ind)
			Ip := peer
			conn.Close()
			available = append(available, Mixed{StrVal: Ip, IntVal: int(Ind)})
		}
	}
	if len(available) == 0 {
		log.Fatal("No available servers for leader election!")
	}

	sort.Slice(available, func(i, j int) bool {
		if available[i].IntVal == available[j].IntVal {
			return available[i].StrVal < available[j].StrVal
		}
		return available[i].IntVal < available[j].IntVal
	})
	newLeader := available[len(available)-1].StrVal
	return newLeader
}

func (s *LockServer) notifyPeers(newLeader string) {
	if len(s.pendingQueue) > 0 {
		go s.grantLock()
	}
	for _, peer := range s.peers {
		if peer == s.selfIp {
			continue
		}

		if !isReachable(peer) {
			continue
		}
		conn, err := grpc.Dial(peer, grpc.WithInsecure())
		if err != nil {
			log.Printf("Warning: cannot connect to %s to update leader: %v", peer, err)
			continue
		}
		client := pb.NewDistributedLockClient(conn)
		_, err = client.UpdateLeader(context.Background(), &pb.UpdateLeaderRequest{LeaderIp: newLeader})
		if err != nil {
			log.Printf("Warning: cannot update leader on %s: %v", peer, err)
		}
		conn.Close()
	}
}

func (s *LockServer) UpdateLeader(ctx context.Context, req *pb.UpdateLeaderRequest) (*pb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.leaderIp = req.LeaderIp
	log.Printf("Leader updated to %s", s.leaderIp)
	return &pb.Empty{}, nil
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

func NewLockServer(ip string, peers []string) *LockServer {
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
		queueindex:       0,
		selfIp:           ip,
		peers:            peers,
		clientnumber:     0,
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

// func (s *LockServer) LockAcquire(req *pb.LockRequest, stream pb.DistributedLock_LockAcquireServer) error {
// 	/*
// 	   LockAcquire handles a client's request to acquire the distributed lock.
// 	   If the current server is not the leader, it forwards the request to the leader.
// 	   If the client already holds the lock, it immediately responds with success.
// 	   If the client is already in the queue, it acknowledges their position.
// 	   Otherwise, the client is added to the pending queue and must wait for
// 	   its turn to acquire the lock.
// 	*/

// 	if s.selfIp != s.leaderIp {
// 		conn, err := grpc.Dial(s.leaderIp, grpc.WithInsecure())
// 		if err != nil {
// 			return stream.Send(&pb.LockResponse{
// 				Success:    false,
// 				StatusCode: 500,
// 				Counter:    0,
// 			})
// 		}
// 		defer conn.Close()

// 		client := pb.NewDistributedLockClient(conn)
// 		leaderStream, err := client.LockAcquire(context.Background(), req)
// 		if err != nil {
// 			return stream.Send(&pb.LockResponse{
// 				Success:    false,
// 				StatusCode: 503,
// 				Counter:    0,
// 			})
// 		}

// 		for {
// 			resp, err := leaderStream.Recv()
// 			if err == io.EOF {
// 				return nil
// 			}
// 			if err != nil {
// 				return err
// 			}
// 			if err := stream.Send(resp); err != nil {
// 				return err
// 			}
// 		}
// 	}

// 	s.mu.Lock()
// 	clientID := req.ClientId

// 	if s.lockHolder == clientID {
// 		s.mu.Unlock()
// 		return stream.Send(&pb.LockResponse{Success: true, StatusCode: 200, Counter: 0})
// 	}

// 	if s.queueMap[clientID] {
// 		s.mu.Unlock()
// 		return stream.Send(&pb.LockResponse{Success: true, StatusCode: 201, Counter: 0})
// 	}

// 	respChan := make(chan int32, 1)
// 	s.queueMap[clientID] = true
// 	s.clientnumber++
// 	s.pendingQueue = append(s.pendingQueue, pendingRequest{
// 		clientID:    clientID,
// 		response:    respChan,
// 		queuenumber: int32(s.clientnumber),
// 	})
// 	log.Printf("Client %s added to queue (position %d)", clientID, len(s.pendingQueue))

// 	if s.selfIp == s.leaderIp {
// 		for _, peer := range s.peers {
// 			if peer == s.selfIp {
// 				continue
// 			}
// 			go func(peer string) {
// 				conn, err := grpc.Dial(peer, grpc.WithInsecure())
// 				if err != nil {
// 					log.Printf("Warning: cannot connect to %s to add to queue: %v", peer, err)
// 					return
// 				}
// 				defer conn.Close()
// 				client := pb.NewDistributedLockClient(conn)
// 				_, err = client.AddQueue(context.Background(), &pb.AddQueueRequest{
// 					ClientId:   clientID,
// 					QueueIndex: int32(s.clientnumber),
// 				})
// 				if err != nil {
// 					log.Printf("Warning: failed to add client %s to queue on %s: %v", clientID, peer, err)
// 				}
// 			}(peer)
// 		}
// 	}

// 	if s.lockHolder == "" && len(s.pendingQueue) == 1 {
// 		s.grantLock()
// 	}

// 	s.mu.Unlock()

// 	code := <-respChan
// 	s.saveState()
// 	return stream.Send(&pb.LockResponse{Success: true, StatusCode: code, Counter: 0})
// }

func (s *LockServer) LockAcquire(req *pb.LockRequest, stream pb.DistributedLock_LockAcquireServer) error {
	// Forward to leader if this isn't the leader
	if s.selfIp != s.leaderIp {
		conn, err := grpc.Dial(s.leaderIp, grpc.WithInsecure())
		if err != nil {
			return stream.Send(&pb.LockResponse{
				Success:    false,
				StatusCode: 500,
				Counter:    0,
			})
		}
		defer conn.Close()

		client := pb.NewDistributedLockClient(conn)
		leaderStream, err := client.LockAcquire(context.Background(), req)
		if err != nil {
			return stream.Send(&pb.LockResponse{
				Success:    false,
				StatusCode: 503,
				Counter:    0,
			})
		}

		for {
			resp, err := leaderStream.Recv()
			if err != nil {
				return err
			}
			if err := stream.Send(resp); err != nil {
				return err
			}
			if resp.StatusCode == 200 { // Lock acquired
				return nil
			}
		}
	}

	s.mu.Lock()
	clientID := req.ClientId

	// Check if client already holds the lock
	if s.lockHolder == clientID {
		s.mu.Unlock()
		return stream.Send(&pb.LockResponse{
			Success:    true,
			StatusCode: 200,
			Counter:    0,
		})
	}

	// Check if client is already in queue
	if s.queueMap[clientID] {
		s.mu.Unlock()
		return stream.Send(&pb.LockResponse{
			Success:    true,
			StatusCode: 201,
			Counter:    0,
		})
	}

	// Create response channel and add to queue
	respChan := make(chan int32, 1)
	s.queueMap[clientID] = true
	s.clientnumber++
	s.pendingQueue = append(s.pendingQueue, pendingRequest{
		clientID:    clientID,
		response:    respChan,
		queuenumber: int32(s.clientnumber),
	})
	log.Printf("Client %s added to queue (position %d)", clientID, len(s.pendingQueue))

	// Propagate to followers if leader
	if s.selfIp == s.leaderIp {
		for _, peer := range s.peers {
			if peer == s.selfIp {
				continue
			}
			go func(peer string) {
				conn, err := grpc.Dial(peer, grpc.WithInsecure())
				if err != nil {
					log.Printf("Warning: cannot connect to %s to add to queue: %v", peer, err)
					return
				}
				defer conn.Close()
				client := pb.NewDistributedLockClient(conn)
				_, err = client.AddQueue(context.Background(), &pb.AddQueueRequest{
					ClientId:   clientID,
					QueueIndex: int32(s.clientnumber),
				})
				if err != nil {
					log.Printf("Warning: failed to add client %s to queue on %s: %v", clientID, peer, err)
				}
			}(peer)
		}
	}

	// Grant lock immediately if queue was empty
	if s.lockHolder == "" && len(s.pendingQueue) == 1 {
		s.grantLock()
	}
	s.mu.Unlock()

	// Wait for response (lock granted or error)
	code := <-respChan
	return stream.Send(&pb.LockResponse{
		Success:    true,
		StatusCode: code,
		Counter:    0,
	})
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
	s.queueindex++
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
	if s.selfIp == s.leaderIp {
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

func (s *LockServer) AddQueue(ctx context.Context, req *pb.AddQueueRequest) (*pb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clientID := req.ClientId
	if !s.queueMap[clientID] {
		s.queueMap[clientID] = true
		s.pendingQueue = append(s.pendingQueue, pendingRequest{
			clientID:    clientID,
			response:    make(chan int32, 1),
			queuenumber: req.QueueIndex,
		})
		log.Printf("Added client %s to queue from leader", clientID)
	}
	return &pb.Empty{}, nil
}

func (s *LockServer) RemoveQueue(ctx context.Context, req *pb.RemoveQueueRequest) (*pb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	queueIndex := req.QueueIndex
	s.queueindex = queueIndex
	s.lockHolder = req.Lockholder
	s.counter = req.Counter
	s.clientnumber = int(req.Clientnumber)

	if len(s.pendingQueue) == 0 || queueIndex <= 0 {
		return &pb.Empty{}, nil
	}

	if int(queueIndex) > len(s.pendingQueue) {
		queueIndex = int32(len(s.pendingQueue))
	}

	for i := 0; i < int(queueIndex); i++ {
		if i < len(s.pendingQueue) {
			delete(s.queueMap, s.pendingQueue[i].clientID)
			if s.pendingQueue[i].response != nil {
				close(s.pendingQueue[i].response)
			}
		}
	}

	if int(queueIndex) <= len(s.pendingQueue) {
		s.pendingQueue = s.pendingQueue[queueIndex:]
	} else {
		s.pendingQueue = s.pendingQueue[:0]
	}

	s.saveState()

	log.Printf("Removed %d clients from queue", queueIndex)
	return &pb.Empty{}, nil
}

func (s *LockServer) UpdateQueue() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		if s.selfIp == s.leaderIp && s.queueindex > 0 {
			currentIndex := s.queueindex

			for _, peer := range s.peers {
				if peer == s.selfIp {
					continue
				}
				go func(peer string) {
					conn, err := grpc.Dial(peer, grpc.WithInsecure())
					if err != nil {
						log.Printf("Warning: cannot connect to %s for queue cleanup: %v", peer, err)
						return
					}
					defer conn.Close()
					client := pb.NewDistributedLockClient(conn)
					_, err = client.RemoveQueue(context.Background(), &pb.RemoveQueueRequest{
						QueueIndex:   currentIndex,
						Lockholder:   s.lockHolder,
						Counter:      s.counter,
						Clientnumber: int32(s.clientnumber),
					})
					// if err != nil {
					// 	log.Printf("Warning: failed to cleanup queue on %s: %v", peer, err)
					// }
				}(peer)
			}
		}
		s.mu.Unlock()
	}
}

func (s *LockServer) GetQueueState(ctx context.Context, req *pb.GetQueueRequest) (*pb.GetQueueResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clientIDs := make([]string, len(s.pendingQueue))
	for i, req := range s.pendingQueue {
		clientIDs[i] = req.clientID
	}

	return &pb.GetQueueResponse{
		ClientIds:  clientIDs,
		QueueIndex: s.queueindex,
		LockHolder: s.lockHolder,
	}, nil
}

func (s *LockServer) syncWithLeader() error {
	conn, err := grpc.Dial(s.leaderIp, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to connect to leader: %v", err)
	}
	defer conn.Close()

	client := pb.NewDistributedLockClient(conn)
	resp, err := client.GetQueueState(context.Background(), &pb.GetQueueRequest{})
	if err != nil {
		return fmt.Errorf("failed to get queue state: %v", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.pendingQueue = make([]pendingRequest, 0)
	s.queueMap = make(map[string]bool)

	for _, clientID := range resp.ClientIds {
		s.queueMap[clientID] = true
		s.pendingQueue = append(s.pendingQueue, pendingRequest{
			clientID: clientID,
			response: make(chan int32, 1),
		})
	}

	s.queueindex = resp.QueueIndex
	s.lockHolder = resp.LockHolder

	log.Println("Successfully synchronized with leader")
	return nil
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

	if s.selfIp != s.leaderIp {
		conn, err := grpc.Dial(s.leaderIp, grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		client := pb.NewDistributedLockClient(conn)
		return client.AppendFile(ctx, req)
	}

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
	if len(os.Args) < 3 {
		log.Fatalf("Usage: server <self-ip> <peer1> <peer2> ...")
	}

	selfIp := os.Args[1]
	peers := os.Args[2:]

	lis, err := net.Listen("tcp", selfIp)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	server := grpc.NewServer()
	srv := NewLockServer(selfIp, peers)
	pb.RegisterDistributedLockServer(server, srv)
	go func() {
		fmt.Printf("Server listening at %s\n", selfIp)
		if err := server.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	time.Sleep(1 * time.Second)
	newLeader := srv.electLeader()
	srv.mu.Lock()
	srv.leaderIp = newLeader
	srv.mu.Unlock()
	log.Printf("Leader elected: %s", newLeader)

	if srv.selfIp == newLeader {
		srv.notifyPeers(newLeader)
	}

	if srv.selfIp != srv.leaderIp {
		err := srv.syncWithLeader()
		if err != nil {
			log.Printf("Failed to sync with leader: %v", err)
		}
	}

	go srv.SendandReceiveHeartbeat()
	go srv.UpdateQueue()

	select {}
}
