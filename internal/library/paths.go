package library

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ManagedBookPath returns the deterministic managed path for a book's EPUB file.
func (s *Store) ManagedBookPath(bookID, originalFileName string) (string, error) {
	bookID, err := cleanBookID(bookID)
	if err != nil {
		return "", err
	}

	name := safeEPUBFileName(originalFileName)
	path := filepath.Join(s.booksDir, bookID, name)
	if err := s.requireManagedBookPath(bookID, path); err != nil {
		return "", err
	}

	return path, nil
}

// requireManagedBookPath verifies path is inside books/<bookID>/.
func (s *Store) requireManagedBookPath(bookID, path string) error {
	bookID, err := cleanBookID(bookID)
	if err != nil {
		return err
	}

	bookDir := filepath.Join(s.booksDir, bookID)
	absBookDir, err := filepath.Abs(bookDir)
	if err != nil {
		return fmt.Errorf("resolve managed book storage: %w", err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve book file path: %w", err)
	}

	rel, err := filepath.Rel(absBookDir, absPath)
	if err != nil {
		return fmt.Errorf("compare managed book path: %w", err)
	}
	// filepath.Rel is the safety gate here: anything equal to "." or escaping
	// with ".." is outside the per-book managed directory.
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("%w: %s", ErrPathOutsideLibrary, path)
	}

	return nil
}

// cleanBookID keeps book IDs safe to use as a single Windows-first path segment.
func cleanBookID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("%w: book id is required", ErrInvalidBook)
	}
	if id == "." || id == ".." || strings.Trim(id, " .") != id || filepath.Base(id) != id || strings.ContainsAny(id, `<>:"/\|?*`) {
		return "", fmt.Errorf("%w: book id must be a single path segment", ErrInvalidBook)
	}

	return id, nil
}

// safeEPUBFileName converts user-provided filenames into a managed .epub name.
func safeEPUBFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == string(os.PathSeparator) || name == "" {
		name = "book.epub"
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	base = strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			return '-'
		default:
			return r
		}
	}, base)
	base = strings.Trim(base, " .")
	if base == "" {
		base = "book"
	}

	return base + ".epub"
}
