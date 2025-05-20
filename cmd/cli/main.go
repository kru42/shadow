package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	dhtdisc "github.com/libp2p/go-libp2p/p2p/discovery/routing"

	"shadow/internal/identity"
	"shadow/internal/node"
	"shadow/internal/pubsub"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	relayAddrStr := flag.String("relay", "", "Multiaddr of static relay")
	name := flag.String("name", "anon", "Identity name")
	peerAddrStr := flag.String("peer", "", "Multiaddr of another peer to connect to")
	peerIDStr := flag.String("peerid", "", "Connect to peer by ID using DHT")
	flag.Parse()

	fmt.Println("Name:", *name)
	// Load or generate identity
	id, err := identity.LoadOrCreate("data/"+*name, *name)
	if err != nil {
		panic(err)
	}

	// Graceful shutdown on Ctrl+C
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		fmt.Println("shutting down...")
		cancel()
	}()

	// Init node
	n, err := node.NewNode(ctx, id, *relayAddrStr)
	if err != nil {
		panic(err)
	}
	defer n.Shutdown(ctx)

	n.PrintInfo()

	// You can announce yourself by putting your peer ID into the DHT
	go func() {
		disc := dhtdisc.NewRoutingDiscovery(n.DHT)
		if _, err := disc.Advertise(ctx, n.Host.ID().String()); err != nil {
			fmt.Println("Failed to advertise:", err)
		} else {
			fmt.Println("Advertised peer ID in DHT:", n.Host.ID())
		}
	}()

	if *peerIDStr != "" {
		pid, err := peer.Decode(*peerIDStr)
		if err != nil {
			fmt.Println("Invalid peer ID:", err)
		} else {
			ctx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()

			// Search for peer in DHT
			pi, err := n.DHT.FindPeer(ctx, pid)
			if err != nil {
				fmt.Println("Peer not found:", err)
			} else {
				if err := n.Host.Connect(ctx, pi); err != nil {
					fmt.Println("Failed to connect to peer:", err)
				} else {
					fmt.Println("Connected to peer:", pi.ID)
				}
			}
		}
	}

	// Connect to another peer if provided
	if *peerAddrStr != "" {
		ai, err := peer.AddrInfoFromString(*peerAddrStr)
		if err != nil {
			fmt.Println("Invalid peer address:", err)
		} else {
			if err := n.Host.Connect(ctx, *ai); err != nil {
				fmt.Println("Failed to connect to peer:", err)
			} else {
				fmt.Println("Connected to peer:", ai.ID)
			}
		}
	}

	// Init pubsub
	ps, err := pubsub.NewPubSub(ctx, n.Host)
	if err != nil {
		panic(err)
	}
	topic, sub, err := ps.JoinTopic("chat")
	if err != nil {
		panic(err)
	}

	fmt.Println("Joined pubsub topic: chat")
	fmt.Println("Type messages to send:")

	// Reader for sending messages
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			msg := scanner.Text()
			if msg == "" {
				continue
			}
			err := topic.Publish(ctx, []byte(fmt.Sprintf("%s: %s", *name, msg)))
			if err != nil {
				fmt.Println("Failed to publish:", err)
			}
		}
	}()

	// Reader for receiving messages
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			break
		}
		fmt.Printf("[msg] %s\n", string(msg.Data))
	}
}
