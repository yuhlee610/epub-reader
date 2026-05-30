package main

import (
	"archive/zip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"epub-reader/internal/library"
)

// TestImportEPUBImportsSelectedFile verifies the Wails-facing import method
// copies a picker-selected EPUB into managed storage and returns metadata.
func TestImportEPUBImportsSelectedFile(t *testing.T) {
	source := writeAppTestEPUB(t, t.TempDir(), "cradle.epub", "Cradle", "Will Wight")
	app := newTestApp(t, func(context.Context) (string, error) {
		return source, nil
	})

	result, err := app.ImportEPUB()
	if err != nil {
		t.Fatalf("ImportEPUB() error = %v", err)
	}
	if result.Canceled {
		t.Fatal("Canceled = true, want false")
	}
	if result.Duplicate {
		t.Fatal("Duplicate = true, want false for first import")
	}
	if result.Book.Title != "Cradle" || result.Book.Author != "Will Wight" {
		t.Fatalf("Book = %#v, want EPUB metadata", result.Book)
	}
	if _, err := os.Stat(result.Book.FilePath); err != nil {
		t.Fatalf("managed copy missing: %v", err)
	}
}

// TestImportEPUBHandlesDuplicateSelection verifies repeated picker selections
// are deterministic and do not add another metadata record.
func TestImportEPUBHandlesDuplicateSelection(t *testing.T) {
	source := writeAppTestEPUB(t, t.TempDir(), "cradle.epub", "Cradle", "Will Wight")
	app := newTestApp(t, func(context.Context) (string, error) {
		return source, nil
	})

	first, err := app.ImportEPUB()
	if err != nil {
		t.Fatalf("first ImportEPUB() error = %v", err)
	}
	second, err := app.ImportEPUB()
	if err != nil {
		t.Fatalf("second ImportEPUB() error = %v", err)
	}
	if !second.Duplicate {
		t.Fatal("Duplicate = false, want true for repeated selection")
	}
	if second.Book.ID != first.Book.ID {
		t.Fatalf("second book ID = %q, want %q", second.Book.ID, first.Book.ID)
	}

	books, err := app.ListBooks()
	if err != nil {
		t.Fatalf("ListBooks() error = %v", err)
	}
	if len(books) != 1 {
		t.Fatalf("book count = %d, want 1", len(books))
	}
}

// TestImportEPUBHandlesCanceledPicker verifies cancellation is represented as
// a non-error result for the frontend.
func TestImportEPUBHandlesCanceledPicker(t *testing.T) {
	app := newTestApp(t, func(context.Context) (string, error) {
		return "", nil
	})

	result, err := app.ImportEPUB()
	if err != nil {
		t.Fatalf("ImportEPUB() error = %v, want nil on cancellation", err)
	}
	if !result.Canceled {
		t.Fatal("Canceled = false, want true")
	}
}

// TestImportEPUBRejectsInvalidSelection verifies invalid picker selections
// return the library validation error instead of creating metadata.
func TestImportEPUBRejectsInvalidSelection(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "bad.epub")
	if err := os.WriteFile(source, []byte("not a zip"), 0o644); err != nil {
		t.Fatalf("write invalid fixture: %v", err)
	}
	app := newTestApp(t, func(context.Context) (string, error) {
		return source, nil
	})

	if _, err := app.ImportEPUB(); !errors.Is(err, library.ErrInvalidEPUB) {
		t.Fatalf("ImportEPUB() error = %v, want ErrInvalidEPUB", err)
	}

	books, err := app.ListBooks()
	if err != nil {
		t.Fatalf("ListBooks() error = %v", err)
	}
	if len(books) != 0 {
		t.Fatalf("book count = %d, want 0 after invalid selection", len(books))
	}
}

// newTestApp creates an App with temp storage and a controlled picker result.
func newTestApp(t *testing.T, openEPUBFile func(context.Context) (string, error)) *App {
	t.Helper()

	store, err := library.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	return &App{
		ctx:          context.Background(),
		store:        store,
		openEPUBFile: openEPUBFile,
	}
}

// writeAppTestEPUB creates a minimal valid EPUB fixture for app-level tests.
func writeAppTestEPUB(t *testing.T, dir, name, title, author string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create epub fixture: %v", err)
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	mustWriteAppZipFile(t, archive, "mimetype", "application/epub+zip")
	mustWriteAppZipFile(t, archive, "META-INF/container.xml", `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OPS/package.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)
	mustWriteAppZipFile(t, archive, "OPS/package.opf", `<?xml version="1.0"?>
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

// mustWriteAppZipFile writes one fixture entry into the app test EPUB archive.
func mustWriteAppZipFile(t *testing.T, archive *zip.Writer, name, body string) {
	t.Helper()

	writer, err := archive.Create(name)
	if err != nil {
		t.Fatalf("create zip entry %q: %v", name, err)
	}
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatalf("write zip entry %q: %v", name, err)
	}
}
