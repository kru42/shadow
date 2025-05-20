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

	"shadow/internal/identity"
	"shadow/internal/node"
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

	// If relay address is not provided, try to load from relay.addr file
	if *relayAddrStr == "" {
		if b, err := os.ReadFile("data/relay.addr"); err == nil {
			*relayAddrStr = string(b)
			fmt.Println("Loaded relay address from data/relay.addr:", *relayAddrStr)
		}
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

	if *peerIDStr != "" {
		pid, err := peer.Decode(*peerIDStr)
		if err != nil {
			fmt.Println("Invalid peer ID:", err)
		} else {
			// Wait for DHT to populate
			time.Sleep(5 * time.Second)

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

	// Join pubsub topic
	topic, err := n.PubSub.Join("chat")
	if err != nil {
		fmt.Println("Failed to join pubsub topic:", err)
		return
	}
	sub, err := topic.Subscribe()
	if err != nil {
		fmt.Println("Failed to subscribe to pubsub topic:", err)
		return
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
