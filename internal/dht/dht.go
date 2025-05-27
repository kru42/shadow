// dht.go
package dht

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	dual "github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

type DHT struct {
	impl *dual.DHT
}

// DefaultBootstrapPeers are the default bootstrap peers for the DHT.
var DefaultBootstrapPeers = []string{
	"/ip4/127.0.0.1/tcp/59810/p2p/12D3KooWSAHX3PDuFo5BKpHGkU8jpPoFrYHLACnPKB13ePKe2Rjj", // bob
	"/ip4/127.0.0.1/tcp/59848/p2p/12D3KooWCdnSstPmm2hb2DYLUgfYfbNpgucHRAzB52fo5q4hZn1A", // alice
}

func NewDHT(ctx context.Context, h host.Host) (*DHT, error) {
	dht, err := dual.New(ctx, h)
	if err != nil {
		return nil, err
	}
	if err := dht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	return &DHT{impl: dht}, nil
}

func (d *DHT) Bootstrap(ctx context.Context) error {
	if d.impl == nil {
		return fmt.Errorf("DHT not initialized")
	}
	if err := d.impl.Bootstrap(ctx); err != nil {
		return err
	}
	return nil
}

func (d *DHT) FindPeer(ctx context.Context, id peer.ID) (peerID peer.AddrInfo, err error) {
	fmt.Println("Finding peer:", id)
	if d.impl == nil {
		return peer.AddrInfo{}, fmt.Errorf("DHT not initialized")
	}
	// Use the underlying libp2p DHT to find the peer
	peerAi, err := d.impl.FindPeer(ctx, id)
	if err != nil {
		return peer.AddrInfo{}, err
	}
	return peerAi, nil
}

func (d *DHT) FindProvidersAsync(ctx context.Context, c cid.Cid, count int) <-chan peer.AddrInfo {
	// Forward to the underlying libp2p DHT if you have one, or implement accordingly
	if d.impl != nil {
		return d.impl.FindProvidersAsync(ctx, c, count)
	}
	ch := make(chan peer.AddrInfo)
	close(ch)
	return ch
}

func (d *DHT) Provide(ctx context.Context, c cid.Cid, recursive bool) error {
	if d.impl != nil {
		return d.impl.Provide(ctx, c, recursive)
	}
	return nil
}

func (d *DHT) Close() error {
	return d.impl.Close()
}

func BootstrapPeers(relayId peer.ID) ([]peer.AddrInfo, error) {
	// Add the relay ID to the list of bootstrap peers
	relayCid := peer.ToCid(relayId)
	relayAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/4001/p2p/%s", relayCid.String())
	_, err := peer.AddrInfoFromString(relayAddr)
	if err != nil {
		return nil, err
	}
	// Add the relay address to the list of bootstrap peers
	DefaultBootstrapPeers = append(DefaultBootstrapPeers, relayAddr)

	var peers []peer.AddrInfo
	for _, addr := range DefaultBootstrapPeers {
		maddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}
		ai, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			return nil, err
		}
		peers = append(peers, *ai)
	}
	return peers, nil
}
