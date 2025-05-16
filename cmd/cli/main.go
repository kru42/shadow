package main

import (
	"fmt"
	"log"
	"time"

	"shadow/internal/identity"
)

func main() {
	// Load or create identity
	id, err := identity.LoadOrCreate("./data", "testuser")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Identity loaded:", id.DisplayName())

	// Simulate some work
	time.Sleep(2 * time.Second)

	fmt.Println("Done.")
}
