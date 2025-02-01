package main

import (
	"log"
	"net"
	"os"
	"sync"
	"time"

	"peer-to-peer/common"
)

type Tracker struct {
	peers     map[string]time.Time
	files     map[string]map[string]bool
	peersLock sync.RWMutex
	filesLock sync.RWMutex
}

func NewTracker() *Tracker {
	return &Tracker{
		peers: make(map[string]time.Time),
		files: make(map[string]map[string]bool),
	}
}

func (t *Tracker) Start(port string) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Printf("Tracker running on port %s\n", port)

	go t.cleanupRoutine()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Connection error:", err)
			continue
		}
		go t.handleConnection(conn)
	}
}

func (t *Tracker) cleanupRoutine() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		t.peersLock.Lock()
		now := time.Now()
		for addr, lastSeen := range t.peers {
			if now.Sub(lastSeen) > 3*time.Minute {
				delete(t.peers, addr)
				t.filesLock.Lock()
				for file := range t.files {
					delete(t.files[file], addr)
					if len(t.files[file]) == 0 {
						delete(t.files, file)
					}
				}
				t.filesLock.Unlock()
				log.Printf("Removed inactive peer: %s\n", addr)
			}
		}
		t.peersLock.Unlock()
	}
}

func (t *Tracker) handleConnection(conn net.Conn) {
	defer conn.Close()

	msg, err := common.ReadMessage(conn, 10*time.Second)
	if err != nil {
		log.Println("Read error:", err)
		return
	}

	switch msg.Command {
	case common.RegisterCmd:
		t.handleRegister(conn, msg)
	case common.HeartbeatCmd:
		t.handleHeartbeat(conn, msg)
	case common.ListCmd:
		t.handleList(conn)
	case common.GetPeersCmd:
		t.handleGetPeers(conn, msg)
	default:
		log.Println("Unknown command:", msg.Command)
	}
}

func (t *Tracker) handleRegister(conn net.Conn, msg *common.Message) {
	t.peersLock.Lock()
	defer t.peersLock.Unlock()
	t.filesLock.Lock()
	defer t.filesLock.Unlock()

	t.peers[msg.Address] = time.Now()

	for _, file := range msg.Files {
		if t.files[file] == nil {
			t.files[file] = make(map[string]bool)
		}
		t.files[file][msg.Address] = true
	}

	response := &common.Message{Status: "ok"}
	if err := common.WriteMessage(conn, response, 5*time.Second); err != nil {
		log.Println("Write error:", err)
	}
}

func (t *Tracker) handleHeartbeat(conn net.Conn, msg *common.Message) {
	t.peersLock.Lock()
	defer t.peersLock.Unlock()

	if _, exists := t.peers[msg.Address]; exists {
		t.peers[msg.Address] = time.Now()
		response := &common.Message{Status: "ok"}
		if err := common.WriteMessage(conn, response, 5*time.Second); err != nil {
			log.Println("Write error:", err)
		}
	} else {
		response := &common.Message{Status: "peer not registered"}
		if err := common.WriteMessage(conn, response, 5*time.Second); err != nil {
			log.Println("Write error:", err)
		}
	}
}

func (t *Tracker) handleList(conn net.Conn) {
	t.filesLock.RLock()
	defer t.filesLock.RUnlock()

	fileMap := make(map[string][]string)
	for file, peers := range t.files {
		for peer := range peers {
			fileMap[file] = append(fileMap[file], peer)
		}
	}

	response := &common.Message{
		Status:  "ok",
		FileMap: fileMap,
	}
	if err := common.WriteMessage(conn, response, 10*time.Second); err != nil {
		log.Println("Write error:", err)
	}
}

func (t *Tracker) handleGetPeers(conn net.Conn, msg *common.Message) {
	t.filesLock.RLock()
	defer t.filesLock.RUnlock()

	peers := []string{}
	if filePeers, exists := t.files[msg.Filename]; exists {
		for peer := range filePeers {
			peers = append(peers, peer)
		}
	}

	response := &common.Message{
		Status: "ok",
		Peers:  peers,
	}
	if err := common.WriteMessage(conn, response, 5*time.Second); err != nil {
		log.Println("Write error:", err)
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: tracker <port>")
	}
	port := os.Args[1]
	tracker := NewTracker()
	tracker.Start(port)
}
