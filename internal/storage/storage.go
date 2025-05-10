package storage

import (
	"sync"
	"time"
)

// Message holds the encrypted payload and when it was stored.
type Message struct {
	ReceiverID string
	Blob       []byte
	Timestamp  time.Time
}

var (
	lock     sync.Mutex
	messages = make(map[string][]Message)
)

// Store appends a message for receiverID.
func Store(receiverID string, blob []byte) {
	lock.Lock()
	defer lock.Unlock()
	messages[receiverID] = append(messages[receiverID], Message{
		ReceiverID: receiverID,
		Blob:       blob,
		Timestamp:  time.Now(),
	})
}

// Fetch returns and clears all msgs for receiverID.
func Fetch(receiverID string) [][]byte {
	lock.Lock()
	defer lock.Unlock()
	list := messages[receiverID]
	delete(messages, receiverID)
	blobs := make([][]byte, len(list))
	for i, m := range list {
		blobs[i] = m.Blob
	}
	return blobs
}
