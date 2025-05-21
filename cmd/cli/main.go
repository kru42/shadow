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

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"shadow/internal/identity"
	"shadow/internal/node"
	"shadow/internal/utils"
)

const privateMsgProtocol = "/chat/1.0.0"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	relayAddrStr := flag.String("relay", "", "Multiaddr of static relay")
	name := flag.String("name", "anon", "Identity name")
	// peerAddrStr := flag.String("peer", "", "Multiaddr of another peer to connect to")
	// peerIDStr := flag.String("peerid", "", "Connect to peer by ID using DHT")
	flag.Parse()

	fmt.Println("Name:", *name)
	// Load or generate identity
	id, err := identity.LoadOrCreate("data/"+*name, *name)
	if err != nil {
		panic(err)
	}

	// Generate identicon for self
	pubBytes, _ := id.PublicKey().Raw()
	iconPath := fmt.Sprintf("data/%s_identicon.png", *name)
	if err := utils.GenerateIdenticon(pubBytes, iconPath); err == nil {
		fmt.Println("Generated identicon at:", iconPath)
	} else {
		fmt.Println("Failed to generate identicon:", err)
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
	fmt.Println("Your PeerID (zbase32):", identity.PeerIDToZbase32(n.Host.ID()))

	// Channel for incoming private messages
	privateMsgChan := make(chan string, 10)

	// Channel to signal REPL exit
	done := make(chan struct{})

	// Stream handler for private messages
	n.Host.SetStreamHandler(privateMsgProtocol, func(s network.Stream) {
		defer s.Close()
		buf := make([]byte, 4096)
		nr, err := s.Read(buf)
		if err == nil && nr > 0 {
			privateMsgChan <- string(buf[:nr])
		}
	})

	// REPL for chat and commands
	scanner := bufio.NewScanner(os.Stdin)
	go func() {
		for {
			select {
			case pm := <-privateMsgChan:
				fmt.Printf("\n[private msg] %s\n", pm)
				fmt.Print("> ")
			default:
				if scanner.Scan() {
					msg := scanner.Text()
					if msg == "" {
						continue
					}
					switch {
					case msg == "/quit":
						fmt.Println("Exiting...")
						cancel()
						close(done) // Signal main to exit
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
						// Only handle /msg command, ignore group chat
						if strings.HasPrefix(msg, "/msg ") {
							parts := strings.SplitN(msg, " ", 3)
							if len(parts) < 3 {
								fmt.Println("Usage: /msg <peerid> <message>")
								continue
							}

							// Extract peer ID and message
							peerIDStr := parts[1]
							privateMsg := parts[2]

							var pid peer.ID
							if strings.HasPrefix(peerIDStr, "z:") {
								pid, err = identity.Zbase32ToPeerID(peerIDStr[2:])
							} else {
								pid, err = peer.Decode(peerIDStr)
							}
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
						// Ignore all other input (no group chat)
					}
				} else {
					close(done) // Signal main to exit on EOF
					return
				}
			}
		}
	}()

	<-done // Block main until REPL signals exit
}
