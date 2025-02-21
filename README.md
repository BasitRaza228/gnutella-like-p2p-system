# Peer-to-Peer File Sharing System (Go)

A lightweight **Peer-to-Peer (P2P) File Sharing System** built in **Golang**, featuring a central tracker for peer discovery, peer servers for file hosting, and clients for file downloading.

This project demonstrates distributed systems concepts such as peer registration, heartbeat monitoring, file discovery, and chunked file transfer.

---

## Features

- **Tracker-based Coordination**
  - Keeps track of active peers and their shared files.
  - Uses heartbeats to monitor peer availability.
- **Peer-to-Peer Communication**
  - Peers act as both servers (serving files) and clients (downloading files).
  - Shared directories are automatically scanned for available files.
- **File Transfer**
  - Files are transferred in chunks (default: `4096` bytes).
  - Supports multiple peers hosting the same file.
- **Dynamic Discovery**
  - Retrieve a list of available files and peers from the tracker.
  - Download files from multiple peers.

---

## Project Structure

```
peer-to-peer/
│── common/              # Shared protocol definitions & message structures
│   └── protocol.go
│── peer/                # Peer implementation (server + client)
│   ├── peer.go
│   ├── server.go
│   └── client.go
│── tracker/             # Tracker (central coordinator)
│   └── tracker.go
│── shared1/             # Example shared directory
│   └── example.txt
│── shared2/             # Another example shared directory
│   └── example.txt
│── commands.txt         # Sample commands for testing
│── go.mod               # Go module definition
```

---

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/BasitRaza228/gnutella-like-p2p-system.git
   cd peer-to-peer
   ```

2. Install dependencies (Go modules):

   ```bash
   go mod tidy
   ```

3. Build the tracker and peer:
   ```bash
   cd tracker && go build -o tracker
   cd ../peer && go build -o peer
   ```

---

## Usage

### 1. Start the Tracker

Run the tracker (default port `9000`):

```bash
./tracker :9000
```

### 2. Start a Peer

Each peer needs:

- A tracker address
- A server port for incoming requests
- A shared directory with files

Example:

```bash
./peer -tracker=localhost:9000 -port=10001 -shared=../shared1
./peer -tracker=localhost:9000 -port=10002 -shared=../shared2
```

### 3. Peer Commands

Inside a running peer instance, you can:

- **List files** available across the network.
- **Download file** from peers.
- **Check active peers** via tracker.

(See `commands.txt` for examples.)

---

## Example Workflow

1. Start the tracker:

   ```bash
   ./tracker :9000
   ```

2. Start two peers with different shared directories:

   ```bash
   ./peer -tracker=localhost:9000 -port=10001 -shared=../shared1
   ./peer -tracker=localhost:9000 -port=10002 -shared=../shared2
   ```

3. From Peer 1, list available files:

   ```
   list
   ```

4. Download a file from Peer 2:
   ```
   download example.txt
   ```

---

## Protocol

Peers and tracker communicate using JSON messages over TCP.  
Supported commands:

- `register` → Register peer & shared files with tracker.
- `heartbeat` → Maintain peer liveness.
- `list` → Get list of available files.
- `getpeers` → Fetch peers hosting a specific file.
- `download` → Request file transfer from another peer.

---

## Future Improvements

- Parallel downloads from multiple peers.
- Resume support for interrupted downloads.
- File integrity verification (hash checks).
- GUI or Web-based interface.

---

## License

This project is licensed under the MIT License.
