package main

import (
	"context"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"

	"shadow/internal/identity"
	gossip "shadow/internal/pubsub"
)

type mdnsNotifee struct{ host host.Host }

func (n *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Println("Found peer via mDNS:", pi.ID)
	n.host.Connect(context.Background(), pi)
}

func main() {
	ctx := context.Background()

	// Load or create identity
	id, err := identity.LoadOrCreate("./data", "cooluser")
	if err != nil {
		log.Fatal(err)
	}

	// Build libp2p host
	host, err := libp2p.New(
		libp2p.Identity(id.PrivateKey()),
		libp2p.EnableRelay(),
		libp2p.EnableAutoRelayWithPeerSource(nil),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Peer ID:", host.ID())

	// Set up Kademlia DHT
	kademlia, err := dht.New(ctx, host)
	if err != nil {
		log.Fatal(err)
	}
	if err := kademlia.Bootstrap(ctx); err != nil {
		log.Fatal(err)
	}

	// Optional: mDNS for local peer discovery
	service := mdns.NewMdnsService(host, "yourapp-mdns", &mdnsNotifee{host})
	if err := service.Start(); err != nil {
		log.Fatal(err)
	}
	defer service.Close()

	// Set up PubSub
	ps, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		log.Fatal(err)
	}

	// Join topic and start gossiping
	handler := func(username, peerid string) {
		fmt.Printf("Discovered: %s@%s\n", username, peerid)
	}
	go gossip.StartGossip(ctx, ps, id, handler)

	// Hold program
	select {}
}
