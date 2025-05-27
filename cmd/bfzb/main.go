package main

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tv42/zbase32"
)

const (
	prefix      = "p"
	numWorkers  = 48        // Ryzen 9 9900X
	batchSize   = 4096      // Each worker tests this many inputs per loop
	reportEvery = 1_000_000 // Print progress every N tries (optional)
)

func main() {
	runtime.GOMAXPROCS(numWorkers)

	var counter uint64
	var found int32
	var wg sync.WaitGroup

	start := time.Now()
	fmt.Printf("Brute-forcing for prefix \"%s\" using %d goroutines...\n", prefix, numWorkers)

	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			buf := make([]byte, 8)

			for atomic.LoadInt32(&found) == 0 {
				base := atomic.AddUint64(&counter, batchSize) - batchSize
				for j := uint64(0); j < batchSize; j++ {
					id := base + j
					binary.BigEndian.PutUint64(buf, id)
					encoded := zbase32.EncodeToString(buf)

					if strings.HasPrefix(encoded, prefix) {
						if atomic.CompareAndSwapInt32(&found, 0, 1) {
							elapsed := time.Since(start)
							fmt.Printf("✅ Found!\nInput: %x\nEncoded: %s\nTried: %d\nTime: %s\n",
								buf, encoded, id, elapsed)
						}
						return
					}

					if id%reportEvery == 0 && i == 0 { // Only one goroutine reports
						fmt.Printf("Checked %d inputs...\n", id)
					}
				}
			}
		}()
	}

	wg.Wait()
}
