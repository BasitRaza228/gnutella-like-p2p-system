package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"peer-to-peer/common"
)

func (p *Peer) getLocalFiles() ([]string, error) {
	files := []string{}
	entries, err := os.ReadDir(p.SharedDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
			p.FilesLock.Lock()
			p.KnownFiles[entry.Name()] = true
			p.FilesLock.Unlock()
		}
	}
	return files, nil
}

func (p *Peer) sendHeartbeat() error {
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
		Command: common.HeartbeatCmd,
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
		return fmt.Errorf("heartbeat failed: %s", response.Status)
	}
	return nil
}

func (p *Peer) ListFiles() (map[string][]string, error) {
	conn, err := net.Dial("tcp", p.TrackerAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	msg := &common.Message{Command: common.ListCmd}
	if err := common.WriteMessage(conn, msg, 5*time.Second); err != nil {
		return nil, err
	}

	response, err := common.ReadMessage(conn, 10*time.Second)
	if err != nil {
		return nil, err
	}

	if response.Status != "ok" {
		return nil, fmt.Errorf("list failed: %s", response.Status)
	}
	return response.FileMap, nil
}

func (p *Peer) DownloadFile(filename, outputDir string) error {
	conn, err := net.Dial("tcp", p.TrackerAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	msg := &common.Message{
		Command:  common.GetPeersCmd,
		Filename: filename,
	}
	if err := common.WriteMessage(conn, msg, 5*time.Second); err != nil {
		return err
	}

	response, err := common.ReadMessage(conn, 5*time.Second)
	if err != nil {
		return err
	}

	if response.Status != "ok" || len(response.Peers) == 0 {
		return fmt.Errorf("no peers available for file")
	}

	// Update the active peers
	p.PeersLock.Lock()
	for _, peer := range response.Peers {
		p.ActivePeers[peer] = time.Now()
	}
	p.PeersLock.Unlock()

	// Try each peer until successful
	var lastError error
	for _, peerAddr := range response.Peers {
		if err := p.downloadFromPeer(peerAddr, filename, outputDir); err == nil {
			return nil
		} else {
			lastError = err
			log.Printf("Download from %s failed: %v", peerAddr, err)
		}
	}
	return fmt.Errorf("all download attempts failed. Last error: %v", lastError)
}

func (p *Peer) downloadFromPeer(peerAddr, filename, outputDir string) error {
	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	msg := &common.Message{
		Command:  common.DownloadCmd,
		Filename: filename,
	}
	if err := common.WriteMessage(conn, msg, 30*time.Second); err != nil {
		return err
	}

	response, err := common.ReadMessage(conn, 30*time.Second)
	if err != nil {
		return err
	}

	if response.Status != "ok" {
		return fmt.Errorf("peer error: %s", response.Status)
	}

	outputPath := filepath.Join(outputDir, filename)
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := common.ReceiveFile(conn, file, response.Size); err != nil {
		os.Remove(outputPath)
		return err
	}

	// Add to known files
	p.FilesLock.Lock()
	p.KnownFiles[filename] = true
	p.FilesLock.Unlock()

	log.Printf("Downloaded %s (%d bytes) from %s", filename, response.Size, peerAddr)
	return nil
}

func (p *Peer) CLI() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Peer-to-Peer File Sharing System")
	fmt.Println("Commands: list, download <filename>, exit")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.Fields(scanner.Text())
		if len(input) == 0 {
			continue
		}

		switch input[0] {
		case "list":
			if files, err := p.ListFiles(); err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Available files:")
				for file, peers := range files {
					fmt.Printf("  %s (%d peers)\n", file, len(peers))
				}
			}

		case "download":
			if len(input) < 2 {
				fmt.Println("Usage: download <filename>")
				continue
			}
			filename := input[1]
			fmt.Printf("Downloading %s...\n", filename)
			if err := p.DownloadFile(filename, p.SharedDir); err != nil {
				fmt.Println("Download failed:", err)
			} else {
				fmt.Println("Download successful!")
			}

		case "exit":
			fmt.Println("Exiting...")
			return

		default:
			fmt.Println("Unknown command")
		}
	}
}
