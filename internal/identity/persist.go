package identity

import (
	"encoding/json"
	"os"
	"path/filepath"

	crypto "github.com/libp2p/go-libp2p/core/crypto"
)

const identityFile = "identity.json"

type diskIdentity struct {
	PrivKey  []byte `json:"priv_key"`
	Username string `json:"username"`
}

func LoadOrCreate(path, username string) (*Identity, error) {
	filePath := filepath.Join(path, identityFile)
	if _, err := os.Stat(filePath); err == nil {
		return loadFromDisk(filePath)
	}
	return createNew(filePath, username)
}

func loadFromDisk(path string) (*Identity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var d diskIdentity
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	priv, err := crypto.UnmarshalPrivateKey(d.PrivKey)
	if err != nil {
		return nil, err
	}
	return New(priv, d.Username)
}

func createNew(path, username string) (*Identity, error) {
	priv, _, err := crypto.GenerateEd25519Key(nil)
	if err != nil {
		return nil, err
	}
	id, err := New(priv, username)
	if err != nil {
		return nil, err
	}
	serialized, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(diskIdentity{
		PrivKey:  serialized,
		Username: username,
	})
	if err != nil {
		return nil, err
	}
	// Ensure the directory exists before writing the file
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return nil, err
	}
	return id, nil
}
