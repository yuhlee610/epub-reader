package library

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// Store persists local book metadata for the desktop app.
type Store struct {
	mu           sync.Mutex
	rootDir      string
	booksDir     string
	libraryPath  string
	idGenerator  func() (string, error)
	timeProvider func() time.Time
}

// NewStore creates a Store rooted at rootDir. When rootDir is empty, the
// platform user config directory is used.
func NewStore(rootDir string) (*Store, error) {
	if rootDir == "" {
		var err error
		rootDir, err = DefaultRootDir()
		if err != nil {
			return nil, err
		}
	}

	rootDir = filepath.Clean(rootDir)
	store := &Store{
		rootDir:      rootDir,
		booksDir:     filepath.Join(rootDir, booksDirName),
		libraryPath:  filepath.Join(rootDir, libraryFileName),
		idGenerator:  newBookID,
		timeProvider: func() time.Time { return time.Now().UTC() },
	}

	if err := store.ensureLayout(); err != nil {
		return nil, err
	}

	return store, nil
}

// Info returns the managed storage paths used by the store.
func (s *Store) Info() StorageInfo {
	return StorageInfo{
		RootDir:     s.rootDir,
		BooksDir:    s.booksDir,
		LibraryPath: s.libraryPath,
	}
}

// ListBooks returns all persisted book metadata sorted by import date.
func (s *Store) ListBooks() ([]BookMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return nil, err
	}

	// Clone before sorting so callers never observe mutations to the persisted
	// in-memory slice returned from readLibrary.
	books := slices.Clone(file.Books)
	slices.SortFunc(books, func(a, b BookMetadata) int {
		return a.ImportedAt.Compare(b.ImportedAt)
	})

	return books, nil
}

// GetBook returns one persisted book by ID.
func (s *Store) GetBook(id string) (BookMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return BookMetadata{}, err
	}

	for _, book := range file.Books {
		if book.ID == id {
			return book, nil
		}
	}

	return BookMetadata{}, fmt.Errorf("%w: %s", ErrBookNotFound, id)
}

// SaveBook creates or updates one book metadata record.
func (s *Store) SaveBook(book BookMetadata) (BookMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return BookMetadata{}, err
	}

	// Locate an existing record before normalization so updates can preserve
	// fields that callers commonly omit, such as importedAt or filePath.
	var existing *BookMetadata
	for i := range file.Books {
		if file.Books[i].ID == strings.TrimSpace(book.ID) {
			existing = &file.Books[i]
			break
		}
	}

	prepared, err := s.prepareBook(book, existing)
	if err != nil {
		return BookMetadata{}, err
	}

	// Replace by stable ID if the book already exists; otherwise append a new
	// record while preserving the existing file order.
	replaced := false
	for i := range file.Books {
		if file.Books[i].ID == prepared.ID {
			file.Books[i] = prepared
			replaced = true
			break
		}
	}
	if !replaced {
		file.Books = append(file.Books, prepared)
	}

	if err := s.writeLibrary(file); err != nil {
		return BookMetadata{}, err
	}

	return prepared, nil
}

// DeleteBook removes a book metadata record without deleting the managed EPUB.
func (s *Store) DeleteBook(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return err
	}

	// Reuse the backing array while filtering the deleted record. This keeps the
	// operation allocation-light without exposing the internal slice to callers.
	next := file.Books[:0]
	deleted := false
	for _, book := range file.Books {
		if book.ID == id {
			deleted = true
			continue
		}
		next = append(next, book)
	}
	if !deleted {
		return fmt.Errorf("%w: %s", ErrBookNotFound, id)
	}

	file.Books = next
	return s.writeLibrary(file)
}

// UpdateReadingProgress stores the user's last known chapter location.
func (s *Store) UpdateReadingProgress(id string, progress ReadingProgress) (BookMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := s.readLibrary()
	if err != nil {
		return BookMetadata{}, err
	}

	for i := range file.Books {
		if file.Books[i].ID != strings.TrimSpace(id) {
			continue
		}

		progress.ChapterHref = strings.TrimSpace(progress.ChapterHref)
		progress.Location = strings.TrimSpace(progress.Location)
		if progress.ChapterIndex < 0 {
			progress.ChapterIndex = 0
		}
		progress.UpdatedAt = s.timeProvider()

		file.Books[i].Progress = progress
		if err := s.writeLibrary(file); err != nil {
			return BookMetadata{}, err
		}

		return file.Books[i], nil
	}

	return BookMetadata{}, fmt.Errorf("%w: %s", ErrBookNotFound, id)
}

// prepareBook normalizes and validates metadata before it is written to disk.
func (s *Store) prepareBook(book BookMetadata, existing *BookMetadata) (BookMetadata, error) {
	var err error
	book.ID, err = cleanBookID(book.ID)
	if err != nil {
		// Empty IDs are allowed on create, but malformed explicit IDs are
		// rejected because they become directory names under managed storage.
		if book.ID != "" {
			return BookMetadata{}, err
		}
		id, idErr := s.idGenerator()
		if idErr != nil {
			return BookMetadata{}, fmt.Errorf("generate book id: %w", idErr)
		}
		book.ID, err = cleanBookID(id)
		if err != nil {
			return BookMetadata{}, err
		}
	}
	book.Title = strings.TrimSpace(book.Title)
	book.Author = strings.TrimSpace(book.Author)
	book.OriginalFileName = strings.TrimSpace(book.OriginalFileName)
	book.FilePath = filepath.Clean(strings.TrimSpace(book.FilePath))

	if book.Title == "" {
		return BookMetadata{}, fmt.Errorf("%w: title is required", ErrInvalidBook)
	}
	// Updates may intentionally only send changed fields. Preserve immutable
	// import details when the caller omits them.
	if existing != nil && book.OriginalFileName == "" {
		book.OriginalFileName = existing.OriginalFileName
	}
	if existing != nil && (book.FilePath == "" || book.FilePath == ".") {
		book.FilePath = existing.FilePath
	}
	if book.OriginalFileName == "" {
		book.OriginalFileName = filepath.Base(book.FilePath)
	}
	if book.FilePath == "" || book.FilePath == "." {
		path, err := s.ManagedBookPath(book.ID, book.OriginalFileName)
		if err != nil {
			return BookMetadata{}, err
		}
		book.FilePath = path
	}
	if !strings.EqualFold(filepath.Ext(book.FilePath), ".epub") {
		return BookMetadata{}, fmt.Errorf("%w: managed file path must use .epub extension", ErrInvalidBook)
	}
	// Metadata must point at this book's own managed directory, not merely
	// somewhere under the global books folder.
	if err := s.requireManagedBookPath(book.ID, book.FilePath); err != nil {
		return BookMetadata{}, err
	}
	if existing != nil && book.ImportedAt.IsZero() {
		book.ImportedAt = existing.ImportedAt
	}
	if book.ImportedAt.IsZero() {
		book.ImportedAt = s.timeProvider()
	}

	return book, nil
}
