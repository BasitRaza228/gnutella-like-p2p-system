package main

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"peer-to-peer/common"
)

func (p *Peer) startServer() {
	ln, err := net.Listen("tcp", ":"+p.ServerPort)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Printf("Peer server listening on %s\n", p.ID)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		go p.handleConnection(conn)
	}
}

func (p *Peer) handleConnection(conn net.Conn) {
	defer conn.Close()

	msg, err := common.ReadMessage(conn, 30*time.Second)
	if err != nil {
		log.Println("Read error:", err)
		return
	}

	switch msg.Command {
	case common.DownloadCmd:
		p.handleDownload(conn, msg)
	default:
		log.Println("Unknown command:", msg.Command)
	}
}

func (p *Peer) handleDownload(conn net.Conn, msg *common.Message) {
	filePath := filepath.Join(p.SharedDir, msg.Filename)
	file, err := os.Open(filePath)
	if err != nil {
		response := &common.Message{Status: "file not found"}
		if err := common.WriteMessage(conn, response, 5*time.Second); err != nil {
			log.Println("Write error:", err)
		}
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		response := &common.Message{Status: "server error"}
		if err := common.WriteMessage(conn, response, 5*time.Second); err != nil {
			log.Println("Write error:", err)
		}
		return
	}

	response := &common.Message{
		Status: "ok",
		Size:   stat.Size(),
	}
	if err := common.WriteMessage(conn, response, 5*time.Second); err != nil {
		log.Println("Write error:", err)
		return
	}

	if err := common.SendFile(conn, file, stat.Size()); err != nil {
		log.Println("File send error:", err)
	} else {
		log.Printf("Sent %s (%d bytes) to %s", msg.Filename, stat.Size(), conn.RemoteAddr())
	}
}
