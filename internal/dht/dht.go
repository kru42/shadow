// dht.go
package dht

import (
	"context"

	"github.com/ipfs/go-cid"
	dual "github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type DHT struct {
	impl *dual.DHT
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

func (d *DHT) FindPeer(ctx context.Context, id peer.ID) (peerID peer.AddrInfo, err error) {
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
