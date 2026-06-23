package library

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// libraryFile is the on-disk JSON shape for the local metadata file.
type libraryFile struct {
	Version int            `json:"version"`
	Books   []BookMetadata `json:"books"`
}

// readLibrary loads and normalizes the JSON metadata file.
func (s *Store) readLibrary() (libraryFile, error) {
	data, err := os.ReadFile(s.libraryPath)
	if err != nil {
		return libraryFile{}, fmt.Errorf("read library metadata: %w", err)
	}

	var file libraryFile
	if err := json.Unmarshal(data, &file); err != nil {
		return libraryFile{}, fmt.Errorf("decode library metadata: %w", err)
	}
	if file.Version == 0 {
		file.Version = libraryVersion
	}
	if file.Books == nil {
		file.Books = []BookMetadata{}
	}
	for i := range file.Books {
		file.Books[i].Appearance = normalizeReaderAppearance(file.Books[i].Appearance)
		file.Books[i].Prompt = normalizePromptConfig(file.Books[i].Prompt, file.Books[i].Prompt.UpdatedAt)
		file.Books[i].Notes = normalizeReaderNotes(file.Books[i].Notes)
	}

	return file, nil
}

// writeLibrary persists a normalized metadata file.
func (s *Store) writeLibrary(file libraryFile) error {
	file.Version = libraryVersion
	if file.Books == nil {
		file.Books = []BookMetadata{}
	}
	if err := writeJSONAtomic(s.libraryPath, file); err != nil {
		return fmt.Errorf("write library metadata: %w", err)
	}

	return nil
}

// writeJSONAtomic writes JSON through a temp file before replacing the target.
func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".library-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp metadata file: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return fmt.Errorf("write temp metadata file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temp metadata file: %w", err)
	}
	// Rename keeps readers from observing partially written JSON.
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("replace metadata file: %w", err)
	}

	return nil
}
