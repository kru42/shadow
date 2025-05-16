package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"shadow/internal/identity"
	"shadow/internal/node"
	"shadow/internal/pubsub"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on Ctrl+C
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		fmt.Println("shutting down...")
		cancel()
	}()

	// Load or generate identity
	id, err := identity.LoadOrCreate("id.key", "testuser")
	if err != nil {
		panic(err)
	}

	// Init node
	n, err := node.NewNode(ctx, id)
	if err != nil {
		panic(err)
	}
	defer n.Shutdown(ctx)

	n.PrintInfo()

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
			err := topic.Publish(ctx, []byte(msg))
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
		fmt.Printf("[msg] %s: %s\n", msg.GetFrom().Loggable(), string(msg.Data))
	}
}
