package library

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadReaderBookReturnsSpineChapters verifies managed EPUB content is
// loaded from the imported copy and ordered by the package spine.
func TestLoadReaderBookReturnsSpineChapters(t *testing.T) {
	store := newTestStore(t)
	source := writeReaderTestEPUB(t, t.TempDir(), "cradle.epub")

	imported, err := store.ImportEPUBFromPath(source)
	if err != nil {
		t.Fatalf("ImportEPUBFromPath() error = %v", err)
	}
	if err := os.Remove(source); err != nil {
		t.Fatalf("remove original fixture: %v", err)
	}

	readerBook, err := store.LoadReaderBook(imported.Book.ID)
	if err != nil {
		t.Fatalf("LoadReaderBook() error = %v", err)
	}
	if readerBook.Book.ID != imported.Book.ID {
		t.Fatalf("Book.ID = %q, want %q", readerBook.Book.ID, imported.Book.ID)
	}
	if len(readerBook.Chapters) != 2 {
		t.Fatalf("chapter count = %d, want 2", len(readerBook.Chapters))
	}
	if readerBook.Chapters[0].Title != "Chapter One" {
		t.Fatalf("first chapter title = %q, want Chapter One", readerBook.Chapters[0].Title)
	}
	if !strings.Contains(readerBook.Chapters[0].BodyHTML, "<p>The first lesson.</p>") {
		t.Fatalf("first chapter HTML = %q, want sanitized paragraph", readerBook.Chapters[0].BodyHTML)
	}
}

// TestLoadReaderBookUsesPersistedProgress verifies reopening starts near the
// last saved chapter.
func TestLoadReaderBookUsesPersistedProgress(t *testing.T) {
	store := newTestStore(t)
	source := writeReaderTestEPUB(t, t.TempDir(), "cradle.epub")
	imported, err := store.ImportEPUBFromPath(source)
	if err != nil {
		t.Fatalf("ImportEPUBFromPath() error = %v", err)
	}

	_, err = store.UpdateReadingProgress(imported.Book.ID, ReadingProgress{
		ChapterHref:  "OPS/chapter-2.xhtml",
		ChapterIndex: 1,
	})
	if err != nil {
		t.Fatalf("UpdateReadingProgress() error = %v", err)
	}

	readerBook, err := store.LoadReaderBook(imported.Book.ID)
	if err != nil {
		t.Fatalf("LoadReaderBook() error = %v", err)
	}
	if readerBook.CurrentChapterIndex != 1 {
		t.Fatalf("CurrentChapterIndex = %d, want 1", readerBook.CurrentChapterIndex)
	}
}

// TestLoadReaderBookReportsMalformedManagedCopy verifies reader errors are
// surfaced as invalid EPUB failures.
func TestLoadReaderBookReportsMalformedManagedCopy(t *testing.T) {
	store := newTestStore(t)
	path, err := store.ManagedBookPath("book-1", "bad.epub")
	if err != nil {
		t.Fatalf("ManagedBookPath() error = %v", err)
	}

	book, err := store.SaveBook(BookMetadata{
		ID:               "book-1",
		Title:            "Bad Book",
		OriginalFileName: "bad.epub",
		FilePath:         path,
	})
	if err != nil {
		t.Fatalf("SaveBook() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create managed dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("not an epub"), 0o644); err != nil {
		t.Fatalf("write malformed managed copy: %v", err)
	}

	if _, err := store.LoadReaderBook(book.ID); !errors.Is(err, ErrInvalidEPUB) {
		t.Fatalf("LoadReaderBook() error = %v, want ErrInvalidEPUB", err)
	}
}

func writeReaderTestEPUB(t *testing.T, dir, name string) string {
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
    <dc:title>Cradle</dc:title>
    <dc:creator>Will Wight</dc:creator>
  </metadata>
  <manifest>
    <item id="chapter-1" href="chapter-1.xhtml" media-type="application/xhtml+xml"/>
    <item id="chapter-2" href="chapter-2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chapter-1"/>
    <itemref idref="chapter-2"/>
  </spine>
</package>`)
	mustWriteZipFile(t, archive, "OPS/chapter-1.xhtml", `<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head><title>Chapter One</title></head>
  <body><h1>Chapter One</h1><p>The first lesson.</p><script>bad()</script></body>
</html>`)
	mustWriteZipFile(t, archive, "OPS/chapter-2.xhtml", `<?xml version="1.0"?>
<html xmlns="http://www.w3.org/1999/xhtml">
  <head><title>Chapter Two</title></head>
  <body><h1>Chapter Two</h1><p>The second lesson.</p></body>
</html>`)
	if err := archive.Close(); err != nil {
		t.Fatalf("close epub fixture: %v", err)
	}

	return path
}
