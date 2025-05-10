package encryption

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/nacl/box"
)

// GenerateKeyPair returns pointers to a new curve25519 keypair.
func GenerateKeyPair() (pub, priv *[32]byte, err error) {
	priv, pub, err = box.GenerateKey(rand.Reader)
	return
}

// Encrypt encrypts message with recipientPub + senderPriv; prepends 24-byte nonce.
func Encrypt(message []byte, recipientPub, senderPriv *[32]byte) ([]byte, error) {
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	// Seal appends: ciphertext = nonce||box(nonce, message)
	return box.Seal(nonce[:], message, &nonce, recipientPub, senderPriv), nil
}

// Decrypt reverses Encrypt. Expects nonce||ciphertext.
func Decrypt(blob []byte, senderPub, recipientPriv *[32]byte) ([]byte, error) {
	if len(blob) < 24 {
		return nil, fmt.Errorf("blob too small")
	}
	var nonce [24]byte
	copy(nonce[:], blob[:24])
	plaintext, ok := box.Open(nil, blob[24:], &nonce, senderPub, recipientPriv)
	if !ok {
		return nil, fmt.Errorf("decryption failed")
	}
	return plaintext, nil
}
