package library

import "github.com/google/uuid"

// newBookID returns a random UUID suitable for a managed folder name.
func newBookID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	return id.String(), nil
}
