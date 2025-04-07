# Distributed Lock Service using gRPC

This project implements a fault-tolerant distributed lock service using **Go** and **gRPC**, allowing multiple clients to coordinate access to shared resources like files. It supports features like queuing, timeout-based lock expiration, persistent state recovery, and file writes with idempotency guarantees.

---

## Features

- 🔐 **Mutual Exclusion**: Only one client can hold the lock at a time.
- ⏳ **Automatic Lock Timeout**: Lock is automatically released after inactivity (20s).
- 🕓 **Persistent State**: Lock state is saved to disk and restored on restart (`lockserver_state.gob`).
- 🧾 **Queueing**: Clients are added to a queue if the lock is already held.
- ✍️ **File Append**: Only the current lock holder can append to files.
- ⚠️ **Idempotent Writes**: Counter mechanism prevents duplicate writes.
- 💬 **gRPC Streaming**: Clients are notified when the lock is granted via stream.

---

## Run the Server File
```
go run server.go
```
The server listens on port :50051 and auto-initializes 100 files named file_0 to file_99.

## Run a Client
```
go run client.go localhost:50051 [client_name(Optional)]
```
You can open multiple terminal windows to simulate multiple clients.

## Client Commands

```
> acquire              # Request the distributed lock
> write <file> <data>  # Append <data> to <file> (only if lock is held)
> release              # Release the lock
> exit                 # Disconnect client and auto-release lock if held
```

### Contributors
1. Kaushal Kothiya(21110107)
2. Anish Karnik(21110098)
