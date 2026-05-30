package library

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestImportEPUBFromPathCopiesAndPersistsMetadata verifies the happy path.
func TestImportEPUBFromPathCopiesAndPersistsMetadata(t *testing.T) {
	store := newTestStore(t)
	source := writeTestEPUB(t, t.TempDir(), "cradle.epub", "Cradle", "Will Wight")

	result, err := store.ImportEPUBFromPath(source)
	if err != nil {
		t.Fatalf("ImportEPUBFromPath() error = %v", err)
	}
	if result.Duplicate {
		t.Fatal("Duplicate = true, want false for first import")
	}
	if result.Book.Title != "Cradle" || result.Book.Author != "Will Wight" {
		t.Fatalf("book metadata = %#v, want title and author from EPUB", result.Book)
	}
	if _, err := os.Stat(result.Book.FilePath); err != nil {
		t.Fatalf("managed copy missing: %v", err)
	}
	if filepath.Dir(filepath.Dir(result.Book.FilePath)) != store.Info().BooksDir {
		t.Fatalf("FilePath = %q, want inside managed books dir", result.Book.FilePath)
	}

	books, err := store.ListBooks()
	if err != nil {
		t.Fatalf("ListBooks() error = %v", err)
	}
	if len(books) != 1 || books[0].ID != result.Book.ID {
		t.Fatalf("books = %#v, want imported book persisted", books)
	}
}

// TestImportEPUBFromPathHandlesDuplicate verifies same-content imports reuse metadata.
func TestImportEPUBFromPathHandlesDuplicate(t *testing.T) {
	store := newTestStore(t)
	source := writeTestEPUB(t, t.TempDir(), "cradle.epub", "Cradle", "Will Wight")

	first, err := store.ImportEPUBFromPath(source)
	if err != nil {
		t.Fatalf("first ImportEPUBFromPath() error = %v", err)
	}
	second, err := store.ImportEPUBFromPath(source)
	if err != nil {
		t.Fatalf("second ImportEPUBFromPath() error = %v", err)
	}
	if !second.Duplicate {
		t.Fatal("Duplicate = false, want true for repeated import")
	}
	if second.Book.ID != first.Book.ID || second.Book.FilePath != first.Book.FilePath {
		t.Fatalf("duplicate result = %#v, want same metadata as %#v", second.Book, first.Book)
	}

	books, err := store.ListBooks()
	if err != nil {
		t.Fatalf("ListBooks() error = %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("book count = %d, want 1 after duplicate import", len(books))
	}
}

// TestImportEPUBFromPathRestoresCorruptDuplicateCopy verifies duplicate imports
// repair a managed copy that no longer matches the source content.
func TestImportEPUBFromPathRestoresCorruptDuplicateCopy(t *testing.T) {
	store := newTestStore(t)
	source := writeTestEPUB(t, t.TempDir(), "cradle.epub", "Cradle", "Will Wight")

	first, err := store.ImportEPUBFromPath(source)
	if err != nil {
		t.Fatalf("first ImportEPUBFromPath() error = %v", err)
	}
	if err := os.WriteFile(first.Book.FilePath, []byte("corrupt"), 0o644); err != nil {
		t.Fatalf("corrupt managed copy: %v", err)
	}

	second, err := store.ImportEPUBFromPath(source)
	if err != nil {
		t.Fatalf("second ImportEPUBFromPath() error = %v", err)
	}
	if !second.Duplicate {
		t.Fatal("Duplicate = false, want true for repaired duplicate import")
	}

	sourceHash, err := fileSHA256(source)
	if err != nil {
		t.Fatalf("hash source: %v", err)
	}
	targetHash, err := fileSHA256(first.Book.FilePath)
	if err != nil {
		t.Fatalf("hash managed copy: %v", err)
	}
	if targetHash != sourceHash {
		t.Fatalf("managed copy hash = %q, want %q", targetHash, sourceHash)
	}

	books, err := store.ListBooks()
	if err != nil {
		t.Fatalf("ListBooks() error = %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("book count = %d, want 1 after duplicate repair", len(books))
	}
}

// TestImportEPUBFromPathRejectsInvalidFiles verifies extension and content checks.
func TestImportEPUBFromPathRejectsInvalidFiles(t *testing.T) {
	store := newTestStore(t)
	dir := t.TempDir()
	textFile := filepath.Join(dir, "not-epub.txt")
	if err := os.WriteFile(textFile, []byte("nope"), 0o644); err != nil {
		t.Fatalf("write text fixture: %v", err)
	}
	badEPUB := filepath.Join(dir, "bad.epub")
	if err := os.WriteFile(badEPUB, []byte("nope"), 0o644); err != nil {
		t.Fatalf("write bad epub fixture: %v", err)
	}

	if _, err := store.ImportEPUBFromPath(textFile); !errors.Is(err, ErrInvalidEPUB) {
		t.Fatalf("text import error = %v, want ErrInvalidEPUB", err)
	}
	if _, err := store.ImportEPUBFromPath(badEPUB); !errors.Is(err, ErrInvalidEPUB) {
		t.Fatalf("bad epub import error = %v, want ErrInvalidEPUB", err)
	}
}

// TestImportEPUBFromPathUsesFilenameFallbackTitle verifies metadata fallback behavior.
func TestImportEPUBFromPathUsesFilenameFallbackTitle(t *testing.T) {
	store := newTestStore(t)
	source := writeTestEPUB(t, t.TempDir(), "Unsouled.epub", "", "")

	result, err := store.ImportEPUBFromPath(source)
	if err != nil {
		t.Fatalf("ImportEPUBFromPath() error = %v", err)
	}
	if result.Book.Title != "Unsouled" {
		t.Fatalf("Title = %q, want filename fallback", result.Book.Title)
	}
}

// writeTestEPUB creates a minimal valid EPUB fixture.
func writeTestEPUB(t *testing.T, dir, name, title, author string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create epub fixture: %v", err)
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	mustWriteZipFile(t, archive, "mimetype", "application/epub+zip")
	mustWriteZipFile(t, archive, "META-INF/container.xml", `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OPS/package.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)
	mustWriteZipFile(t, archive, "OPS/package.opf", `<?xml version="1.0"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>`+title+`</dc:title>
    <dc:creator>`+author+`</dc:creator>
  </metadata>
</package>`)
	if err := archive.Close(); err != nil {
		t.Fatalf("close epub fixture: %v", err)
	}

	return path
}

// mustWriteZipFile writes one fixture file into a ZIP archive.
func mustWriteZipFile(t *testing.T, archive *zip.Writer, name, body string) {
	t.Helper()

	writer, err := archive.Create(name)
	if err != nil {
		t.Fatalf("create zip entry %q: %v", name, err)
	}
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatalf("write zip entry %q: %v", name, err)
	}
}
