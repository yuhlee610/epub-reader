package library

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// newTestStore creates a store with deterministic IDs and timestamps for tests.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	store.idGenerator = func() (string, error) { return "book-1", nil }
	store.timeProvider = func() time.Time { return time.Date(2026, 5, 29, 1, 2, 3, 0, time.UTC) }

	return store
}

// TestNewStoreInitializesLayout verifies a fresh store creates the expected
// managed storage layout and starts with no books.
func TestNewStoreInitializesLayout(t *testing.T) {
	store := newTestStore(t)
	info := store.Info()

	if info.RootDir == "" {
		t.Fatal("RootDir is empty")
	}
	if info.BooksDir != filepath.Join(info.RootDir, booksDirName) {
		t.Fatalf("BooksDir = %q, want managed books dir under root", info.BooksDir)
	}
	if info.LibraryPath != filepath.Join(info.RootDir, libraryFileName) {
		t.Fatalf("LibraryPath = %q, want library file under root", info.LibraryPath)
	}

	books, err := store.ListBooks()
	if err != nil {
		t.Fatalf("ListBooks() error = %v", err)
	}
	if len(books) != 0 {
		t.Fatalf("ListBooks() length = %d, want 0", len(books))
	}
}

// TestDefaultRootDirUsesAppDirectoryName verifies the default root is stable
// and app-scoped on the current OS.
func TestDefaultRootDirUsesAppDirectoryName(t *testing.T) {
	root, err := DefaultRootDir()
	if err != nil {
		t.Fatalf("DefaultRootDir() error = %v", err)
	}
	if filepath.Base(root) != appDirName {
		t.Fatalf("DefaultRootDir() = %q, want path ending in %q", root, appDirName)
	}
	t.Logf("default storage root: %s", root)
}

// TestSaveBookCreatesAndPersistsMetadata verifies create, reload, and metadata
// persistence for progress and prompt fields.
func TestSaveBookCreatesAndPersistsMetadata(t *testing.T) {
	store := newTestStore(t)

	book, err := store.SaveBook(BookMetadata{
		Title:            "Cradle",
		Author:           "Will Wight",
		OriginalFileName: "cradle.epub",
		Progress: ReadingProgress{
			ChapterHref:  "chapter-1.xhtml",
			ChapterIndex: 1,
			Location:     "p10",
		},
		Prompt: PromptConfig{
			CustomPrompt: "Translate with context and explain grammar.",
		},
	})
	if err != nil {
		t.Fatalf("SaveBook() error = %v", err)
	}
	if book.ID != "book-1" {
		t.Fatalf("book.ID = %q, want generated id", book.ID)
	}
	if book.ImportedAt.IsZero() {
		t.Fatal("ImportedAt was not set")
	}
	if filepath.Dir(filepath.Dir(book.FilePath)) != store.Info().BooksDir {
		t.Fatalf("FilePath = %q, want inside managed books dir", book.FilePath)
	}

	reloaded, err := NewStore(store.Info().RootDir)
	if err != nil {
		t.Fatalf("reload NewStore() error = %v", err)
	}
	got, err := reloaded.GetBook("book-1")
	if err != nil {
		t.Fatalf("GetBook() error = %v", err)
	}
	if got.Title != "Cradle" || got.Prompt.CustomPrompt == "" || got.Progress.ChapterHref == "" {
		t.Fatalf("persisted book missing expected metadata: %#v", got)
	}
}

// TestSaveBookUpdatePreservesImportedAtWhenOmitted verifies partial updates do
// not accidentally rewrite import-time fields.
func TestSaveBookUpdatePreservesImportedAtWhenOmitted(t *testing.T) {
	store := newTestStore(t)

	book, err := store.SaveBook(BookMetadata{
		Title:            "Cradle",
		OriginalFileName: "cradle.epub",
	})
	if err != nil {
		t.Fatalf("initial SaveBook() error = %v", err)
	}

	updated, err := store.SaveBook(BookMetadata{
		ID:     book.ID,
		Title:  "Cradle",
		Prompt: PromptConfig{CustomPrompt: "Explain vocabulary."},
	})
	if err != nil {
		t.Fatalf("update SaveBook() error = %v", err)
	}
	if !updated.ImportedAt.Equal(book.ImportedAt) {
		t.Fatalf("ImportedAt = %s, want preserved %s", updated.ImportedAt, book.ImportedAt)
	}
	if updated.FilePath != book.FilePath {
		t.Fatalf("FilePath = %q, want preserved %q", updated.FilePath, book.FilePath)
	}
}

// TestSaveBookRejectsPathOutsideManagedStorage verifies metadata cannot point
// at a file outside the app-managed books directory.
func TestSaveBookRejectsPathOutsideManagedStorage(t *testing.T) {
	store := newTestStore(t)

	_, err := store.SaveBook(BookMetadata{
		ID:               "book-1",
		Title:            "Cradle",
		OriginalFileName: "cradle.epub",
		FilePath:         filepath.Join(t.TempDir(), "cradle.epub"),
	})
	if !errors.Is(err, ErrPathOutsideLibrary) {
		t.Fatalf("SaveBook() error = %v, want ErrPathOutsideLibrary", err)
	}
}

// TestSaveBookRejectsPathUnderDifferentBookDirectory verifies each book is
// constrained to its own managed folder.
func TestSaveBookRejectsPathUnderDifferentBookDirectory(t *testing.T) {
	store := newTestStore(t)

	_, err := store.SaveBook(BookMetadata{
		ID:               "book-2",
		Title:            "Cradle",
		OriginalFileName: "cradle.epub",
		FilePath:         filepath.Join(store.Info().BooksDir, "book-1", "cradle.epub"),
	})
	if !errors.Is(err, ErrPathOutsideLibrary) {
		t.Fatalf("SaveBook() error = %v, want ErrPathOutsideLibrary", err)
	}
}

// TestManagedBookPathSanitizesName verifies generated managed filenames ignore
// source path traversal and normalize to a .epub file.
func TestManagedBookPathSanitizesName(t *testing.T) {
	store := newTestStore(t)

	path, err := store.ManagedBookPath("book-1", `..\bad:name.txt`)
	if err != nil {
		t.Fatalf("ManagedBookPath() error = %v", err)
	}
	if filepath.Dir(filepath.Dir(path)) != store.Info().BooksDir {
		t.Fatalf("path = %q, want inside managed books dir", path)
	}
	if filepath.Ext(path) != ".epub" {
		t.Fatalf("path = %q, want .epub extension", path)
	}
	if filepath.Base(path) != "bad-name.epub" {
		t.Fatalf("path base = %q, want sanitized name", filepath.Base(path))
	}
}

// TestManagedBookPathRejectsPathLikeBookID verifies book IDs cannot smuggle path
// traversal or Windows-invalid filename characters into managed paths.
func TestManagedBookPathRejectsPathLikeBookID(t *testing.T) {
	store := newTestStore(t)

	for _, id := range []string{`..\outside`, `bad:id`, ` trailing.`} {
		_, err := store.ManagedBookPath(id, "cradle.epub")
		if !errors.Is(err, ErrInvalidBook) {
			t.Fatalf("ManagedBookPath(%q) error = %v, want ErrInvalidBook", id, err)
		}
	}
}

// TestSaveBookAcceptsUppercaseEPUBExtension verifies validation accepts common
// uppercase EPUB extensions from imported files.
func TestSaveBookAcceptsUppercaseEPUBExtension(t *testing.T) {
	store := newTestStore(t)
	path := filepath.Join(store.Info().BooksDir, "book-1", "CRADLE.EPUB")

	_, err := store.SaveBook(BookMetadata{
		ID:               "book-1",
		Title:            "Cradle",
		OriginalFileName: "CRADLE.EPUB",
		FilePath:         path,
	})
	if err != nil {
		t.Fatalf("SaveBook() error = %v", err)
	}
}
