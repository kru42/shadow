// node.go
package node

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	ma "github.com/multiformats/go-multiaddr"

	"shadow/internal/dht"
	"shadow/internal/identity"
)

type Node struct {
	Host     host.Host
	DHT      *dht.DHT
	Identity *identity.Identity
	PubSub   *pubsub.PubSub
}

func NewNode(ctx context.Context, id *identity.Identity) (*Node, error) {
	// Define static relay multiaddrs
	relayAddrs := []string{
		"/ip4/1.2.3.4/tcp/4001/p2p/QmRelayPeerID1",
		"/ip4/5.6.7.8/tcp/4001/p2p/QmRelayPeerID2",
	}

	var staticRelays []peer.AddrInfo
	for _, addrStr := range relayAddrs {
		maddr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return nil, fmt.Errorf("invalid multiaddr: %w", err)
		}
		ai, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			return nil, fmt.Errorf("invalid AddrInfo: %w", err)
		}
		staticRelays = append(staticRelays, *ai)
	}

	h, err := libp2p.New(
		libp2p.Identity(id.PrivateKey()),
		libp2p.EnableAutoRelayWithStaticRelays(staticRelays),
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
