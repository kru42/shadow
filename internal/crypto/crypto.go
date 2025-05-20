// crypto.go
package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"io"

	"filippo.io/edwards25519"
	"github.com/libp2p/go-libp2p/core/crypto"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

// deriveShared securely derives a 32-byte AEAD key from your Ed25519 priv and their Ed25519 pub
func deriveShared(priv crypto.PrivKey, theirPub crypto.PubKey) ([]byte, error) {
	rawPriv, _ := priv.Raw() // seed || pub
	seed := rawPriv[:32]     // Ed25519 private seed
	xPriv, err := ed25519SeedToX25519Priv(seed)
	if err != nil {
		return nil, err
	}

	rawPub, _ := theirPub.Raw()
	xPub, err := ed25519PubKeyToX25519(rawPub)
	if err != nil {
		return nil, err
	}

	dh, err := curve25519.X25519(xPriv, xPub)
	if err != nil {
		return nil, err
	}

	// Derive AEAD key from shared secret using HKDF-SHA256
	h := hkdf.New(sha256.New, dh, nil, []byte("libp2p-chat-v1"))
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(h, key); err != nil {
		return nil, err
	}
	return key, nil
}

// Converts a 32-byte Ed25519 seed to a 32-byte clamped X25519 private key
func ed25519SeedToX25519Priv(seed []byte) ([]byte, error) {
	if len(seed) != 32 {
		return nil, fmt.Errorf("expected 32-byte seed, got %d", len(seed))
	}
	h := sha512.Sum512(seed)
	clamped := make([]byte, 32)
	copy(clamped, h[:32])
	clamped[0] &= 248
	clamped[31] &= 127
	clamped[31] |= 64
	return clamped, nil
}

func ed25519PubKeyToX25519(ed25519Pub []byte) ([]byte, error) {
	if len(ed25519Pub) != 32 {
		return nil, fmt.Errorf("invalid Ed25519 public key length: %d", len(ed25519Pub))
	}
	var A edwards25519.Point
	if _, err := A.SetBytes(ed25519Pub); err != nil {
		return nil, err
	}
	montX := A.BytesMontgomery()
	return montX[:], nil
}

// seal encrypts msg with a shared secret using ChaCha20-Poly1305
func seal(sharedSecret, msg []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(sharedSecret)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, chacha20poly1305.NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext := aead.Seal(nil, nonce, msg, nil)
	return append(nonce, ciphertext...), nil
}

// open decrypts the envelope using the shared secret
func open(sharedSecret, envelope []byte) ([]byte, error) {
	if len(envelope) < chacha20poly1305.NonceSize+chacha20poly1305.Overhead {
		return nil, fmt.Errorf("envelope too short")
	}
	aead, err := chacha20poly1305.New(sharedSecret)
	if err != nil {
		return nil, err
	}
	nonce, ct := envelope[:chacha20poly1305.NonceSize], envelope[chacha20poly1305.NonceSize:]
	return aead.Open(nil, nonce, ct, nil)
}

// Exported wrappers
func DeriveShared(priv crypto.PrivKey, pub crypto.PubKey) ([]byte, error) {
	return deriveShared(priv, pub)
}
func Seal(sharedSecret, msg []byte) ([]byte, error) {
	return seal(sharedSecret, msg)
}
func Open(sharedSecret, envelope []byte) ([]byte, error) {
	return open(sharedSecret, envelope)
}
