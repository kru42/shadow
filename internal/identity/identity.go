package identity

import (
	"fmt"

	crypto "github.com/libp2p/go-libp2p/core/crypto"
	peer "github.com/libp2p/go-libp2p/core/peer"
	zbase32 "github.com/tv42/zbase32"
)

// Identity holds persistent identity info for a node
type Identity struct {
	privKey  crypto.PrivKey
	pubKey   crypto.PubKey
	peerID   peer.ID
	username string
}

// New creates a new Identity from key and username
func New(priv crypto.PrivKey, username string) (*Identity, error) {
	fmt.Printf("Creating new identity for %s...\n", username)
	peerID, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	return &Identity{
		privKey:  priv,
		pubKey:   priv.GetPublic(),
		peerID:   peerID,
		username: username,
	}, nil
}

func (id *Identity) Username() string {
	return id.username
}

func (id *Identity) PeerID() peer.ID {
	return id.peerID
}

func (id *Identity) PublicKey() crypto.PubKey {
	return id.pubKey
}

func (id *Identity) PrivateKey() crypto.PrivKey {
	return id.privKey
}

func (id *Identity) DisplayName() string {
	return fmt.Sprintf("%s@%s", id.username, encodeID(id.peerID))
}

func encodeID(id peer.ID) string {
	// Encode the ID using zbase32
	encoded := make([]byte, zbase32.EncodedLen(len(id)))
	written := zbase32.Encode(encoded, []byte(id))
	return string(encoded[:written])
}
