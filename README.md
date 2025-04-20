# Distributed Lock Service using gRPC

A bounded consistency-based robust, fault-tolerant, leader-based distributed lock service implemented in Go using gRPC. This system enables clients to safely acquire and release locks and perform write operations to a distributed storage backend, ensuring mutual exclusion in concurrent environments.

## 📑 Table of Contents
- [Introduction](#introduction)
- [Architecture](#architecture)
- [Features](#features)
- [Usage](#usage)
- [Contributors](#contributors)

## 📘 Introduction
This project implements a **Distributed Lock Server** with client-server architecture using gRPC. It ensures:
- Bounded consistency (within 20s)
- Mutual exclusion across distributed clients.
- Automatic leader election.
- Heartbeat-based failure detection.
- Persistent state recovery.
- Write operations only by lock holders (append to a distributed storage or file system).

## 🏗️ Architecture
- **Server**:
  - Maintains a queue of pending lock requests.
  - Elects a leader among peers using a priority queue based on queue index.
  - Propagates queue updates and lock states to all peers.
  - Exposes lock acquire/release, heartbeat, and file append services via gRPC.

- **Client**:
  - Interacts with the server to request/release locks.
  - Performs file write operations when the lock is acquired.
  - Handles reconnects and retries.

- **Storage**:
  - Integrates with a distributed backend (like MongoDB via an HTTP endpoint) for persistent writes.

## ✨ Features
- 🛡️ **Leader Election**: Automatically selects a new leader on failure.
- 🔄 **Heartbeat Monitoring**: Detects server health.
- 🗂️ **Queue Management**: Ensures fair access to locks.
- 📁 **File Writing Support**: Only the lock holder can write.
- 💾 **Persistent State**: Saves and restores server state after crash.
- 👥 **Multi-Client Support**: Simultaneous lock requests handled orderly.
- 📡 **gRPC Interface**: Simple and efficient RPC calls.

### Usage
#### 1. Start the File Writer Server (MongoDB interface)
```bash
go run fileServer/server.go
```
#### 2. Start Lock Servers
```bash
# Example with 3 servers
./lock-server 127.0.0.1:5000 127.0.0.1:5001 127.0.0.1:5002 localhost:8080
./lock-server 127.0.0.1:5001 127.0.0.1:5000 127.0.0.1:5002 localhost:8080
./lock-server 127.0.0.1:5002 127.0.0.1:5000 127.0.0.1:5001 localhost:8080
```
#### 3. Start a Client
```bash
./lock-client 127.0.0.1:5000 [client-name]
```

#### 4. Available Commands
```bash
1. acquire             - Request the lock (blocks until acquired)
2. write <file> <data> - Append data to file (requires lock)
3. release             - Release the lock
4. exit                - Exit the client gracefully
```

### Contributors
1. Kaushal Kothiya(21110107)
2. Anish Karnik(21110098)
