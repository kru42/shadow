package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	ma "github.com/multiformats/go-multiaddr"
)

func main() {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a new libp2p Host with Relay HOP enabled
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/4001"),
		libp2p.EnableNATService(),
		libp2p.NATPortMap(),
		libp2p.EnableRelayService(),
	)
	if err != nil {
		log.Fatalf("Failed to create host: %v", err)
	}

	printHostInfo(h)

	// Wait until Ctrl+C
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch

	fmt.Println("\nShutting down...")
	if err := h.Close(); err != nil {
		log.Printf("error while closing host: %s", err)
	}
}

func printHostInfo(h host.Host) {
	fmt.Println("Relay node is running.")
	fmt.Println("Peer ID:", h.ID().String())
	fmt.Println("Addresses:")

	for _, addr := range h.Addrs() {
		fullAddr := addr.Encapsulate(ma.StringCast("/p2p/" + h.ID().String()))
		fmt.Printf(" - %s\n", fullAddr)
	}

	fmt.Println("\nGive one of these to clients to use as a static relay address.")
	fmt.Println("Example:")
	if len(h.Addrs()) > 0 {
		example := h.Addrs()[0].Encapsulate(ma.StringCast("/p2p/" + h.ID().String()))
		fmt.Printf(`peer.AddrInfoFromString("%s")`, example.String())
	}
}
