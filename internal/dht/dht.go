// dht.go
package dht

import (
	"context"

	dual "github.com/libp2p/go-libp2p-kad-dht/dual"
	"github.com/libp2p/go-libp2p/core/host"
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

func (d *DHT) Close() error {
	return d.impl.Close()
}
