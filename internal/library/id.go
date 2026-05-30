package library

import (
	"crypto/rand"
	"encoding/hex"
)

// newBookID returns a random hex identifier suitable for a managed folder name.
func newBookID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes[:]), nil
}
