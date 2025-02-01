package common

import (
	"encoding/json"
	"io"
	"net"
	"time"
)

const (
	RegisterCmd   = "register"
	HeartbeatCmd  = "heartbeat"
	ListCmd       = "list"
	GetPeersCmd   = "getpeers"
	DownloadCmd   = "download"
	FileChunkSize = 4096
)

type Message struct {
	Command  string              `json:"command"`
	Address  string              `json:"address,omitempty"`
	Files    []string            `json:"files,omitempty"`
	FileMap  map[string][]string `json:"filemap,omitempty"`
	Filename string              `json:"filename,omitempty"`
	Peers    []string            `json:"peers,omitempty"`
	Status   string              `json:"status,omitempty"`
	Size     int64               `json:"size,omitempty"`
}

func ReadMessage(conn net.Conn, timeout time.Duration) (*Message, error) {
	conn.SetReadDeadline(time.Now().Add(timeout))
	decoder := json.NewDecoder(conn)
	var msg Message
	if err := decoder.Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func WriteMessage(conn net.Conn, msg *Message, timeout time.Duration) error {
	conn.SetWriteDeadline(time.Now().Add(timeout))
	return json.NewEncoder(conn).Encode(msg)
}

func SendFile(conn net.Conn, file io.Reader, size int64) error {
	if _, err := io.CopyN(conn, file, size); err != nil {
		return err
	}
	return nil
}

func ReceiveFile(conn net.Conn, file io.Writer, size int64) error {
	if _, err := io.CopyN(file, conn, size); err != nil {
		return err
	}
	return nil
}

func ResolveLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}
