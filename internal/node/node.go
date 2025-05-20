// node.go
package node

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	dhtdisc "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"

	"shadow/internal/dht"
	"shadow/internal/identity"
)

type Node struct {
	Host     host.Host
	DHT      *dht.DHT
	Identity *identity.Identity
	PubSub   *pubsub.PubSub
}

func NewNode(ctx context.Context, id *identity.Identity, relayAddr string) (*Node, error) {
	if relayAddr == "" {
		return nil, fmt.Errorf("relay address is required")
	}

	ai, err := peer.AddrInfoFromString(relayAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse relay address: %w", err)
	}

	h, err := libp2p.New(
		libp2p.Identity(id.PrivateKey()),
		libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{*ai}),
		libp2p.NATPortMap(),
		libp2p.EnableNATService(),
		libp2p.EnableRelay(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create host: %w", err)
	}

	dhtInstance, err := dht.NewDHT(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("failed to init DHT: %w", err)
	}

	identify.NewIDService(h)
	_ = ping.NewPingService(h)

	pubsubInstance, err := pubsub.NewPubSub(ctx, h, pubsub.DefaultGossipSubRouter(h), pubsub.WithMessageSigning(true))
	if err != nil {
		return nil, fmt.Errorf("failed to init pubsub: %w", err)
	}

	if err := h.Connect(ctx, *ai); err != nil {
		return nil, fmt.Errorf("failed to connect to relay: %w", err)
	}

	// Connect to bootstrap peers
	peers, err := dht.BootstrapPeers()
	if err != nil {
		return nil, fmt.Errorf("failed to get bootstrap peers: %w", err)
	}
	for _, peerInfo := range peers {
		if err := h.Connect(ctx, peerInfo); err != nil {
			fmt.Println("Failed to connect to bootstrap peer:", err)
		}
	}

	// Explicitly bootstrap the DHT after connecting to peers
	if err := dhtInstance.Bootstrap(ctx); err != nil {
		return nil, fmt.Errorf("DHT bootstrap failed: %w", err)
	}

	// Advertise the peer ID in the DHT, after a short delay
	go func() {
		time.Sleep(5 * time.Second) // Give DHT time to fill buckets
		disc := dhtdisc.NewRoutingDiscovery(dhtInstance)
		fmt.Println("Advertising peer ID " + id.PeerID().String() + " in DHT")
		if _, err := disc.Advertise(ctx, id.PeerID().String()); err != nil {
			fmt.Println("Failed to advertise:", err)
		} else {
			fmt.Println("Advertised peer ID in DHT:", id.PeerID())
		}
	}()

	return &Node{
		Host:     h,
		DHT:      dhtInstance,
		Identity: id,
		PubSub:   pubsubInstance,
	}, nil
}

func (n *Node) PrintInfo() {
	fmt.Println("Peer ID:", n.Identity.DisplayName())
	for _, addr := range n.Host.Addrs() {
		fmt.Printf("- %s/p2p/%s\n", addr, n.Host.ID())
	}
}

func (n *Node) Shutdown(ctx context.Context) error {
	if err := n.DHT.Close(); err != nil {
		return err
	}
	return n.Host.Close()
}
