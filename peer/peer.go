package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"peer-to-peer/common"
)

type Peer struct {
	ID          string
	TrackerAddr string
	SharedDir   string
	ServerPort  string
	KnownFiles  map[string]bool
	ActivePeers map[string]time.Time
	FilesLock   sync.RWMutex
	PeersLock   sync.RWMutex
}

func NewPeer(trackerAddr, serverPort, sharedDir string) (*Peer, error) {
	ip, err := common.ResolveLocalIP()
	if err != nil {
		return nil, err
	}

	return &Peer{
		ID:          fmt.Sprintf("%s:%s", ip, serverPort),
		TrackerAddr: trackerAddr,
		SharedDir:   sharedDir,
		ServerPort:  serverPort,
		KnownFiles:  make(map[string]bool),
		ActivePeers: make(map[string]time.Time),
	}, nil
}

func (p *Peer) Start() {
	go p.startServer()
	go p.heartbeatRoutine()
	go p.peerDiscoveryRoutine()
	p.CLI()
}

func (p *Peer) heartbeatRoutine() {
	// Initial registration
	if err := p.registerWithTracker(); err != nil {
		log.Printf("Initial registration failed: %v", err)
	} else {
		log.Println("Registered with tracker")
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := p.sendHeartbeat(); err != nil {
			log.Printf("Heartbeat failed: %v", err)
		} else {
			log.Println("Heartbeat sent")
		}
	}
}

func (p *Peer) peerDiscoveryRoutine() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.PeersLock.Lock()
		now := time.Now()
		for peer, lastSeen := range p.ActivePeers {
			if now.Sub(lastSeen) > 5*time.Minute {
				delete(p.ActivePeers, peer)
				log.Printf("Removed inactive peer: %s", peer)
			}
		}
		p.PeersLock.Unlock()
	}
}

func (p *Peer) registerWithTracker() error {
	conn, err := net.Dial("tcp", p.TrackerAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	files, err := p.getLocalFiles()
	if err != nil {
		return err
	}

	msg := &common.Message{
		Command: common.RegisterCmd,
		Address: p.ID,
		Files:   files,
	}

	if err := common.WriteMessage(conn, msg, 5*time.Second); err != nil {
		return err
	}

	response, err := common.ReadMessage(conn, 5*time.Second)
	if err != nil {
		return err
	}

	if response.Status != "ok" {
		return fmt.Errorf("registration failed: %s", response.Status)
	}
	return nil
}
func main() {
	if len(os.Args) < 4 {
		log.Fatal("Usage: peer <tracker_addr> <server_port> <shared_dir>")
	}
	trackerAddr := os.Args[1]
	serverPort := os.Args[2]
	sharedDir := os.Args[3]

	// Create shared directory if not exists
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		log.Fatal(err)
	}

	peer, err := NewPeer(trackerAddr, serverPort, sharedDir)
	if err != nil {
		log.Fatal(err)
	}
	peer.Start()
}
