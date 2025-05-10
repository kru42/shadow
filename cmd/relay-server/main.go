package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kru42/shadow/internal/storage"
)

// Peers is a list of other relay addresses (host:port) to mesh with.
var Peers []string

func init() {
	// Read peers from env RELAY_PEERS (comma-separated), e.g:
	// RELAY_PEERS=localhost:9998,localhost:9999
	if s := os.Getenv("RELAY_PEERS"); s != "" {
		Peers = strings.Split(s, ",")
	}
}

func main() {
	addr := ":9999"
	if a := os.Getenv("RELAY_ADDR"); a != "" {
		addr = a
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}
	log.Printf("Relay listening on %s; peers=%v\n", addr, Peers)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("accept:", err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)

	opLine, err := r.ReadString('\n')
	if err != nil {
		return
	}
	op := strings.TrimSpace(opLine)

	idLine, err := r.ReadString('\n')
	if err != nil {
		return
	}
	receiverID := strings.TrimSpace(idLine)

	switch op {
	case "SEND":
		blobLine, err := r.ReadBytes('\n')
		if err != nil {
			return
		}
		blob := blobLine[:len(blobLine)-1] // drop newline

		// store locally
		storage.Store(receiverID, blob)

		// forward to peers concurrently
		var wg sync.WaitGroup
		for _, peer := range Peers {
			wg.Add(1)
			go func(p string) {
				defer wg.Done()
				forward(p, receiverID, blob)
			}(peer)
		}
		wg.Wait()

		conn.Write([]byte("OK\n"))

	case "FETCH":
		blobs := storage.Fetch(receiverID)
		for _, b := range blobs {
			conn.Write(b)
			conn.Write([]byte("\n"))
		}
		conn.Write([]byte("END\n"))

	default:
		conn.Write([]byte("ERR unknown op\n"))
	}
}

// forward dials a peer and re-issues the same SEND.
func forward(peerAddr, receiverID string, blob []byte) {
	c, err := net.DialTimeout("tcp", peerAddr, 2*time.Second)
	if err != nil {
		return
	}
	defer c.Close()
	fmt.Fprintf(c, "SEND\n%s\n%s\n", receiverID, string(blob))
}
