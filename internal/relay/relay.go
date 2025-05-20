package relay

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
)

const keyFileName = "relay.key"

// LoadOrCreateRelayHost loads a persisted private key or creates a new one, returning a libp2p host.
func LoadOrCreateRelayHost(ctx context.Context, dataDir string, listenAddr string) (host.Host, error) {
	keyPath := filepath.Join(dataDir, keyFileName)
	var priv crypto.PrivKey
	if b, err := os.ReadFile(keyPath); err == nil {
		// decode base64
		keyBytes, err := base64.StdEncoding.DecodeString(string(b))
		if err != nil {
			return nil, fmt.Errorf("failed to decode relay key: %w", err)
		}
		priv, err = crypto.UnmarshalPrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal relay key: %w", err)
		}
	} else {
		// generate new key and persist
		var err error
		priv, _, err = crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate relay key: %w", err)
		}
		keyBytes, err := crypto.MarshalPrivateKey(priv)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal relay key: %w", err)
		}
		if err := os.MkdirAll(dataDir, 0700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(keyBytes)), 0600); err != nil {
			return nil, fmt.Errorf("failed to write relay key: %w", err)
		}
	}

	h, err := libp2p.New(
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.EnableRelay(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create relay host: %w", err)
	}
	return h, nil
}
