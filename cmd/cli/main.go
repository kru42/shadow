package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"shadow/internal/crypto"
	"shadow/internal/identity"
	"shadow/internal/node"
)

const privateMsgProtocol = "/chat/1.0.0"

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

	// Stream handler for private messages
	n.Host.SetStreamHandler(privateMsgProtocol, func(s network.Stream) {
		defer s.Close()
		buf := make([]byte, 4096)
		nr, err := s.Read(buf)
		if err == nil && nr > 0 {
			fmt.Printf("[private msg] %s\n", string(buf[:nr]))
		}
	})

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
	fmt.Println("Type messages to send. Commands: /peers, /help, /quit")

	// Derive a shared key for the group chat (using own priv/pub for now)
	sharedKey, err := crypto.DeriveShared(id.PrivateKey(), id.PublicKey())
	if err != nil {
		panic(err)
	}

	// REPL for chat and commands
	scanner := bufio.NewScanner(os.Stdin)
	go func() {
		for scanner.Scan() {
			msg := scanner.Text()
			if msg == "" {
				continue
			}
			switch {
			case msg == "/quit":
				fmt.Println("Exiting...")
				cancel()
				return
			case msg == "/peers":
				fmt.Println("Connected peers:")
				for _, p := range n.Host.Network().Peers() {
					addrs := n.Host.Peerstore().Addrs(p)
					fmt.Printf("- %s", p.Loggable())
					if len(addrs) > 0 {
						fmt.Printf(" (")
						for i, a := range addrs {
							if i > 0 {
								fmt.Print(", ")
							}
							fmt.Print(a.String())
						}
						fmt.Print(")")
					}
					fmt.Println()
				}
			case msg == "/help":
				fmt.Println("Available commands:")
				fmt.Println("  /peers   - List connected peers")
				fmt.Println("  /quit    - Exit the chat")
				fmt.Println("  /help    - Show this help message")
				fmt.Println("  /msg <peerid> <message> - Send a private message")
				fmt.Println("  <text>   - Send a message to the chat")
			default:
				// Handle /msg command
				if strings.HasPrefix(msg, "/msg ") {
					parts := strings.SplitN(msg, " ", 3)
					if len(parts) < 3 {
						fmt.Println("Usage: /msg <peerid> <message>")
						continue
					}
					peerIDStr := parts[1]
					privateMsg := parts[2]
					pid, err := peer.Decode(peerIDStr)
					if err != nil {
						fmt.Println("Invalid peer ID:", err)
						continue
					}
					s, err := n.Host.NewStream(ctx, pid, privateMsgProtocol)
					if err != nil {
						fmt.Println("Failed to open stream to peer:", err)
						continue
					}
					_, err = s.Write([]byte(fmt.Sprintf("[from %s] %s", *name, privateMsg)))
					if err != nil {
						fmt.Println("Failed to send message:", err)
					}
					s.Close()
					continue
				}
				// Default: send to pubsub (ENCRYPT)
				plain := fmt.Sprintf("%s: %s", *name, msg)
				enc, err := crypto.Seal(sharedKey, []byte(plain))
				if err != nil {
					fmt.Println("Encryption failed:", err)
					continue
				}
				err = topic.Publish(ctx, enc)
				if err != nil {
					fmt.Println("Failed to publish:", err)
				}
			}
		}
	}()

	// Reader for receiving messages (DECRYPT)
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			break
		}
		dec, err := crypto.Open(sharedKey, msg.Data)
		if err != nil {
			fmt.Printf("[msg] <decryption failed>\n")
			continue
		}
		fmt.Printf("[msg] %s\n", string(dec))
	}
}
