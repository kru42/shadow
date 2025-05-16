package gossip

import (
	"context"
	"fmt"
	"log"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	host "github.com/libp2p/go-libp2p/core/host"
	peer "github.com/libp2p/go-libp2p/core/peer"
)

const gossipTopicName = "usernames"

// Gossip handles pubsub-based username announcements and discovery
type Gossip struct {
	Host    host.Host
	PubSub  *pubsub.PubSub
	Topic   *pubsub.Topic
	Sub     *pubsub.Subscription
	Context context.Context
	Handler func(peer.ID, string)
}

// NewGossip initializes the pubsub system for username announcements
func NewGossip(ctx context.Context, h host.Host, handler func(peer.ID, string)) (*Gossip, error) {
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}
	topic, err := ps.Join(gossipTopicName)
	if err != nil {
		return nil, err
	}
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	g := &Gossip{
		Host:    h,
		PubSub:  ps,
		Topic:   topic,
		Sub:     sub,
		Context: ctx,
		Handler: handler,
	}

	go g.readLoop()
	return g, nil
}

// AnnounceUsername publishes your username to the network
func (g *Gossip) AnnounceUsername(username string) error {
	msg := fmt.Sprintf("%s@%s", username, g.Host.ID())
	return g.Topic.Publish(g.Context, []byte(msg))
}

// readLoop processes incoming pubsub messages
func (g *Gossip) readLoop() {
	for {
		msg, err := g.Sub.Next(g.Context)
		if err != nil {
			log.Println("error reading pubsub msg:", err)
			return
		}
		if msg.ReceivedFrom == g.Host.ID() {
			continue // ignore our own messages
		}
		peerID := msg.ReceivedFrom
		username := string(msg.Data)
		g.Handler(peerID, username)
	}
}
