package library

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultRootDir returns the deterministic app-managed data directory.
func DefaultRootDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}

	return filepath.Join(configDir, appDirName), nil
}

// ensureLayout creates the managed storage folders and an empty metadata file
// when this is the first run.
func (s *Store) ensureLayout() error {
	if err := os.MkdirAll(s.booksDir, 0o755); err != nil {
		return fmt.Errorf("create managed book storage: %w", err)
	}

	if _, err := os.Stat(s.libraryPath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect library metadata: %w", err)
	}

	file := libraryFile{Version: libraryVersion, Books: []BookMetadata{}}
	if err := writeJSONAtomic(s.libraryPath, file); err != nil {
		return fmt.Errorf("initialize library metadata: %w", err)
	}

	return nil
}
