package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/c-bata/go-prompt"
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

	var once sync.Once

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

	// Channel to signal REPL exit
	done := make(chan struct{})

	// Graceful shutdown on Ctrl+C
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		fmt.Println("shutting down...")
		once.Do(func() {
			cancel()
			close(done)
		})
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

	// Stream handler for private messages
	n.Host.SetStreamHandler(privateMsgProtocol, func(s network.Stream) {
		defer s.Close()
		buf := make([]byte, 4096)
		nr, err := s.Read(buf)
		if err == nil && nr > 0 {
			privateMsgChan <- string(buf[:nr])
		}
	})

	// TODO:
	// Aliases map: @alias -> zbase32 peer ID
	aliases := map[string]string{
		// Example: "bob": <zbase32 peer.ID value>
	}

	// Populate aliases map with @name -> zbase32 peer ID
	for _, p := range n.Host.Network().Peers() {
		if p == n.Host.ID() {
			continue
		}
		peerIDStr := identity.PeerIDToZbase32(p)
		//alias := strings.TrimPrefix(peerIDStr, "z:")
		aliases[peerIDStr] = peerIDStr
	}

	// Tab-completion function
	completer := func(d prompt.Document) []prompt.Suggest {
		text := d.TextBeforeCursor()
		cmds := []prompt.Suggest{
			{Text: "/peers", Description: "List connected peers"},
			{Text: "/quit", Description: "Exit the chat"},
			{Text: "/help", Description: "Show help"},
			{Text: "/msg", Description: "Send a private message"},
		}
		if strings.HasPrefix(text, "/msg ") {
			// Suggest aliases and peer IDs
			suggestions := []prompt.Suggest{}
			for alias := range aliases {
				suggestions = append(suggestions, prompt.Suggest{Text: "@" + alias, Description: "Alias"})
			}
			for _, p := range n.Host.Network().Peers() {
				suggestions = append(suggestions, prompt.Suggest{Text: identity.PeerIDToZbase32(p), Description: "PeerID"})
			}
			return prompt.FilterHasPrefix(suggestions, d.GetWordBeforeCursor(), true)
		}
		return prompt.FilterHasPrefix(cmds, text, true)
	}

	// Input executor
	executor := func(msg string) {
		msg = strings.TrimSpace(msg)
		if msg == "" {
			return
		}
		switch {
		case msg == "/quit":
			fmt.Println("Exiting...")
			once.Do(func() {
				cancel()
				close(done)
			})
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
			fmt.Println("  /msg <peerid|@alias> <message> - Send a private message")
			fmt.Println("  <text>   - Send a message to the chat")
		default:
			if strings.HasPrefix(msg, "/msg ") {
				parts := strings.SplitN(msg, " ", 3)
				if len(parts) < 3 {
					fmt.Println("Usage: /msg <peerid|@alias> <message>")
					return
				}
				peerIDStr := parts[1]
				privateMsg := parts[2]
				var pid peer.ID
				var err error
				if strings.HasPrefix(peerIDStr, "@") {
					alias := peerIDStr[1:]
					peerIDStrVal, ok := aliases[alias]
					if !ok {
						fmt.Println("Unknown alias:", alias)
						return
					}
					pid, err = peer.Decode(peerIDStrVal)
					if err != nil {
						fmt.Println("Invalid peer ID for alias:", err)
						return
					}
				} else if strings.HasPrefix(peerIDStr, "z:") {
					pid, err = identity.Zbase32ToPeerID(peerIDStr[2:])
				} else {
					pid, err = peer.Decode(peerIDStr)
				}
				if err != nil {
					fmt.Println("Invalid peer ID:", err)
					return
				}
				s, err := n.Host.NewStream(ctx, pid, privateMsgProtocol)
				if err != nil {
					fmt.Println("Failed to open stream to peer:", err)
					return
				}
				_, err = s.Write([]byte(fmt.Sprintf("[from %s] %s", *name, privateMsg)))
				if err != nil {
					fmt.Println("Failed to send message:", err)
				}
				s.Close()
				return
			}
			// Ignore all other input (no group chat)
		}
	}

	// Print incoming private messages in a goroutine
	go func() {
		for pm := range privateMsgChan {
			fmt.Printf("\n[private msg] %s\n> ", pm)
		}
	}()

	// Start the prompt REPL
	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix("> "),
	)
	p.Run()

	<-done // Block main until REPL signals exit
	once.Do(func() {
		cancel()
		close(done)
	})
}
